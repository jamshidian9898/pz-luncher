package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"pzlauncher/apps/backend/internal/manifest"
)

func writeRegistryFile(t *testing.T, path string, servers []*ServerRecord) {
	t.Helper()
	data, err := json.Marshal(registryFile{Servers: servers})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestLoadFromFile_PopulatesServers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")
	writeRegistryFile(t, path, []*ServerRecord{{ID: "srv-1", Name: "Server One"}})

	reg, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	srv, ok := reg.Get("srv-1")
	if !ok || srv.Name != "Server One" {
		t.Fatalf("expected srv-1 loaded, got %+v ok=%v", srv, ok)
	}
	// Status defaults to online when absent.
	if srv.Status != "online" {
		t.Fatalf("expected default status 'online', got %q", srv.Status)
	}
}

func TestLoadFromFile_MissingFileErrors(t *testing.T) {
	if _, err := LoadFromFile(filepath.Join(t.TempDir(), "nope.json")); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestUpsert_WritesBackToLoadedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")
	writeRegistryFile(t, path, nil)

	reg, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := reg.Upsert(&ServerRecord{ID: "auto-1", Name: "Auto Registered", Status: "online"}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// Simulate a restart: reload from the same file and confirm the
	// auto-registered server survived.
	reg2, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	srv, ok := reg2.Get("auto-1")
	if !ok || srv.Name != "Auto Registered" {
		t.Fatalf("expected auto-1 to survive restart, got %+v ok=%v", srv, ok)
	}
}

func TestUpsert_NoWriteBackWithoutLoadedFile(t *testing.T) {
	reg := NewMemoryRegistry()
	// Should not panic or error even though there's no backing file.
	if err := reg.Upsert(&ServerRecord{ID: "srv-1"}); err != nil {
		t.Fatalf("expected no error for in-memory-only registry, got %v", err)
	}
	if _, ok := reg.Get("srv-1"); !ok {
		t.Fatal("expected in-memory upsert to still work")
	}
}

func TestRecordHeartbeat_SetsOnlineStatus(t *testing.T) {
	reg := NewMemoryRegistry()
	reg.RecordHeartbeat("srv-1", 5, "v1")

	state := reg.AgentStateFor("srv-1")
	if state == nil {
		t.Fatal("expected agent state to exist after heartbeat")
	}
	if state.Status != AgentOnline {
		t.Fatalf("expected online status, got %q", state.Status)
	}
	if state.ModCount != 5 {
		t.Fatalf("expected modCount 5, got %d", state.ModCount)
	}
}

func TestAgentStateFor_UnknownServerReturnsNil(t *testing.T) {
	reg := NewMemoryRegistry()
	if got := reg.AgentStateFor("never-seen"); got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

func TestComputeStatus_Thresholds(t *testing.T) {
	cases := []struct {
		name string
		age  time.Duration
		want AgentStatus
	}{
		{"just now", 0, AgentOnline},
		{"89 seconds", 89 * time.Second, AgentOnline},
		{"91 seconds", 91 * time.Second, AgentDegraded},
		{"4 minutes", 4 * time.Minute, AgentDegraded},
		{"6 minutes", 6 * time.Minute, AgentOffline},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := computeStatus(time.Now().Add(-c.age))
			if got != c.want {
				t.Errorf("computeStatus(age=%v) = %q, want %q", c.age, got, c.want)
			}
		})
	}
}

func TestSetManifestStore_SwapIsUsedByUpsertManifest(t *testing.T) {
	reg := NewMemoryRegistry()
	custom := manifest.NewStore()
	reg.SetManifestStore(custom)

	if _, err := reg.UpsertManifest("srv-1", []byte(`{"serverId":"srv-1","mods":[]}`)); err != nil {
		t.Fatalf("upsert manifest: %v", err)
	}
	if custom.Latest("srv-1") == nil {
		t.Fatal("expected the swapped-in store to receive the manifest")
	}
}
