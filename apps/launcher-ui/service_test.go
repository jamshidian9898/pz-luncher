package main

import (
	"os"
	"path/filepath"
	"testing"

	"pzlauncher/libs/ziputil"
)

func newTestUIService(t *testing.T, workspaceRoot string) *UIService {
	t.Helper()
	return &UIService{
		workspaceRoot: workspaceRoot,
		sessions:      make(map[string]*SessionStatus),
	}
}

func TestGetSessionStatus_UnknownSessionReturnsIdle(t *testing.T) {
	s := newTestUIService(t, t.TempDir())
	st, err := s.GetSessionStatus("never-seen")
	if err != nil {
		t.Fatalf("GetSessionStatus: %v", err)
	}
	if st.State != "idle" || st.Progress != 0 {
		t.Fatalf("expected idle/0 for unknown session, got %+v", st)
	}
}

func TestUpdateSessionFromEvent_TracksProgressThroughStates(t *testing.T) {
	s := newTestUIService(t, t.TempDir())
	sessionID := "sess-1"

	s.emitEvent(UIEvent{Type: EventModResolveStart, SessionID: sessionID})
	st, _ := s.GetSessionStatus(sessionID)
	if st.State != "resolving" || st.Progress != 5 {
		t.Fatalf("after resolve start: expected resolving/5, got %+v", st)
	}

	s.emitEvent(UIEvent{Type: EventDownloadStart, SessionID: sessionID, PackageID: "ModA"})
	st, _ = s.GetSessionStatus(sessionID)
	if st.State != "downloading" || st.CurrentMod != "ModA" {
		t.Fatalf("after download start: expected downloading/ModA, got %+v", st)
	}

	s.emitEvent(UIEvent{
		Type:      EventDownloadProgress,
		SessionID: sessionID,
		Progress:  &Progress{Current: 50, Total: 100, Percent: 50, Speed: 1024, ETA: 5},
	})
	st, _ = s.GetSessionStatus(sessionID)
	if st.Progress != 50 || st.DownloadSpeed != 1024 || st.ETA != 5 {
		t.Fatalf("after download progress: unexpected state %+v", st)
	}

	s.emitEvent(UIEvent{Type: EventError, SessionID: sessionID, Error: "BOOM"})
	st, _ = s.GetSessionStatus(sessionID)
	if st.State != "error" || len(st.Errors) != 1 || st.Errors[0] != "BOOM" {
		t.Fatalf("after error event: unexpected state %+v", st)
	}
}

func TestRepairCache_RemovesCorruptedModDir(t *testing.T) {
	root := t.TempDir()
	s := newTestUIService(t, root)

	// A mod dir whose name does NOT match its real content hash — corrupted.
	corruptDir := filepath.Join(root, "mods", "not-the-real-hash")
	if err := os.MkdirAll(corruptDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(corruptDir, "data.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := s.RepairCache(); err != nil {
		t.Fatalf("RepairCache: %v", err)
	}

	if _, err := os.Stat(corruptDir); !os.IsNotExist(err) {
		t.Fatalf("expected corrupted mod dir to be removed, stat err = %v", err)
	}
}

func TestRepairCache_KeepsValidModDir(t *testing.T) {
	root := t.TempDir()
	s := newTestUIService(t, root)

	// First create the dir, hash it for real, then rename it to match its own hash
	// (mirrors how the real pipeline names mod dirs by their content hash).
	tmpDir := filepath.Join(root, "mods", "tmp-build")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "data.txt"), []byte("valid content"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	hash, _, err := ziputil.HashDir(tmpDir)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	validDir := filepath.Join(root, "mods", hash)
	if err := os.Rename(tmpDir, validDir); err != nil {
		t.Fatalf("rename: %v", err)
	}

	if err := s.RepairCache(); err != nil {
		t.Fatalf("RepairCache: %v", err)
	}

	if _, err := os.Stat(validDir); err != nil {
		t.Fatalf("expected valid mod dir to survive repair, got err = %v", err)
	}
}

func TestRepairCache_RemovesEmptyVersionDir(t *testing.T) {
	root := t.TempDir()
	s := newTestUIService(t, root)

	emptyVersionDir := filepath.Join(root, "versions", "42.8")
	if err := os.MkdirAll(emptyVersionDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := s.RepairCache(); err != nil {
		t.Fatalf("RepairCache: %v", err)
	}
	if _, err := os.Stat(emptyVersionDir); !os.IsNotExist(err) {
		t.Fatalf("expected empty version dir to be removed, stat err = %v", err)
	}
}

func TestRepairCache_NoModsOrVersionsDirsIsNoop(t *testing.T) {
	s := newTestUIService(t, t.TempDir())
	if err := s.RepairCache(); err != nil {
		t.Fatalf("expected no error when mods/versions dirs don't exist, got %v", err)
	}
}
