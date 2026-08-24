package join

import (
	"context"
	"strings"
	"testing"

	"pzlauncher/apps/backend/internal/registry"
)

func newTestRegistry(t *testing.T) *registry.Registry {
	t.Helper()
	return registry.NewMemoryRegistry()
}

func TestResolve_ServerNotFound(t *testing.T) {
	reg := newTestRegistry(t)
	resolver := NewResolver(reg, "http://localhost:8080", nil)

	_, err := resolver.Resolve(context.Background(), "nope", "sess-1", "2026-01-01T00:00:00Z")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected 'not found' error, got %v", err)
	}
}

func TestResolve_ServerOffline(t *testing.T) {
	reg := newTestRegistry(t)
	reg.Upsert(&registry.ServerRecord{ID: "srv-1", Status: "offline"})
	resolver := NewResolver(reg, "http://localhost:8080", nil)

	_, err := resolver.Resolve(context.Background(), "srv-1", "sess-1", "2026-01-01T00:00:00Z")
	if err == nil || !strings.Contains(err.Error(), "offline") {
		t.Fatalf("expected 'offline' error, got %v", err)
	}
}

func TestResolve_NoManifestFallsBackToEmptyPlan(t *testing.T) {
	reg := newTestRegistry(t)
	reg.Upsert(&registry.ServerRecord{ID: "srv-1", Status: "online", GameVersion: "42.8"})
	resolver := NewResolver(reg, "http://localhost:8080", nil)

	resp, err := resolver.Resolve(context.Background(), "srv-1", "sess-1", "2026-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(resp.DownloadPlan) != 0 {
		t.Fatalf("expected empty download plan with no manifest, got %+v", resp.DownloadPlan)
	}
	if resp.Manifest.GameVersion != "42.8" {
		t.Fatalf("expected fallback manifest to carry the server's gameVersion, got %q", resp.Manifest.GameVersion)
	}
}

func TestResolve_BuildsDownloadPlanFromManifest(t *testing.T) {
	reg := newTestRegistry(t)
	reg.Upsert(&registry.ServerRecord{ID: "srv-1", Status: "online"})
	if _, err := reg.UpsertManifest("srv-1", []byte(`{
		"serverId": "srv-1",
		"gameVersion": "42.8",
		"mods": [{"id":"ModA","name":"Mod A","version":"1","sha256":"abc123","sizeBytes":1000,"dependencies":[]}],
		"launchArgs": [],
		"profile": {}
	}`)); err != nil {
		t.Fatalf("upsert manifest: %v", err)
	}

	resolver := NewResolver(reg, "http://localhost:8080", nil)
	resp, err := resolver.Resolve(context.Background(), "srv-1", "sess-1", "2026-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(resp.DownloadPlan) != 1 {
		t.Fatalf("expected 1 download item, got %d", len(resp.DownloadPlan))
	}
	item := resp.DownloadPlan[0]
	if item.SHA256 != "abc123" || item.SizeBytes != 1000 {
		t.Fatalf("unexpected download item: %+v", item)
	}
	wantURL := "http://localhost:8080/api/v1/download/abc123"
	if item.URL != wantURL {
		t.Fatalf("expected URL %q, got %q", wantURL, item.URL)
	}
	if resp.ManifestVersion != 1 {
		t.Fatalf("expected manifestVersion 1, got %d", resp.ManifestVersion)
	}
}

func TestResolve_SkipsModsWithoutSHA256(t *testing.T) {
	reg := newTestRegistry(t)
	reg.Upsert(&registry.ServerRecord{ID: "srv-1", Status: "online"})
	if _, err := reg.UpsertManifest("srv-1", []byte(`{
		"serverId": "srv-1",
		"mods": [{"id":"NoHash","name":"No Hash","version":"1","sha256":"","dependencies":[]}],
		"launchArgs": [],
		"profile": {}
	}`)); err != nil {
		t.Fatalf("upsert manifest: %v", err)
	}

	resolver := NewResolver(reg, "http://localhost:8080", nil)
	resp, err := resolver.Resolve(context.Background(), "srv-1", "sess-1", "2026-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(resp.DownloadPlan) != 0 {
		t.Fatalf("expected mods without sha256 to be skipped, got %+v", resp.DownloadPlan)
	}
}
