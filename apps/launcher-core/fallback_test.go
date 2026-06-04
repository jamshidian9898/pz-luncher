package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"pzlauncher/libs/contracts"
	"pzlauncher/libs/providers"
)

func TestProviderFallback(t *testing.T) {
	tmp := t.TempDir()
	cacheDir := filepath.Join(tmp, "cache", "sha256")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}

	shaA := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	shaB := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	// create blob only for mod-a
	fpath := filepath.Join(cacheDir, shaA)
	if err := os.WriteFile(fpath, []byte("mod-a blob"), 0o644); err != nil {
		t.Fatal(err)
	}

	pkgs := []contracts.ResolvedPackage{
		{ID: "mod-a", Version: "1.0", SHA256: shaA},
		{ID: "mod-b", Version: "1.0", SHA256: shaB},
	}

	provs := []providers.Provider{
		providers.NewLocalCacheProvider(cacheDir),
		providers.NewSteamProvider(),
	}

	out, decisions, err := providers.ApplyFallback(context.Background(), pkgs, provs)
	if err != nil {
		t.Fatalf("ApplyFallback error: %v", err)
	}

	if out[0].ProviderName != "LocalCache" {
		t.Fatalf("expected mod-a to use LocalCache, got %s", out[0].ProviderName)
	}
	if out[1].ProviderName != "Steam" {
		t.Fatalf("expected mod-b to fallback to Steam, got %s", out[1].ProviderName)
	}

	if len(decisions) != 2 {
		t.Fatalf("expected 2 decisions, got %d", len(decisions))
	}
	if decisions[0].FinalReason == "" {
		t.Fatalf("expected decision finalReason for mod-a, got empty")
	}

	// Validate trace structure
	if len(decisions[0].Attempts) == 0 {
		t.Fatalf("expected attempts trace for mod-a, got none")
	}
	if decisions[0].Attempts[0].ProviderName != "LocalCache" {
		t.Fatalf("expected first attempt to be LocalCache, got %s", decisions[0].Attempts[0].ProviderName)
	}
	if !decisions[0].Attempts[0].Exists {
		t.Fatalf("expected LocalCache to report exists=true for mod-a")
	}

	// Validate mod-b fallback trace
	if len(decisions[1].Attempts) < 2 {
		t.Fatalf("expected at least 2 attempts for mod-b (LocalCache + Steam)")
	}
	if decisions[1].Attempts[0].Exists {
		t.Fatalf("expected LocalCache to report exists=false for mod-b")
	}
	if decisions[1].Cached {
		t.Fatalf("expected mod-b to not be cached")
	}
}
