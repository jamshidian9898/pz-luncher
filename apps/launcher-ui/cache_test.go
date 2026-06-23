package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetCacheStats_Empty(t *testing.T) {
	tmpdir := t.TempDir()
	app := &App{service: &UIService{workspaceRoot: tmpdir}}

	stats, err := app.GetCacheStats()
	if err != nil {
		t.Fatalf("GetCacheStats failed: %v", err)
	}
	if stats.TotalBytes != 0 {
		t.Errorf("expected 0 total bytes for empty cache, got %d", stats.TotalBytes)
	}
	if len(stats.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(stats.Entries))
	}
}

func TestGetCacheStats_WithVersions(t *testing.T) {
	tmpdir := t.TempDir()
	app := &App{service: &UIService{workspaceRoot: tmpdir}}

	// Create fake version directory with a file
	versionsDir := filepath.Join(tmpdir, "versions", "42.16")
	os.MkdirAll(versionsDir, 0o755)

	testFile := filepath.Join(versionsDir, "test.bin")
	os.WriteFile(testFile, make([]byte, 1000), 0o644)

	stats, err := app.GetCacheStats()
	if err != nil {
		t.Fatalf("GetCacheStats failed: %v", err)
	}

	if stats.VersionBytes != 1000 {
		t.Errorf("expected 1000 version bytes, got %d", stats.VersionBytes)
	}
	if len(stats.Entries) == 0 {
		t.Fatal("expected at least 1 entry")
	}

	entry := stats.Entries[0]
	if entry.Type != "version" {
		t.Errorf("expected type=version, got %s", entry.Type)
	}
	if entry.SizeBytes != 1000 {
		t.Errorf("expected 1000 bytes, got %d", entry.SizeBytes)
	}
}

func TestDeleteCacheEntry_NotFound(t *testing.T) {
	tmpdir := t.TempDir()
	app := &App{service: &UIService{workspaceRoot: tmpdir}}

	err := app.DeleteCacheEntry("version", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent entry")
	}
}

func TestDeleteCacheEntry_Success(t *testing.T) {
	tmpdir := t.TempDir()
	app := &App{service: &UIService{workspaceRoot: tmpdir}}

	// Create version to delete
	versionsDir := filepath.Join(tmpdir, "versions", "42.16")
	os.MkdirAll(versionsDir, 0o755)
	testFile := filepath.Join(versionsDir, "test.bin")
	os.WriteFile(testFile, []byte("test"), 0o644)

	// Delete it
	err := app.DeleteCacheEntry("version", "42.16")
	if err != nil {
		t.Fatalf("DeleteCacheEntry failed: %v", err)
	}

	// Verify it's gone
	if _, err := os.Stat(versionsDir); err == nil {
		t.Fatal("directory still exists after deletion")
	}
}

func TestDeleteCacheEntry_InUseProfile(t *testing.T) {
	tmpdir := t.TempDir()
	app := &App{service: &UIService{workspaceRoot: tmpdir}}

	// Create version
	versionsDir := filepath.Join(tmpdir, "versions", "42.16")
	os.MkdirAll(versionsDir, 0o755)

	// Create fake profile that uses this version
	profileDir := filepath.Join(tmpdir, "profiles", "test-server")
	os.MkdirAll(profileDir, 0o755)

	versionMetaFile := filepath.Join(profileDir, ".version")
	os.WriteFile(versionMetaFile, []byte(`{"gameVersion":"42.16"}`), 0o644)

	// Try to delete — should fail because profile uses it
	err := app.DeleteCacheEntry("version", "42.16")
	if err == nil {
		t.Fatal("expected error when deleting in-use version")
	}

	// Verify it still exists
	if _, err := os.Stat(versionsDir); err != nil {
		t.Fatal("version was deleted despite being in use")
	}
}

func TestDirSize(t *testing.T) {
	tmpdir := t.TempDir()

	// Create files
	os.WriteFile(filepath.Join(tmpdir, "file1.txt"), make([]byte, 100), 0o644)
	os.WriteFile(filepath.Join(tmpdir, "file2.txt"), make([]byte, 200), 0o644)

	size, err := dirSize(tmpdir)
	if err != nil {
		t.Fatalf("dirSize failed: %v", err)
	}
	if size != 300 {
		t.Errorf("expected 300 bytes, got %d", size)
	}
}

func TestVersionsUsedBy_Empty(t *testing.T) {
	tmpdir := t.TempDir()
	profiles := versionsUsedBy(tmpdir, "42.16")
	if len(profiles) != 0 {
		t.Errorf("expected 0 profiles, got %d", len(profiles))
	}
}

func TestVersionsUsedBy_Match(t *testing.T) {
	tmpdir := t.TempDir()

	// Create profile using 42.16
	profileDir := filepath.Join(tmpdir, "profiles", "survival")
	os.MkdirAll(profileDir, 0o755)
	versionFile := filepath.Join(profileDir, ".version")
	os.WriteFile(versionFile, []byte(`{"gameVersion":"42.16"}`), 0o644)

	// Create profile using 42.20
	profileDir2 := filepath.Join(tmpdir, "profiles", "rp-server")
	os.MkdirAll(profileDir2, 0o755)
	versionFile2 := filepath.Join(profileDir2, ".version")
	os.WriteFile(versionFile2, []byte(`{"gameVersion":"42.20"}`), 0o644)

	// Check who uses 42.16
	profiles := versionsUsedBy(tmpdir, "42.16")
	if len(profiles) != 1 {
		t.Errorf("expected 1 profile using 42.16, got %d", len(profiles))
	}
	if profiles[0] != "survival" {
		t.Errorf("expected 'survival', got %s", profiles[0])
	}
}
