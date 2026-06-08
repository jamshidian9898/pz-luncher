// agent is the PZ platform Agent executable.
//
// Responsibilities:
//  1. Discover mods in a local directory
//  2. Push blobs to the Backend Content Store (idempotent, retried)
//  3. Publish a server manifest to the Backend (only when content changes)
//  4. Send periodic heartbeats (always, even on partial failure)
//
// Reliability model (B3):
//   - All network operations use retry.Policy with exponential backoff.
//   - Heartbeat is independent of blob/manifest sync: a backend restart
//     does not silence the agent.
//   - Manifest is published only when the set of mod SHA256s changes
//     (content diff), reducing spurious writes.
//   - Signal handling: SIGINT/SIGTERM trigger a clean shutdown.
package main

import (
	"context"
	"crypto/sha256"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pzlauncher/apps/pz-agent/internal/discover"
	"pzlauncher/apps/pz-agent/internal/ingest"
	"pzlauncher/apps/pz-agent/internal/pzdetect"
)

func main() {
	serverID := flag.String("server", "", "server ID (auto-detected if omitted)")
	backendURL := flag.String("backend", "http://localhost:8080", "backend base URL")
	modsDir := flag.String("mods", "", "local mods directory to scan (auto-detected if omitted)")
	gameVersion := flag.String("game-version", "42.8", "game version string")
	interval := flag.Duration("interval", 5*time.Minute, "sync interval (0 = run once and exit)")
	token := flag.String("token", "", "agent auth token (or set PZ_AGENT_TOKEN env var)")
	flag.Parse()

	// Auto-detect PZ server if -server or -mods not provided.
	if *serverID == "" || *modsDir == "" {
		log.Printf("agent: auto-detecting PZ server...")
		detected := pzdetect.Detect()
		if detected != nil {
			if *serverID == "" {
				*serverID = detected.ServerName
				log.Printf("agent: auto-detected server name: %q", *serverID)
			}
			if *modsDir == "" {
				*modsDir = detected.ModsDir
				log.Printf("agent: auto-detected mods dir: %q", *modsDir)
			}
		} else {
			log.Printf("agent: auto-detection failed — no PZ server found")
		}
	}

	if *serverID == "" {
		log.Fatal("agent: could not detect server name. Use -server flag or ensure PZ server is installed.")
	}
	if *modsDir == "" {
		log.Fatal("agent: could not detect mods directory. Use -mods flag or ensure PZ server is installed.")
	}

	// Root context — cancelled on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Resolve token: flag → env → auto-register (with retry).
	effectiveToken := *token
	if effectiveToken == "" {
		effectiveToken = os.Getenv("PZ_AGENT_TOKEN")
	}

	bootstrapClient := ingest.NewClient(*backendURL, *serverID).WithServerName(*serverID)
	if effectiveToken == "" {
		log.Printf("agent: no token provided, registering with backend...")
		var err error
		effectiveToken, err = bootstrapClient.Register(ctx)
		if err != nil {
			log.Printf("agent: registration failed (%v) — proceeding without auth (backend may reject requests)", err)
		} else {
			log.Printf("agent: registered, token=%s...", effectiveToken[:8])
		}
	}

	scanner := discover.NewScanner(*modsDir)
	client := bootstrapClient.WithToken(effectiveToken)

	// lastContentHash tracks the hash of (mod-id + sha256) pairs from the last
	// successful manifest publish. We only re-publish when this changes.
	var lastContentHash string

	sync := func() {
		mods, err := scanner.Scan()
		if err != nil {
			log.Printf("agent: discover error: %v", err)
			// Still heartbeat so backend knows agent is alive but degraded.
			_ = heartbeat(ctx, client, 0)
			return
		}
		log.Printf("agent: discovered %d mod(s) in %q", len(mods), *modsDir)

		// Push blobs — each is retried independently; partial success is fine.
		pushed := 0
		for _, mod := range mods {
			if err := client.PushBlob(ctx, mod); err != nil {
				log.Printf("agent: push blob %s: %v", mod.ID, err)
				continue
			}
			log.Printf("agent: pushed %s (%s...)", mod.ID, mod.SHA256[:12])
			pushed++
		}

		// Publish manifest only when content actually changed AND all blobs OK.
		if pushed == len(mods) && len(mods) > 0 {
			ch := contentHash(mods)
			if ch != lastContentHash {
				version := time.Now().UTC().Format("20060102T150405Z")
				if err := client.PublishManifest(ctx, mods, *gameVersion, version); err != nil {
					log.Printf("agent: publish manifest: %v", err)
				} else {
					log.Printf("agent: manifest published (version %s, %d mods)", version, len(mods))
					lastContentHash = ch
				}
			} else {
				log.Printf("agent: manifest unchanged, skipping publish")
			}
		}

		// Heartbeat — always, independent of sync outcome.
		_ = heartbeat(ctx, client, len(mods))
	}

	sync()
	if *interval == 0 {
		return
	}
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Printf("agent: shutting down")
			return
		case <-ticker.C:
			sync()
		}
	}
}

// heartbeat sends a heartbeat and logs the result without returning an error
// to the caller — a missed heartbeat should never abort the sync loop.
func heartbeat(ctx context.Context, client *ingest.Client, modCount int) error {
	if err := client.Heartbeat(ctx, modCount); err != nil {
		log.Printf("agent: heartbeat: %v", err)
		return err
	}
	log.Printf("agent: heartbeat ok (modCount=%d)", modCount)
	return nil
}

// contentHash returns a stable fingerprint of the current mod set so we can
// detect changes without comparing full manifests.
func contentHash(mods []discover.Mod) string {
	h := sha256.New()
	for _, m := range mods {
		fmt.Fprintf(h, "%s=%s\n", m.ID, m.SHA256)
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}
