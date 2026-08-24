package manifest

import (
	"path/filepath"
	"testing"
)

func sampleManifestJSON(mods string) []byte {
	return []byte(`{"serverId":"srv","gameVersion":"42.8","mods":[` + mods + `],"launchArgs":[],"profile":{}}`)
}

func TestPut_AssignsSequentialVersions(t *testing.T) {
	s := NewStore()
	v1, err := s.Put("srv", sampleManifestJSON(`{"id":"A","name":"A","version":"1","sha256":"aaa","dependencies":[]}`))
	if err != nil {
		t.Fatalf("put 1: %v", err)
	}
	v2, err := s.Put("srv", sampleManifestJSON(`{"id":"A","name":"A","version":"2","sha256":"bbb","dependencies":[]}`))
	if err != nil {
		t.Fatalf("put 2: %v", err)
	}
	if v1 != 1 || v2 != 2 {
		t.Fatalf("expected versions 1,2 got %d,%d", v1, v2)
	}
}

func TestLatest_ReturnsMostRecent(t *testing.T) {
	s := NewStore()
	_, _ = s.Put("srv", sampleManifestJSON(`{"id":"A","name":"A","version":"1","sha256":"aaa","dependencies":[]}`))
	_, _ = s.Put("srv", sampleManifestJSON(`{"id":"A","name":"A","version":"2","sha256":"bbb","dependencies":[]}`))

	latest := s.Latest("srv")
	if latest == nil || latest.Version != 2 {
		t.Fatalf("expected latest version 2, got %+v", latest)
	}
}

func TestLatest_UnknownServerReturnsNil(t *testing.T) {
	s := NewStore()
	if got := s.Latest("nope"); got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

func TestHistory_TrimsToMaxVersions(t *testing.T) {
	s := NewStore()
	for i := 0; i < MaxVersions+5; i++ {
		if _, err := s.Put("srv", sampleManifestJSON(`{"id":"A","name":"A","version":"1","sha256":"aaa","dependencies":[]}`)); err != nil {
			t.Fatalf("put %d: %v", i, err)
		}
	}
	history := s.History("srv")
	if len(history) != MaxVersions {
		t.Fatalf("expected history capped at %d, got %d", MaxVersions, len(history))
	}
	// Newest first, and the oldest MaxVersions+5-MaxVersions=5 versions should be gone
	// so the lowest surviving version number is 6.
	if history[len(history)-1].Version != 6 {
		t.Fatalf("expected oldest surviving version to be 6, got %d", history[len(history)-1].Version)
	}
}

func TestDiff_AddedAndUnchanged(t *testing.T) {
	s := NewStore()
	_, _ = s.Put("srv", sampleManifestJSON(`{"id":"A","name":"A","version":"1","sha256":"aaa","dependencies":[]}`))
	_, _ = s.Put("srv", sampleManifestJSON(`{"id":"A","name":"A","version":"1","sha256":"aaa","dependencies":[]},{"id":"B","name":"B","version":"1","sha256":"bbb","dependencies":[]}`))

	diff, err := s.Diff("srv", 1, 2)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	if len(diff.Added) != 1 || diff.Added[0].ID != "B" {
		t.Fatalf("expected B added, got %+v", diff.Added)
	}
	if diff.Unchanged != 1 {
		t.Fatalf("expected 1 unchanged (A), got %d", diff.Unchanged)
	}
}

func TestDiff_FromZeroTreatsAllAsAdded(t *testing.T) {
	s := NewStore()
	_, _ = s.Put("srv", sampleManifestJSON(`{"id":"A","name":"A","version":"1","sha256":"aaa","dependencies":[]}`))

	diff, err := s.Diff("srv", 0, 1)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	if len(diff.Added) != 1 {
		t.Fatalf("expected 1 added mod from empty baseline, got %d", len(diff.Added))
	}
}

func TestDiff_UnknownServerErrors(t *testing.T) {
	s := NewStore()
	if _, err := s.Diff("nope", 0, 1); err == nil {
		t.Fatal("expected error for unknown server")
	}
}

func TestNewDiskStore_SurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	s1, err := NewDiskStore(dir)
	if err != nil {
		t.Fatalf("new disk store: %v", err)
	}
	if _, err := s1.Put("srv", sampleManifestJSON(`{"id":"A","name":"A","version":"1","sha256":"aaa","dependencies":[]}`)); err != nil {
		t.Fatalf("put: %v", err)
	}

	s2, err := NewDiskStore(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	latest := s2.Latest("srv")
	if latest == nil {
		t.Fatal("expected manifest to survive restart")
	}
	if latest.Version != 1 || len(latest.Manifest.Mods) != 1 || latest.Manifest.Mods[0].SHA256 != "aaa" {
		t.Fatalf("unexpected reloaded manifest: %+v", latest)
	}
}

func TestNewDiskStore_MissingDirStartsEmpty(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "does-not-exist")
	s, err := NewDiskStore(dir)
	if err != nil {
		t.Fatalf("expected no error for missing dir, got %v", err)
	}
	if s.Latest("srv") != nil {
		t.Fatal("expected empty store")
	}
}

func TestPut_EmptyServerIDErrors(t *testing.T) {
	s := NewStore()
	if _, err := s.Put("", sampleManifestJSON("")); err == nil {
		t.Fatal("expected error for empty serverID")
	}
}

func TestManifestFileName_SanitizesUnsafeChars(t *testing.T) {
	got := manifestFileName("my/server:weird name")
	want := "my_server_weird_name.json"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
