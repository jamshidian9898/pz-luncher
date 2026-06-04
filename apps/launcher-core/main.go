package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"pzlauncher/libs/contracts"
	"pzlauncher/libs/fixtures"
	"pzlauncher/libs/game"
	"pzlauncher/libs/launchstate"
	"pzlauncher/libs/profile"
	"pzlauncher/libs/providers"
	"pzlauncher/libs/resolver"
	"pzlauncher/libs/session"
)

func main() {
	serverPath := flag.String("server", "examples/server.json", "path to server descriptor JSON")
	cacheDir := flag.String("cache", "cache/sha256", "local cache directory")
	profilesDir := flag.String("profiles", "profiles", "profiles base dir")
	demo := flag.Bool("demo", false, "seed deterministic demo blobs and run in demo mode")
	flag.Parse()

	data, err := ioutil.ReadFile(*serverPath)
	if err != nil {
		log.Fatalf("read server: %v", err)
	}
	var srv struct {
		Name     string `json:"name"`
		Manifest string `json:"manifestUrl"`
		ServerID string `json:"serverId"`
	}
	if err := json.Unmarshal(data, &srv); err != nil {
		log.Fatalf("parse server: %v", err)
	}

	mdata, err := ioutil.ReadFile(srv.Manifest)
	if err != nil {
		log.Fatalf("read manifest: %v", err)
	}
	var manifest contracts.Manifest
	if err := json.Unmarshal(mdata, &manifest); err != nil {
		log.Fatalf("parse manifest: %v", err)
	}

	// Resolve packages
	r := resolver.NewDefaultResolver()
	resolved, err := r.Resolve(manifest)
	if err != nil {
		log.Fatalf("resolve manifest: %v", err)
	}
	fmt.Printf("Resolved %d packages\n", len(resolved))

	// If demo flag set, seed deterministic blobs (only mod-a)
	if *demo {
		if err := fixtures.SeedCache(manifest, *cacheDir, []string{"mod-a"}); err != nil {
			log.Fatalf("seed cache: %v", err)
		}
		log.Printf("demo: seeded cache for mod-a into %s", *cacheDir)
	}

	// Initialize state machine early for full flow tracking
	sm := launchstate.NewSimpleMachine()
	sm.Transition(contracts.LaunchStateResolvingPackages)

	// Providers and minimal runtime fallback decision
	local := providers.NewLocalCacheProvider(*cacheDir)
	steam := providers.NewSteamProvider()
	provs := []providers.Provider{local, steam}

	var decisions []contracts.ProviderDecision
	resolved, decisions, err = providers.ApplyFallback(context.Background(), resolved, provs)
	if err != nil {
		log.Fatalf("apply provider fallback: %v", err)
	}
	for _, p := range resolved {
		fmt.Printf("package %s -> provider=%s cached=%v\n", p.ID, p.ProviderName, p.Cached)
	}

	// Prepare profile
	pb := profile.NewProfileBuilder(*profilesDir)
	profilePath, err := pb.Prepare(srv.ServerID, manifest.ID, resolved, *cacheDir)
	if err != nil {
		log.Fatalf("prepare profile: %v", err)
	}
	fmt.Printf("Profile prepared at %s\n", profilePath)

	// Print human-readable trace summary
	fmt.Println("\n=== Provider Decision Trace ===")
	for _, d := range decisions {
		status := "✓ cached"
		if !d.Cached {
			status = "✗ download needed"
		}
		fmt.Printf("\n[%s] %s@%s\n", status, d.PackageID, d.PackageVersion)
		fmt.Printf("  SHA256: %s...%s\n", d.PackageSHA256[:8], d.PackageSHA256[len(d.PackageSHA256)-8:])
		fmt.Printf("  Decision: %s → %s\n", d.FinalReason, d.ChosenProvider)
		fmt.Printf("  Duration: %dms\n", d.TotalDurationMs)
		fmt.Println("  Attempts:")
		for _, a := range d.Attempts {
			result := "✗"
			if a.Exists {
				result = "✓"
			}
			if a.Error != "" {
				result = "⚠"
			}
			fmt.Printf("    %s %s (%dms)", result, a.ProviderName, a.DurationMs)
			if a.Exists && a.CachePath != "" {
				fmt.Printf(" → %s", a.CachePath)
			}
			if a.Error != "" {
				fmt.Printf(" [error: %s]", a.Error)
			}
			fmt.Println()
		}
	}

	// Write structured provider decision trace to profile
	traceFile := filepath.Join(profilePath, "provider-trace.json")
	traceData, err := json.MarshalIndent(decisions, "", "  ")
	if err != nil {
		log.Fatalf("marshal trace: %v", err)
	}
	if err := os.WriteFile(traceFile, traceData, 0o644); err != nil {
		log.Fatalf("write trace file: %v", err)
	}
	fmt.Printf("\nStructured provider trace written to %s\n", traceFile)

	// === DOWNLOAD SESSION EXECUTION ===
	sessionsDir := filepath.Join(*profilesDir, ".sessions")
	sessionMgr := session.NewSimpleManager(sessionsDir)

	// Use composite executor that routes to provider-specific implementations
	// - LocalCache: verifies existing files
	// - Steam: real download from Steam Workshop with API + SteamCMD fallback
	executor := session.DefaultExecutor(*cacheDir)

	// Optional: Configure Steam executor with steamcmd if available
	if steamcmdPath := session.FindSteamCMD(); steamcmdPath != "" {
		fmt.Printf("[Steam] SteamCMD found at: %s\n", steamcmdPath)
	}

	// Create or resume session from provider decisions
	sm.Transition(contracts.LaunchStateCreatingSession)
	sess, err := sessionMgr.CreateSession(srv.ServerID, profilePath, decisions)
	if err != nil {
		log.Fatalf("create session: %v", err)
	}
	fmt.Printf("\n[Session] ID: %s\n", sess.ID)
	fmt.Printf("[Session] Packages: %d total, %d cached, %d to download\n",
		sess.Summary.TotalPackages, sess.Summary.SkippedCount, sess.Summary.DownloadCount)

	// Execute session (idempotent - will resume if partial)
	if !sess.IsComplete {
		sm.Transition(contracts.LaunchStateDownloading)
		fmt.Println("\n=== Download Session Execution ===")
		if err := sessionMgr.Execute(context.Background(), sess, executor); err != nil {
			log.Fatalf("session execution: %v", err)
		}
		sm.Transition(contracts.LaunchStateVerifying)
	}

	// Print execution summary
	fmt.Println("\n=== Session Execution Summary ===")
	for _, exec := range sess.Executions {
		icon := "⏸"
		switch exec.State {
		case contracts.PackageStateSkipped:
			icon = "⏭ skipped"
		case contracts.PackageStateComplete:
			icon = "✓ complete"
		case contracts.PackageStateFailed:
			icon = "✗ failed"
		case contracts.PackageStateDownloading:
			icon = "↓ downloading"
		}
		fmt.Printf("  [%s] %s (attempts: %d, duration: %dms)\n",
			icon, exec.PackageID, exec.Attempts, exec.DurationMs)
	}
	fmt.Printf("\nSession: %d/%d complete, %d failed\n",
		sess.Summary.CompletedCount, sess.Summary.TotalPackages, sess.Summary.FailedCount)

	// Write session trace
	sessionTraceFile := filepath.Join(profilePath, "session-trace.json")
	sessionTrace := sessionMgr.GetTrace(sess)
	sessionTraceData, err := json.MarshalIndent(sessionTrace, "", "  ")
	if err != nil {
		log.Fatalf("marshal session trace: %v", err)
	}
	if err := os.WriteFile(sessionTraceFile, sessionTraceData, 0o644); err != nil {
		log.Fatalf("write session trace: %v", err)
	}
	fmt.Printf("Session trace written to %s\n", sessionTraceFile)

	sm.Transition(contracts.LaunchStateMaterializing)

	// Find game and launch
	finder := game.NewSimpleFinder()
	inst, err := finder.FindInstallation()
	if err != nil {
		log.Fatalf("find installation: %v", err)
	}
	launcher := game.NewSimpleLauncher()
	req := contracts.LaunchRequest{ServerID: srv.ServerID, ProfileID: profilePath, ManifestID: manifest.ID}
	res, err := launcher.Launch(inst, req)
	if err != nil {
		log.Fatalf("launch failed: %v", err)
	}
	fmt.Printf("launch result: success=%v profile=%s\n", res.Success, res.ProfileID)

	// Record final state
	sm.Transition(contracts.LaunchStateRunning)
	fmt.Printf("Final state: %s\n", sm.CurrentState())

	fmt.Println("Done.")
}
