package pzdetect

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestDetectServerName_FindsIniSkipsCompanionFiles(t *testing.T) {
	dir := t.TempDir()
	serverDir := filepath.Join(dir, "Server")
	writeFile(t, filepath.Join(serverDir, "myserver_SandboxVars.ini"), "")
	writeFile(t, filepath.Join(serverDir, "myserver_spawnpoints.ini"), "")
	writeFile(t, filepath.Join(serverDir, "myserver.ini"), "MaxPlayers=32\n")

	got := detectServerName(dir)
	if got != "myserver" {
		t.Fatalf("expected myserver, got %q", got)
	}
}

func TestDetectServerName_MissingDirReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	if got := detectServerName(dir); got != "" {
		t.Fatalf("expected empty for missing Server dir, got %q", got)
	}
}

func TestDetectMaxPlayers_ParsesFromINI(t *testing.T) {
	dir := t.TempDir()
	serverDir := filepath.Join(dir, "Server")
	writeFile(t, filepath.Join(serverDir, "myserver.ini"), "PVP=false\nMaxPlayers=64\nPublicName=Test\n")

	got := detectMaxPlayers(serverDir, "myserver")
	if got != 64 {
		t.Fatalf("expected 64, got %d", got)
	}
}

func TestDetectMaxPlayers_MissingSettingReturnsZero(t *testing.T) {
	dir := t.TempDir()
	serverDir := filepath.Join(dir, "Server")
	writeFile(t, filepath.Join(serverDir, "myserver.ini"), "PVP=false\n")

	if got := detectMaxPlayers(serverDir, "myserver"); got != 0 {
		t.Fatalf("expected 0 when MaxPlayers absent, got %d", got)
	}
}

func TestDetectMaxPlayers_MissingFileReturnsZero(t *testing.T) {
	dir := t.TempDir()
	if got := detectMaxPlayers(filepath.Join(dir, "Server"), "myserver"); got != 0 {
		t.Fatalf("expected 0 for missing ini, got %d", got)
	}
}

func TestParseModsFromINI_FindsWorkshopItems(t *testing.T) {
	dir := t.TempDir()
	serverDir := filepath.Join(dir, "Server")
	writeFile(t, filepath.Join(serverDir, "myserver.ini"), "Mods=SomeMod\nWorkshopItems=123456789\n")

	// No actual Workshop content dir exists on this machine, so this should
	// return "" (findWorkshopDirs finds nothing) rather than panic.
	got := parseModsFromINI(serverDir)
	if got != "" {
		t.Fatalf("expected empty (no workshop dirs present on test machine), got %q", got)
	}
}

func TestDetectGameVersionFromPath_VersionFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".pz-version"), "42.13.1\n")

	if got := detectGameVersionFromPath(dir); got != "42.13.1" {
		t.Fatalf("expected 42.13.1, got %q", got)
	}
}

func TestDetectGameVersionFromPath_VersionTxtFallback(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "version.txt"), "41.78.7\n")

	if got := detectGameVersionFromPath(dir); got != "41.78.7" {
		t.Fatalf("expected 41.78.7, got %q", got)
	}
}

func TestDetectGameVersionFromPath_DirNamePattern(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ProjectZomboid-42.16.3")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if got := detectGameVersionFromPath(dir); got != "42.16.3" {
		t.Fatalf("expected 42.16.3, got %q", got)
	}
}

func TestDetectGameVersionFromPath_UnknownWhenNoMarkers(t *testing.T) {
	dir := t.TempDir()
	if got := detectGameVersionFromPath(dir); got != "unknown" {
		t.Fatalf("expected 'unknown', got %q", got)
	}
}

func TestDetectGamePath_EmptyPIDReturnsEmpty(t *testing.T) {
	if got := detectGamePath(""); got != "" {
		t.Fatalf("expected empty for empty pid, got %q", got)
	}
}

func TestDirExists(t *testing.T) {
	dir := t.TempDir()
	if !dirExists(dir) {
		t.Fatal("expected existing temp dir to report true")
	}
	if dirExists(filepath.Join(dir, "nope")) {
		t.Fatal("expected missing dir to report false")
	}
}

func TestIsDirEmpty(t *testing.T) {
	dir := t.TempDir()
	if !isDirEmpty(dir) {
		t.Fatal("expected fresh temp dir to be empty")
	}
	writeFile(t, filepath.Join(dir, "x.txt"), "x")
	if isDirEmpty(dir) {
		t.Fatal("expected dir with a file to be non-empty")
	}
}
