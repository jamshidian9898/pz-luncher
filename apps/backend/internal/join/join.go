// Package join handles POST /api/v1/join/{serverId} logic.
package join

import (
	"encoding/json"
	"fmt"
	"os"

	"pzlauncher/apps/backend/internal/registry"
)

// ModEntry mirrors RFC-0030 ModEntry (subset needed for JoinResponse).
type ModEntry struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	SHA256       string   `json:"sha256"`
	SizeBytes    int64    `json:"sizeBytes,omitempty"`
	WorkshopID   string   `json:"workshopId,omitempty"`
	Dependencies []string `json:"dependencies"`
	Optional     bool     `json:"optional,omitempty"`
}

// Manifest mirrors RFC-0030 ServerManifest.
type Manifest struct {
	ServerID    string      `json:"serverId"`
	Version     string      `json:"version"`
	GameVersion string      `json:"gameVersion"`
	Mods        []ModEntry  `json:"mods"`
	LaunchArgs  []string    `json:"launchArgs"`
	Profile     interface{} `json:"profile"`
}

// DownloadItem is one entry in the download plan.
type DownloadItem struct {
	ModID     string `json:"modId"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"sizeBytes,omitempty"`
	URL       string `json:"url"`
}

// Response is the canonical JoinResponse (RFC-0055).
type Response struct {
	SessionID    string         `json:"sessionId"`
	Server       *registry.ServerRecord `json:"server"`
	Manifest     *Manifest      `json:"manifest"`
	DownloadPlan []DownloadItem `json:"downloadPlan"`
	IssuedAt     string         `json:"issuedAt"`
}

// Resolver builds a JoinResponse for a given server.
type Resolver struct {
	reg        *registry.Registry
	baseURL    string
}

// NewResolver creates a Resolver.
// baseURL is the Backend's own public base URL (e.g. "http://localhost:8080").
func NewResolver(reg *registry.Registry, baseURL string) *Resolver {
	return &Resolver{reg: reg, baseURL: baseURL}
}

// Resolve builds a JoinResponse. For Phase A the manifest is read from disk
// via ServerRecord.ManifestPath. downloadPlan URLs point at GET /api/v1/download/{sha256}.
func (r *Resolver) Resolve(serverID, sessionID, issuedAt string) (*Response, error) {
	srv, ok := r.reg.Get(serverID)
	if !ok {
		return nil, fmt.Errorf("server %q not found", serverID)
	}
	if srv.Status == "offline" {
		return nil, fmt.Errorf("server %q is offline", serverID)
	}

	manifest, err := r.loadManifest(srv)
	if err != nil {
		return nil, fmt.Errorf("manifest unavailable for server %q: %w", serverID, err)
	}

	plan := make([]DownloadItem, 0, len(manifest.Mods))
	for _, mod := range manifest.Mods {
		if mod.SHA256 == "" {
			continue
		}
		plan = append(plan, DownloadItem{
			ModID:     mod.ID,
			SHA256:    mod.SHA256,
			SizeBytes: mod.SizeBytes,
			URL:       fmt.Sprintf("%s/api/v1/download/%s", r.baseURL, mod.SHA256),
		})
	}

	return &Response{
		SessionID:    sessionID,
		Server:       srv,
		Manifest:     manifest,
		DownloadPlan: plan,
		IssuedAt:     issuedAt,
	}, nil
}

func (r *Resolver) loadManifest(srv *registry.ServerRecord) (*Manifest, error) {
	if srv.ManifestPath == "" {
		return nil, fmt.Errorf("no manifestPath configured")
	}
	data, err := os.ReadFile(srv.ManifestPath)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return &m, nil
}
