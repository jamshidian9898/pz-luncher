package pipeline

// join_v2.go — v2.0.0 join path (RFC-0050/A3).
// The Launcher calls POST /join on the Backend and receives a JoinResponse.
// This file implements the pipeline stage that converts a JoinResponse into
// a ready profile without touching manifest resolution, provider selection,
// or SteamCMD. Those are Backend concerns.

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pzlauncher/libs/contracts"
	"pzlauncher/libs/game"
	"pzlauncher/libs/profile"
)

// BackendModEntry is one item in JoinResponse.downloadPlan.
type BackendModEntry struct {
	ModID     string `json:"modId"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"sizeBytes,omitempty"`
	URL       string `json:"url"`
}

// BackendManifest is the manifest section of JoinResponse.
type BackendManifest struct {
	ServerID    string            `json:"serverId"`
	Version     string            `json:"version"`
	GameVersion string            `json:"gameVersion"`
	Mods        []backendModMeta  `json:"mods"`
	LaunchArgs  []string          `json:"launchArgs"`
	Profile     backendProfile    `json:"profile"`
}

type backendModMeta struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	SHA256       string   `json:"sha256"`
	Dependencies []string `json:"dependencies"`
}

type backendProfile struct {
	ProfileID string `json:"profileId,omitempty"`
}

// BackendJoinResponse is the canonical JoinResponse (RFC-0055).
type BackendJoinResponse struct {
	SessionID    string            `json:"sessionId"`
	Manifest     BackendManifest   `json:"manifest"`
	DownloadPlan []BackendModEntry `json:"downloadPlan"`
	IssuedAt     string            `json:"issuedAt"`
}

// RunJoinFromBackend runs the v2.0.0 join pipeline using a JoinResponse
// from the Backend. It skips manifest resolution and provider selection;
// the Backend has already resolved those. The Launcher only downloads,
// verifies, installs, and prepares the profile.
func (s *Service) RunJoinFromBackend(ctx context.Context, jr BackendJoinResponse, emit Emitter) (*JoinResult, error) {
	sessionID := jr.SessionID
	serverID := jr.Manifest.ServerID
	if sessionID == "" {
		sessionID = fmt.Sprintf("session-%d", time.Now().Unix())
	}

	emit(Event{Type: "session.start", SessionID: sessionID, Metadata: map[string]interface{}{"serverId": serverID}})
	emit(Event{Type: "manifest.loaded", SessionID: sessionID, Metadata: map[string]interface{}{
		"serverId": serverID, "version": jr.Manifest.Version,
	}})

	emit(Event{Type: "mod.resolve.start", SessionID: sessionID})
	emit(Event{Type: "mod.resolve.complete", SessionID: sessionID, Metadata: map[string]interface{}{
		"modCount": len(jr.DownloadPlan),
	}})

	_ = s.writeJoinTrace(serverID, sessionID, "JoinRequest", "ok", map[string]interface{}{
		"version": jr.Manifest.Version, "modCount": len(jr.DownloadPlan),
	})

	// Download each mod in the plan (cache-check + HTTP download + SHA256 verify)
	total := len(jr.DownloadPlan)
	for i, item := range jr.DownloadPlan {
		if item.SHA256 == "" || item.URL == "" {
			continue
		}
		emit(Event{Type: "download.start", SessionID: sessionID, PackageID: item.ModID})

		destPath := filepath.Join(s.cfg.CacheDir, item.SHA256)
		if _, err := os.Stat(destPath); err == nil {
			// Already in local cache — skip download
			emit(Event{
				Type: "download.progress", SessionID: sessionID, PackageID: item.ModID,
				Progress: &Progress{Current: int64(i + 1), Total: int64(total), Percent: percent(i+1, total)},
			})
			emit(Event{Type: "download.complete", SessionID: sessionID, PackageID: item.ModID})
			continue
		}

		if err := s.downloadAndCache(ctx, item, destPath, emit, sessionID, i, total); err != nil {
			return nil, s.fail(emit, sessionID, "PIPELINE_DOWNLOAD", fmt.Errorf("mod %s: %w", item.ModID, err))
		}
		emit(Event{Type: "download.complete", SessionID: sessionID, PackageID: item.ModID})
	}

	// Build profile from cached blobs
	profileID := jr.Manifest.Profile.ProfileID
	if profileID == "" {
		profileID = serverID
	}

	resolved := downloadPlanToResolved(jr.DownloadPlan)
	pb := profile.NewProfileBuilder(s.cfg.ProfilesDir)
	profilePath, err := pb.Prepare(profileID, serverID, resolved, s.cfg.CacheDir)
	if err != nil {
		return nil, s.fail(emit, sessionID, "PIPELINE_PROFILE", err)
	}

	_ = s.writeJoinTrace(serverID, sessionID, "Ready", "ok", nil)
	emit(Event{Type: "install.complete", SessionID: sessionID})
	emit(Event{Type: "session.complete", SessionID: sessionID, Metadata: map[string]interface{}{
		"ready": true, "profilePath": profilePath,
	}})

	// Store enough launch info for LaunchServer
	launchArgs := strings.Join(jr.Manifest.LaunchArgs, " ")
	_ = launchArgs // passed via manifest snapshot below
	_ = s.writeLaunchManifest(profilePath, jr)

	return &JoinResult{
		SessionID:   sessionID,
		ProfilePath: profilePath,
		Ready:       true,
		// Manifest field intentionally nil — v2 Launcher doesn't own the manifest
	}, nil
}

