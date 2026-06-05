// Package join handles POST /api/v1/join/{serverId} logic.
package join

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"pzlauncher/apps/backend/internal/obs"
	"pzlauncher/apps/backend/internal/registry"
	"pzlauncher/apps/backend/internal/storage"
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
// Version is interface{} because the Agent sends a timestamp string while
// the backend-versioned store uses a sequential integer.
type Manifest struct {
	ServerID    string      `json:"serverId"`
	Version     interface{} `json:"version"`
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
	SessionID       string                 `json:"sessionId"`
	TraceID         string                 `json:"traceId"`
	Server          *registry.ServerRecord `json:"server"`
	Manifest        *Manifest              `json:"manifest"`
	ManifestVersion int                    `json:"manifestVersion"` // B4: assigned by manifest.Store
	DownloadPlan    []DownloadItem         `json:"downloadPlan"`
	IssuedAt        string                 `json:"issuedAt"`
}

// Resolver builds a JoinResponse for a given server.
type Resolver struct {
	reg     *registry.Registry
	baseURL string
	store   storage.Store // optional; used to populate sizeBytes in downloadPlan
}

// NewResolver creates a Resolver.
// baseURL is the Backend's own public base URL (e.g. "http://localhost:8080").
// store may be nil during tests; sizeBytes will be omitted when it is.
func NewResolver(reg *registry.Registry, baseURL string, store storage.Store) *Resolver {
	return &Resolver{reg: reg, baseURL: baseURL, store: store}
}

// Resolve builds a JoinResponse.
// ctx should carry a trace ID (set via obs.WithTrace) for log correlation.
func (r *Resolver) Resolve(ctx context.Context, serverID, sessionID, issuedAt string) (*Response, error) {
	traceID := obs.TraceFrom(ctx)
	t0 := time.Now()

	obs.Log(ctx, "join.start",
		"server_id", serverID,
		"session_id", sessionID,
	)

	srv, ok := r.reg.Get(serverID)
	if !ok {
		obs.LogError(ctx, "join.server_not_found", "server_id", serverID)
		return nil, fmt.Errorf("server %q not found", serverID)
	}
	if srv.Status == "offline" {
		obs.LogError(ctx, "join.server_offline", "server_id", serverID)
		return nil, fmt.Errorf("server %q is offline", serverID)
	}

	manifest, err := r.loadManifest(srv)
	if err != nil {
		obs.LogError(ctx, "join.manifest_error", "server_id", serverID, "error", err)
		return nil, fmt.Errorf("manifest unavailable for server %q: %w", serverID, err)
	}

	plan := make([]DownloadItem, 0, len(manifest.Mods))
	for _, mod := range manifest.Mods {
		if mod.SHA256 == "" {
			continue
		}
		size := mod.SizeBytes
		if size == 0 && r.store != nil {
			size = r.store.Size(mod.SHA256)
		}
		plan = append(plan, DownloadItem{
			ModID:     mod.ID,
			SHA256:    mod.SHA256,
			SizeBytes: size,
			URL:       fmt.Sprintf("%s/api/v1/download/%s", r.baseURL, mod.SHA256),
		})
	}

	obs.Log(ctx, "join.complete",
		"server_id", serverID,
		"session_id", sessionID,
		"mod_count", len(plan),
		"duration_ms", time.Since(t0).Milliseconds(),
	)

	// Resolve manifest version from store if available.
	var manifestVersion int
	if vr := r.reg.ManifestStore().Latest(serverID); vr != nil {
		manifestVersion = vr.Version
	}

	return &Response{
		SessionID:       sessionID,
		TraceID:         traceID,
		Server:          srv,
		Manifest:        manifest,
		ManifestVersion: manifestVersion,
		DownloadPlan:    plan,
		IssuedAt:        issuedAt,
	}, nil
}

func (r *Resolver) loadManifest(srv *registry.ServerRecord) (*Manifest, error) {
	// Prefer in-memory manifest pushed by Agent (A5) over on-disk file.
	if live := r.reg.GetManifest(srv.ID); live != nil {
		var m Manifest
		if err := json.Unmarshal(live, &m); err != nil {
			return nil, fmt.Errorf("parse live manifest: %w", err)
		}
		return &m, nil
	}
	// Fall back to disk file (Phase A static fixture).
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
