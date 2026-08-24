package tokenstore

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSaveLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, "my-server", "secret-token"); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := Load(dir, "my-server")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got != "secret-token" {
		t.Fatalf("expected secret-token, got %q", got)
	}
}

func TestLoad_MissingFileReturnsEmptyNoError(t *testing.T) {
	dir := t.TempDir()
	got, err := Load(dir, "never-registered")
	if err != nil {
		t.Fatalf("expected no error for missing token file, got %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestSave_EmptyTokenClearsFile(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, "srv", "token1"); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := Save(dir, "srv", ""); err != nil {
		t.Fatalf("clear: %v", err)
	}
	got, err := Load(dir, "srv")
	if err != nil {
		t.Fatalf("load after clear: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty after clear, got %q", got)
	}
}

func TestSave_FilePermissionsRestricted(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not meaningful on Windows")
	}
	dir := t.TempDir()
	if err := Save(dir, "srv", "token"); err != nil {
		t.Fatalf("save: %v", err)
	}
	fi, err := os.Stat(Path(dir, "srv"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Fatalf("expected 0600 permissions, got %o", perm)
	}
}

func TestPath_SanitizesServerID(t *testing.T) {
	dir := "/tmp/tokens"
	got := Path(dir, "my/server:weird name")
	want := filepath.Join(dir, "my_server_weird_name.token")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestPath_EmptyServerIDFallsBackToDefault(t *testing.T) {
	dir := "/tmp/tokens"
	got := Path(dir, "")
	want := filepath.Join(dir, "default.token")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
