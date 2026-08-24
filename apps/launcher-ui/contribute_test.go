package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestNormalizeVersion(t *testing.T) {
	cases := map[string]string{
		"42.16.3":  "42.16.3",
		" 42.8 ":   "42.8",
		"42":       "42",
		"v42.8":    "",
		"unknown":  "",
		"":         "",
	}
	for input, want := range cases {
		if got := normalizeVersion(input); got != want {
			t.Errorf("normalizeVersion(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestCurrentPlatform(t *testing.T) {
	got := currentPlatform()
	if got == "" {
		t.Fatal("expected a non-empty platform string")
	}
	// Sanity: should be derived from the actual OS/arch, not hardcoded.
	if runtime.GOOS == "windows" && got != "windows-x64" && got != "windows-x86" {
		t.Errorf("unexpected platform on windows: %q", got)
	}
}

func TestDetectGameVersion_VersionHintFile(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, ".pz-version"), "42.13.1\n")
	if got := detectGameVersion(dir); got != "42.13.1" {
		t.Fatalf("expected 42.13.1, got %q", got)
	}
}

func TestDetectGameVersion_VersionTxtFallback(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "version.txt"), "41.78.7\n")
	if got := detectGameVersion(dir); got != "41.78.7" {
		t.Fatalf("expected 41.78.7, got %q", got)
	}
}

func TestDetectGameVersion_DirNamePattern(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ProjectZomboid-42.16.3")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if got := detectGameVersion(dir); got != "42.16.3" {
		t.Fatalf("expected 42.16.3, got %q", got)
	}
}

func TestDetectGameVersion_UnknownWhenNoMarkers(t *testing.T) {
	dir := t.TempDir()
	if got := detectGameVersion(dir); got != "unknown" {
		t.Fatalf("expected 'unknown', got %q", got)
	}
}

func TestCountFiles_CountsRegularFilesOnly(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "a.txt"), "a")
	writeTestFile(t, filepath.Join(dir, "sub", "b.txt"), "b")
	writeTestFile(t, filepath.Join(dir, "sub", "c.txt"), "c")

	count, total, err := countFiles(dir)
	if err != nil {
		t.Fatalf("countFiles: %v", err)
	}
	if count != 3 || total != 3 {
		t.Fatalf("expected 3 files, got count=%d total=%d", count, total)
	}
}

func TestEntryFromVersionDir_ComputesSize(t *testing.T) {
	dir := t.TempDir()
	versionDir := filepath.Join(dir, "42.8")
	writeTestFile(t, filepath.Join(versionDir, "file.dat"), "twelve bytes")
	writeTestFile(t, filepath.Join(versionDir, ".version"),
		`{"gameVersion":"42.8","platform":"linux-x64","sha256":"abc123"}`)

	entry, err := entryFromVersionDir(versionDir, "42.8", "cache")
	if err != nil {
		t.Fatalf("entryFromVersionDir: %v", err)
	}
	if entry.GameVersion != "42.8" || entry.Source != "cache" {
		t.Fatalf("unexpected entry: %+v", entry)
	}
	// Size includes both file.dat and the .version metadata file itself.
	if entry.SizeBytes <= int64(len("twelve bytes")) {
		t.Fatalf("expected size to include file.dat content, got %d", entry.SizeBytes)
	}
	if entry.TrustLevel != "unknown" {
		t.Fatalf("expected default trust 'unknown', got %q", entry.TrustLevel)
	}
}

func TestEntryFromVersionDir_MissingMetaErrors(t *testing.T) {
	dir := t.TempDir()
	if _, err := entryFromVersionDir(dir, "42.8", "cache"); err == nil {
		t.Fatal("expected error when .version metadata file is missing")
	}
}

func TestHashGameVersion_File(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "game.bin")
	content := "some game binary content"
	writeTestFile(t, path, content)

	app := &App{ui: NewUIService()}
	result, err := app.HashGameVersion(path)
	if err != nil {
		t.Fatalf("HashGameVersion: %v", err)
	}
	if result.SizeBytes != int64(len(content)) {
		t.Fatalf("expected size %d, got %d", len(content), result.SizeBytes)
	}
	if len(result.SHA256) != 64 {
		t.Fatalf("expected a 64-char hex sha256, got %q", result.SHA256)
	}
}

func TestHashGameVersion_Directory(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "a.txt"), "a")
	writeTestFile(t, filepath.Join(dir, "b.txt"), "b")

	app := &App{ui: NewUIService()}
	result1, err := app.HashGameVersion(dir)
	if err != nil {
		t.Fatalf("HashGameVersion: %v", err)
	}
	result2, err := app.HashGameVersion(dir)
	if err != nil {
		t.Fatalf("HashGameVersion (2nd run): %v", err)
	}
	if result1.SHA256 != result2.SHA256 {
		t.Fatalf("expected deterministic hash, got %q vs %q", result1.SHA256, result2.SHA256)
	}
}

func TestHashGameVersion_MissingPathErrors(t *testing.T) {
	app := &App{ui: NewUIService()}
	if _, err := app.HashGameVersion(filepath.Join(t.TempDir(), "does-not-exist")); err == nil {
		t.Fatal("expected error for missing path")
	}
}