// LaunchFromBackend launches the game after a v2 join.
// It reads launch args from the manifest snapshot written during join.
func (s *Service) LaunchFromBackend(ctx context.Context, serverID, profilePath string, jr BackendJoinResponse, emit Emitter) error {
	emit(Event{Type: "launch.started", SessionID: serverID, Metadata: map[string]interface{}{"serverId": serverID}})

	finder := game.NewSimpleFinder()
	inst, err := finder.FindInstallation()
	if err != nil {
		emit(Event{Type: "launch.failed", SessionID: serverID, Error: err.Error()})
		return err
	}

	launcher := game.NewSimpleLauncher()
	req := contracts.LaunchRequest{
		ServerID:   serverID,
		ProfileID:  profilePath,
		ManifestID: jr.Manifest.ServerID + "-v" + jr.Manifest.Version,
		LaunchArgs: strings.Join(jr.Manifest.LaunchArgs, " "),
	}
	res, err := launcher.Launch(inst, req)
	if err != nil || !res.Success {
		msg := res.Error
		if err != nil {
			msg = err.Error()
		}
		emit(Event{Type: "launch.failed", SessionID: serverID, Error: msg})
		return fmt.Errorf("launch: %s", msg)
	}
	emit(Event{Type: "launch.exited", SessionID: serverID, Metadata: map[string]interface{}{"success": true}})
	return nil
}

// downloadAndCache fetches a blob from url, streams to a temp file, verifies
// SHA256, then moves it to the final cache path.
func (s *Service) downloadAndCache(ctx context.Context, item BackendModEntry, destPath string, emit Emitter, sessionID string, idx, total int) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	tmpPath := destPath + ".tmp"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, item.URL, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %s", resp.Status)
	}

	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create tmp: %w", err)
	}

	written, sha256hex, err := copyAndHash(resp.Body, f, func(n int64) {
		p := percent(idx+1, total)
		if item.SizeBytes > 0 {
			p = int(n * 100 / item.SizeBytes)
		}
		emit(Event{
			Type: "download.progress", SessionID: sessionID, PackageID: item.ModID,
			Progress: &Progress{Current: n, Total: item.SizeBytes, Percent: p},
		})
	})
	f.Close()
	if err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("stream: %w", err)
	}

	if item.SHA256 != "" && sha256hex != item.SHA256 {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("DOWNLOAD_CHECKSUM_MISMATCH: got %s want %s (wrote %d bytes)", sha256hex, item.SHA256, written)
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

func downloadPlanToResolved(plan []BackendModEntry) []contracts.ResolvedPackage {
	out := make([]contracts.ResolvedPackage, 0, len(plan))
	for _, item := range plan {
		out = append(out, contracts.ResolvedPackage{
			ID:          item.ModID,
			SHA256:      item.SHA256,
			Size:        item.SizeBytes,
			DownloadURL: item.URL,
		})
	}
	return out
}

func (s *Service) writeLaunchManifest(profilePath string, jr BackendJoinResponse) error {
	type launchManifest struct {
		ServerID   string   `json:"serverId"`
		Version    string   `json:"version"`
		LaunchArgs []string `json:"launchArgs"`
		IssuedAt   string   `json:"issuedAt"`
	}
	lm := launchManifest{
		ServerID:   jr.Manifest.ServerID,
		Version:    jr.Manifest.Version,
		LaunchArgs: jr.Manifest.LaunchArgs,
		IssuedAt:   jr.IssuedAt,
	}
	data, err := jsonMarshalIndent(lm)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(profilePath, "launch-manifest.json"), data, 0o644)
}

func percent(done, total int) int {
	if total == 0 {
		return 100
	}
	return done * 100 / total
}
