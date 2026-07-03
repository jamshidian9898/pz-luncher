package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"time"

	"pzlauncher/apps/pz-agent/internal/discover"
	"pzlauncher/apps/pz-agent/internal/ingest"
	"pzlauncher/apps/pz-agent/internal/pzdetect"
)

// Options carries the fully-parsed agent configuration. It is shared by the
// interactive path and the Windows service path.
type Options struct {
	ServerID    string
	BackendURL  string
	ModsDir     string
	GameVersion string
	Token       string
	Interval    time.Duration
	LogFile     string
}

// runAgent executes the agent loop until ctx is cancelled.
// With Interval == 0 it performs a single sync and returns.
func runAgent(ctx context.Context, o Options) error {
	if o.LogFile != "" {
		f, err := os.OpenFile(o.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return fmt.Errorf("agent: open log file: %w", err)
		}
		defer f.Close()
		log.SetOutput(f)
	}

	// Auto-detect PZ server if -server or -mods not provided.
	if o.ServerID == "" || o.ModsDir == "" {
		log.Printf("agent: auto-detecting PZ server...")
		detected := pzdetect.Detect()
		if detected != nil {
			if o.ServerID == "" {
				o.ServerID = detected.ServerName
				log.Printf("agent: auto-detected server name: %q", o.ServerID)
			}
			if o.ModsDir == "" {
				o.ModsDir = detected.ModsDir
				log.Printf("agent: auto-detected mods dir: %q", o.ModsDir)
			}
		} else {
			log.Printf("agent: auto-detection failed — no PZ server found")
		}
	}

	if o.ServerID == "" {
		return fmt.Errorf("agent: could not detect server name. Use -server flag or ensure PZ server is installed")
	}
	if o.ModsDir == "" {
		return fmt.Errorf("agent: could not detect mods directory. Use -mods flag or ensure PZ server is installed")
	}

	// Resolve token: flag → env → auto-register (with retry).
	effectiveToken := o.Token
	if effectiveToken == "" {
		effectiveToken = os.Getenv("PZ_AGENT_TOKEN")
	}

	bootstrapClient := ingest.NewClient(o.BackendURL, o.ServerID).WithServerName(o.ServerID)
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

	scanner := discover.NewScanner(o.ModsDir)
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
		log.Printf("agent: discovered %d mod(s) in %q", len(mods), o.ModsDir)

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
				if err := client.PublishManifest(ctx, mods, o.GameVersion, version); err != nil {
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
	if o.Interval == 0 {
		return nil
	}
	ticker := time.NewTicker(o.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Printf("agent: shutting down")
			return nil
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
