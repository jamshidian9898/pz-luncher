package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetVersionSelector_AlreadyLocal(t *testing.T) {
	tmpdir := t.TempDir()
	app := &App{ui: &UIService{workspaceRoot: tmpdir}}

	// Create local version 42.16
	versionDir := filepath.Join(tmpdir, "versions", "42.16")
	os.MkdirAll(versionDir, 0o755)

	selector, err := app.GetVersionSelector("42.16")
	if err != nil {
		t.Fatalf("GetVersionSelector failed: %v", err)
	}

	if selector.NeedDownload {
		t.Error("expected needDownload=false when version exists locally")
	}
	if selector.LocalVersion != "42.16" {
		t.Errorf("expected localVersion=42.16, got %s", selector.LocalVersion)
	}
}

func TestGetVersionSelector_MissingVersion(t *testing.T) {
	tmpdir := t.TempDir()
	app := &App{ui: &UIService{workspaceRoot: tmpdir}}

	selector, err := app.GetVersionSelector("42.16")
	if err != nil {
		t.Fatalf("GetVersionSelector failed: %v", err)
	}

	if !selector.NeedDownload {
		t.Error("expected needDownload=true when version is missing locally")
	}
	if selector.LocalVersion != "" {
		t.Errorf("expected localVersion='', got %s", selector.LocalVersion)
	}

	// No registry available, so candidates should be empty
	if len(selector.Candidates) != 0 {
		t.Errorf("expected 0 candidates (no registry), got %d", len(selector.Candidates))
	}
}
