package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"time"

	"pzlauncher/apps/pz-agent/internal/discover"
	"pzlauncher/apps/pz-agent/internal/ingest"
	"pzlauncher/apps/pz-agent/internal/pzdetect"
	"pzlauncher/apps/pz-agent/internal/tokenstore"
	"pzlauncher/apps/pz-agent/internal/watch"
)

// Options carries the fully-parsed agent configuration. It is shared by the
// interactive path and the Windows service path.
type Options struct {
	ServerID    string
	BackendURL  string
	ModsDir     string
	GameVersion string
	MaxPlayers  int // auto-detected only; no backend field to submit it yet
	Token       string
	TokenDir    string // where the access token is persisted; "" = tokenstore.DefaultDir()
	Interval    time.Duration
	LogFile     string
	ServiceName string
}

// agentSession bundles the current authenticated client with what's needed
// to re-register (and persist the new token) if the Backend ever responds
// 401 — e.g. because the token was revoked or the Backend's store was reset.
type agentSession struct {
	client      *ingest.Client
	bootstrap   *ingest.Client // unauthenticated client with serverName set, used to re-register
	tokenDir    string
	serverID    string
	gameVersion string
}

func (s *agentSession) reregister(ctx context.Context) error {
	token, err := s.bootstrap.Register(ctx, s.gameVersion)
	if err != nil {
		return err
	}
	s.client = s.bootstrap.WithToken(token)
	if serr := tokenstore.Save(s.tokenDir, s.serverID, token); serr != nil {
		log.Printf("agent: warning: failed to persist token: %v", serr)
	}
	log.Printf("agent: re-registered, new token=%s", safeTokenPrefix(token))
	return nil
}

// call runs fn against the current client. If fn fails with ErrUnauthorized
// (Backend returned 401), it re-registers once and retries fn with the
// fresh client before giving up.
func (s *agentSession) call(ctx context.Context, opName string, fn func(*ingest.Client) error) error {
	err := fn(s.client)
	if err != nil && errors.Is(err, ingest.ErrUnauthorized) {
		log.Printf("agent: %s: token rejected (401), re-registering...", opName)
		if rerr := s.reregister(ctx); rerr != nil {
			return fmt.Errorf("%s: re-register after 401: %w", opName, rerr)
		}
		err = fn(s.client)
	}
	return err
}

func safeTokenPrefix(token string) string {
	if len(token) <= 8 {
		return token
	}
	return token[:8] + "..."
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

	// Auto-detect PZ server if -server, -mods, or -game-version not provided.
	if o.ServerID == "" || o.ModsDir == "" || o.GameVersion == "" {
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
			if o.GameVersion == "" {
				o.GameVersion = detected.GameVersion
				log.Printf("agent: auto-detected game version: %q", o.GameVersion)
			}
			if detected.MaxPlayers > 0 {
				o.MaxPlayers = detected.MaxPlayers
				log.Printf("agent: auto-detected max players: %d (not yet submitted to backend)", o.MaxPlayers)
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
	if o.GameVersion == "" {
		o.GameVersion = "unknown"
	}

	tokenDir := o.TokenDir
	if tokenDir == "" {
		tokenDir = tokenstore.DefaultDir()
	}

	// Resolve token: flag → env → persisted file → auto-register (with retry).
	effectiveToken := o.Token
	if effectiveToken == "" {
		effectiveToken = os.Getenv("PZ_AGENT_TOKEN")
	}
	if effectiveToken == "" {
		if saved, err := tokenstore.Load(tokenDir, o.ServerID); err != nil {
			log.Printf("agent: warning: failed to load persisted token: %v", err)
		} else if saved != "" {
			effectiveToken = saved
			log.Printf("agent: loaded persisted token from %s", tokenstore.Path(tokenDir, o.ServerID))
		}
	}

	bootstrapClient := ingest.NewClient(o.BackendURL, o.ServerID).WithServerName(o.ServerID)
	if effectiveToken == "" {
		log.Printf("agent: no token provided, registering with backend...")
		var err error
		effectiveToken, err = bootstrapClient.Register(ctx, o.GameVersion)
		if err != nil {
			log.Printf("agent: registration failed (%v) — proceeding without auth (backend may reject requests)", err)
		} else {
			log.Printf("agent: registered, token=%s", safeTokenPrefix(effectiveToken))
			if serr := tokenstore.Save(tokenDir, o.ServerID, effectiveToken); serr != nil {
				log.Printf("agent: warning: failed to persist token: %v", serr)
			}
		}
	}

	scanner := discover.NewScanner(o.ModsDir)
	session := &agentSession{
		client:      bootstrapClient.WithToken(effectiveToken),
		bootstrap:   bootstrapClient,
		tokenDir:    tokenDir,
		serverID:    o.ServerID,
		gameVersion: o.GameVersion,
	}

	// lastContentHash tracks the hash of (mod-id + sha256) pairs from the last
	// successful manifest publish. We only re-publish when this changes.
	var lastContentHash string

	sync := func() {
		mods, err := scanner.Scan()
		if err != nil {
			log.Printf("agent: discover error: %v", err)
			// Still heartbeat so backend knows agent is alive but degraded.
			_ = heartbeat(ctx, session, 0)
			return
		}
		log.Printf("agent: discovered %d mod(s) in %q", len(mods), o.ModsDir)

		// Push blobs — each is retried independently; partial success is fine.
		pushed := 0
		for _, mod := range mods {
			mod := mod
			err := session.call(ctx, "push_blob:"+mod.ID, func(c *ingest.Client) error {
				return c.PushBlob(ctx, mod)
			})
			if err != nil {
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
				err := session.call(ctx, "publish_manifest", func(c *ingest.Client) error {
					return c.PublishManifest(ctx, mods, o.GameVersion, version)
				})
				if err != nil {
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
		_ = heartbeat(ctx, session, len(mods))
	}

	// syncing guards against overlapping runs: the interval ticker and the
	// filesystem watcher can both request a sync; only one may run at a time.
	var syncing atomic.Bool
	runSync := func() {
		if !syncing.CompareAndSwap(false, true) {
			log.Printf("agent: sync already in progress, skipping this trigger")
			return
		}
		defer syncing.Store(false)
		sync()
	}

	runSync()
	if o.Interval == 0 {
		return nil
	}

	// Filesystem watch triggers an immediate sync on mod changes instead of
	// waiting for the next poll tick. Best-effort: if it can't be set up
	// (missing dir, platform limits) the interval ticker below is still the
	// fallback, so we never block startup on this.
	if err := watch.Watch(ctx, o.ModsDir, func() { go runSync() }); err != nil {
		log.Printf("agent: watch: %v", err)
	}

	ticker := time.NewTicker(o.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Printf("agent: shutting down")
			return nil
		case <-ticker.C:
			runSync()
		}
	}
}

// heartbeat sends a heartbeat and logs the result without returning an error
// to the caller — a missed heartbeat should never abort the sync loop.
func heartbeat(ctx context.Context, session *agentSession, modCount int) error {
	err := session.call(ctx, "heartbeat", func(c *ingest.Client) error {
		return c.Heartbeat(ctx, modCount)
	})
	if err != nil {
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
