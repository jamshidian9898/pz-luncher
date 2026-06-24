package main

// versionselect.go implements RFC-0062: Game Version Selection.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"pzlauncher/libs/settings"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type VersionSource struct {
	Type        string `json:"type"` // "registry" | "agent" | "hoster"
	URL         string `json:"url"`
	TrustLevel  string `json:"trustLevel"`
	Description string `json:"description"`
}

type VersionCandidate struct {
	GameVersion     string           `json:"gameVersion"`
	Platform        string           `json:"platform"`
	SizeBytes       int64            `json:"sizeBytes"`
	TrustLevel      string           `json:"trustLevel"`
	AvailableSources []VersionSource `json:"availableSources"`
	IsLocal         bool             `json:"isLocal"`
	LocalPath       string           `json:"localPath,omitempty"`
}

type VersionSelector struct {
	Required     string              `json:"required"` // version required by server
	LocalVersion string              `json:"localVersion,omitempty"` // user currently has
	Candidates   []VersionCandidate  `json:"candidates"` // available to download
	NeedDownload bool                `json:"needDownload"`
	AutoSelected *VersionSource      `json:"autoSelected,omitempty"` // first choice
}

// GetVersionSelector queries the backend registry and local cache
// to determine what game versions are available and which to download.
func (a *App) GetVersionSelector(requiredVersion string) (*VersionSelector, error) {
	root := a.ui.getWorkspaceRoot()
	backendURL := a.getBackendURL()

	sel := &VersionSelector{
		Required:   requiredVersion,
		Candidates: []VersionCandidate{},
	}

	// Check local cache
	localVersion := a.detectLocalVersion(root, requiredVersion)
	sel.LocalVersion = localVersion

	// If we already have the required version locally, no download needed
	if localVersion == requiredVersion {
		sel.NeedDownload = false
		return sel, nil
	}

	sel.NeedDownload = true

	// Query registry for available versions
	url := fmt.Sprintf("%s/api/v1/registry/catalog?game=pz", strings.TrimRight(backendURL, "/"))
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return sel, nil // soft error — no registry available
	}
	defer resp.Body.Close()

	var catalogResp struct {
		Versions []struct {
			GameVersion string `json:"gameVersion"`
			Platform    string `json:"platform"`
			SizeBytes   int64  `json:"sizeBytes"`
			TrustLevel  string `json:"trustLevel"`
		} `json:"versions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&catalogResp); err != nil {
		return sel, nil // soft error
	}

	// Find all candidates for the required version
	for _, v := range catalogResp.Versions {
		if v.GameVersion != requiredVersion {
			continue
		}

		candidate := VersionCandidate{
			GameVersion:     v.GameVersion,
			Platform:        v.Platform,
			SizeBytes:       v.SizeBytes,
			TrustLevel:      v.TrustLevel,
			AvailableSources: []VersionSource{},
		}

		// Registry is always an option
		candidate.AvailableSources = append(candidate.AvailableSources, VersionSource{
			Type:        "registry",
			URL:         fmt.Sprintf("%s/api/v1/registry/versions/pz/%s/%s", strings.TrimRight(backendURL, "/"), v.GameVersion, v.Platform),
			TrustLevel:  v.TrustLevel,
			Description: "PZ Registry (verified)",
		})

		// TODO: Query backend for agent URLs (RFC-0053 enrollment)
		// For now, registry is the only available source

		sel.Candidates = append(sel.Candidates, candidate)

		// Auto-select first candidate's first source
		if sel.AutoSelected == nil && len(candidate.AvailableSources) > 0 {
			sel.AutoSelected = &candidate.AvailableSources[0]
		}
	}

	return sel, nil
}

// ConfirmVersionDownload initiates the download from a selected source.
func (a *App) ConfirmVersionDownload(gameVersion, platform, sourceURL string) error {
	// Emit event to frontend that download is starting
	a.emitVersionEvent("download_start", map[string]interface{}{
		"version": gameVersion,
		"url":     sourceURL,
	})
	return nil
}

// --- helpers ---

func (a *App) getBackendURL() string {
	root := a.ui.getWorkspaceRoot()
	st, _ := settings.Load(root)
	if st != nil && st.BackendURL != "" {
		return st.BackendURL
	}
	return "http://localhost:8080"
}

func (a *App) detectLocalVersion(root, required string) string {
	versionDir := filepath.Join(root, "versions", required)
	if _, err := os.Stat(versionDir); err == nil {
		return required
	}
	return ""
}

func (a *App) emitVersionEvent(eventType string, data map[string]interface{}) {
	if a.ui.ctx == nil {
		return
	}
	data["type"] = eventType
	wailsRuntime.EventsEmit(a.ui.ctx, "version:event", data)
}
