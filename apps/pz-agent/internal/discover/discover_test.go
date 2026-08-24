package discover

import (
	"os"
	"path/filepath"
	"sort"
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

func TestScan_EmptyDirReturnsNil(t *testing.T) {
	dir := t.TempDir()
	mods, err := NewScanner(filepath.Join(dir, "does-not-exist")).Scan()
	if err != nil {
		t.Fatalf("expected no error for missing dir, got %v", err)
	}
	if mods != nil {
		t.Fatalf("expected nil mods, got %v", mods)
	}
}

func TestScan_SkipsNonModEntries(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".hidden"), "x")
	writeFile(t, filepath.Join(dir, "appmanifest_380870.acf"), "x")
	writeFile(t, filepath.Join(dir, "steam_autocloud.vdf"), "x")
	writeFile(t, filepath.Join(dir, "RealMod.zip"), "actual content")

	mods, err := NewScanner(dir).Scan()
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(mods) != 1 {
		t.Fatalf("expected 1 mod after skipping noise, got %d: %+v", len(mods), mods)
	}
	if mods[0].ID != "RealMod" {
		t.Fatalf("expected ID RealMod, got %q", mods[0].ID)
	}
}

func TestScan_FileModHashAndSize(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "SimpleMod.zip"), "hello world")

	mods, err := NewScanner(dir).Scan()
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(mods) != 1 {
		t.Fatalf("expected 1 mod, got %d", len(mods))
	}
	m := mods[0]
	if m.SizeBytes != int64(len("hello world")) {
		t.Fatalf("expected size %d, got %d", len("hello world"), m.SizeBytes)
	}
	if m.SHA256 == "" || len(m.SHA256) != 64 {
		t.Fatalf("expected a 64-char hex sha256, got %q", m.SHA256)
	}
	if m.Version != "unknown" {
		t.Fatalf("expected version 'unknown' with no version file, got %q", m.Version)
	}
}

func TestScan_DirModDeterministicHash(t *testing.T) {
	dir := t.TempDir()
	modDir := filepath.Join(dir, "DirMod")
	writeFile(t, filepath.Join(modDir, "file.txt"), "content")

	mods1, err := NewScanner(dir).Scan()
	if err != nil {
		t.Fatalf("scan 1: %v", err)
	}
	mods2, err := NewScanner(dir).Scan()
	if err != nil {
		t.Fatalf("scan 2: %v", err)
	}
	if len(mods1) != 1 || len(mods2) != 1 {
		t.Fatalf("expected 1 mod in each scan")
	}
	if mods1[0].SHA256 != mods2[0].SHA256 {
		t.Fatalf("expected deterministic hash across scans, got %q vs %q", mods1[0].SHA256, mods2[0].SHA256)
	}
}

func TestScan_ReadsVersionFromVersionTxt(t *testing.T) {
	dir := t.TempDir()
	modDir := filepath.Join(dir, "VersionedMod")
	writeFile(t, filepath.Join(modDir, "version.txt"), "3.2.1\n")
	writeFile(t, filepath.Join(modDir, "file.txt"), "x")

	mods, err := NewScanner(dir).Scan()
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(mods) != 1 {
		t.Fatalf("expected 1 mod, got %d", len(mods))
	}
	if mods[0].Version != "3.2.1" {
		t.Fatalf("expected version 3.2.1, got %q", mods[0].Version)
	}
}

func TestScan_ExtractsIDNameVersionFromModInfo(t *testing.T) {
	dir := t.TempDir()
	modDir := filepath.Join(dir, "SomeFolderName")
	writeFile(t, filepath.Join(modDir, "mod.info"), "id=ActualModID\nname=Actual Mod Name\nversion=5\nposter=someone\n")

	mods, err := NewScanner(dir).Scan()
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(mods) != 1 {
		t.Fatalf("expected 1 mod, got %d", len(mods))
	}
	m := mods[0]
	if m.ID != "ActualModID" {
		t.Fatalf("expected ID from mod.info, got %q", m.ID)
	}
	if m.Name != "Actual Mod Name" {
		t.Fatalf("expected Name from mod.info, got %q", m.Name)
	}
	if m.Version != "5" {
		t.Fatalf("expected Version from mod.info, got %q", m.Version)
	}
}

func TestScan_WorkshopIDFromNumericFolderName(t *testing.T) {
	dir := t.TempDir()
	// Mirrors real Workshop layout: content/108600/<workshopId>/mods/<Name>/mod.info
	workshopDir := filepath.Join(dir, "2470650615")
	nestedModDir := filepath.Join(workshopDir, "mods", "SomeMod")
	writeFile(t, filepath.Join(nestedModDir, "mod.info"), "id=SomeMod\nname=Some Mod\n")

	mods, err := NewScanner(dir).Scan()
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(mods) != 1 {
		t.Fatalf("expected 1 mod, got %d", len(mods))
	}
	m := mods[0]
	if m.WorkshopID != "2470650615" {
		t.Fatalf("expected WorkshopID 2470650615, got %q", m.WorkshopID)
	}
	if m.ID != "SomeMod" {
		t.Fatalf("expected ID from nested mod.info, got %q", m.ID)
	}
}

func TestScan_NonWorkshopModHasNoWorkshopID(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "LocalMod", "file.txt"), "x")

	mods, err := NewScanner(dir).Scan()
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(mods) != 1 {
		t.Fatalf("expected 1 mod, got %d", len(mods))
	}
	if mods[0].WorkshopID != "" {
		t.Fatalf("expected empty WorkshopID for a local mod, got %q", mods[0].WorkshopID)
	}
}

func TestScan_MultipleMods(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "ModA", "file.txt"), "a")
	writeFile(t, filepath.Join(dir, "ModB.zip"), "b")

	mods, err := NewScanner(dir).Scan()
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	ids := make([]string, 0, len(mods))
	for _, m := range mods {
		ids = append(ids, m.ID)
	}
	sort.Strings(ids)
	if len(ids) != 2 || ids[0] != "ModA" || ids[1] != "ModB" {
		t.Fatalf("expected [ModA ModB], got %v", ids)
	}
}

func TestIsNumeric(t *testing.T) {
	cases := map[string]bool{
		"123456789": true,
		"":          false,
		"12a":       false,
		"0":         true,
		"ModName":   false,
	}
	for input, want := range cases {
		if got := isNumeric(input); got != want {
			t.Errorf("isNumeric(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestParseModInfoFile_MissingReturnsNil(t *testing.T) {
	dir := t.TempDir()
	if info := parseModInfoFile(filepath.Join(dir, "mod.info")); info != nil {
		t.Fatalf("expected nil for missing file, got %v", info)
	}
}

func TestParseModInfoFile_SkipsCommentsAndBlankLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mod.info")
	writeFile(t, path, "# comment\n\nid=Foo\nname=Foo Mod\n")
	info := parseModInfoFile(path)
	if info == nil {
		t.Fatal("expected parsed info, got nil")
	}
	if info["id"] != "Foo" || info["name"] != "Foo Mod" {
		t.Fatalf("unexpected parsed info: %v", info)
	}
}
