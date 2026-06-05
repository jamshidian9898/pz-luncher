// Package ingest pushes discovered mod content to the Backend store.
// The Agent is a content publisher; the Backend decides what to do with it.
package ingest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"pzlauncher/apps/pz-agent/internal/discover"
	"pzlauncher/apps/pz-agent/internal/retry"
)

// Client publishes content to a Backend instance.
type Client struct {
	backendURL string
	serverID   string
	token      string // X-Agent-Token, empty = unauthenticated
	httpClient *http.Client
}

// NewClient creates an ingest Client.
func NewClient(backendURL, serverID string) *Client {
	return &Client{
		backendURL: backendURL,
		serverID:   serverID,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// WithToken returns a copy of the client with the given auth token set.
func (c *Client) WithToken(token string) *Client {
	copy := *c
	copy.token = token
	return &copy
}

// Register calls POST /api/v1/agents/register and returns the issued token.
// Retried with DefaultPolicy — a backend restart should not prevent enrollment.
func (c *Client) Register(ctx context.Context) (string, error) {
	var token string
	err := retry.DefaultPolicy.Do(ctx, "register", log.Printf, func() error {
		body := []byte(`{"serverId":"` + c.serverID + `"}`)
		url := fmt.Sprintf("%s/api/v1/agents/register", c.backendURL)
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("register: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			if resp.StatusCode >= 400 && resp.StatusCode < 500 {
				return retry.Permanent(fmt.Errorf("register: backend %s: %s", resp.Status, b))
			}
			return fmt.Errorf("register: backend %s: %s", resp.Status, b)
		}
		var result struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return retry.Permanent(fmt.Errorf("register: parse response: %w", err))
		}
		token = result.Token
		return nil
	})
	return token, err
}

// PushBlob uploads a single mod blob to the Backend store.
// Idempotent by design: HEAD check first, PUT only if absent.
// Retried with DefaultPolicy on transient errors (network, 5xx).
func (c *Client) PushBlob(ctx context.Context, mod discover.Mod) error {
	url := fmt.Sprintf("%s/api/v1/blobs/%s", c.backendURL, mod.SHA256)

	// Fast path: HEAD check (not retried — failure just falls through to PUT).
	headReq, _ := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	c.setAuthHeader(headReq)
	if hr, err := c.httpClient.Do(headReq); err == nil {
		hr.Body.Close()
		if hr.StatusCode == http.StatusOK {
			return nil // already present, skip upload
		}
	}

	// Upload with retry. Each attempt reopens the content so the reader is fresh.
	return retry.DefaultPolicy.Do(ctx, "push_blob:"+mod.SHA256[:12], log.Printf, func() error {
		f, err := openModContent(mod)
		if err != nil {
			return retry.Permanent(fmt.Errorf("open %s: %w", mod.ID, err))
		}
		defer f.Close()

		req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, f)
		if err != nil {
			return retry.Permanent(fmt.Errorf("build request: %w", err))
		}
		req.Header.Set("Content-Type", "application/octet-stream")
		c.setAuthHeader(req)
		if mod.SizeBytes > 0 {
			req.ContentLength = mod.SizeBytes
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("PUT blob: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusNoContent {
			return nil
		}
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return retry.Permanent(fmt.Errorf("backend %s: %s", resp.Status, body))
		}
		return fmt.Errorf("backend %s: %s", resp.Status, body)
	})
}

// PublishManifest pushes a server manifest to the Backend.
// The Backend stores it and uses it for future JoinResponse resolution.
// Retried with DefaultPolicy on transient errors.
func (c *Client) PublishManifest(ctx context.Context, mods []discover.Mod, gameVersion, version string) error {
	type modEntry struct {
		ID           string   `json:"id"`
		Name         string   `json:"name"`
		Version      string   `json:"version"`
		SHA256       string   `json:"sha256"`
		SizeBytes    int64    `json:"sizeBytes,omitempty"`
		Dependencies []string `json:"dependencies"`
	}
	type manifest struct {
		ServerID    string      `json:"serverId"`
		Version     string      `json:"version"`
		GameVersion string      `json:"gameVersion"`
		Mods        []modEntry  `json:"mods"`
		LaunchArgs  []string    `json:"launchArgs"`
		Profile     interface{} `json:"profile"`
	}

	entries := make([]modEntry, 0, len(mods))
	for _, m := range mods {
		entries = append(entries, modEntry{
			ID:           m.ID,
			Name:         m.Name,
			Version:      m.Version,
			SHA256:       m.SHA256,
			SizeBytes:    m.SizeBytes,
			Dependencies: []string{},
		})
	}

	body := manifest{
		ServerID:    c.serverID,
		Version:     version,
		GameVersion: gameVersion,
		Mods:        entries,
		LaunchArgs:  []string{},
		Profile:     map[string]string{"profileId": c.serverID},
	}

	data, err := json.Marshal(body)
	if err != nil {
		return retry.Permanent(fmt.Errorf("ingest: marshal manifest: %w", err))
	}

	url := fmt.Sprintf("%s/api/v1/manifests/%s", c.backendURL, c.serverID)
	return retry.DefaultPolicy.Do(ctx, "publish_manifest", log.Printf, func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(data))
		if err != nil {
			return retry.Permanent(fmt.Errorf("build manifest request: %w", err))
		}
		req.Header.Set("Content-Type", "application/json")
		c.setAuthHeader(req)
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("PUT manifest: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
			return nil
		}
		b, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return retry.Permanent(fmt.Errorf("backend manifest %s: %s", resp.Status, b))
		}
		return fmt.Errorf("backend manifest %s: %s", resp.Status, b)
	})
}

// Heartbeat sends a heartbeat to the Backend.
// Uses HeartbeatPolicy (fewer attempts, shorter delay) — a missed heartbeat
// is not catastrophic; the backend will mark the agent degraded/offline.
func (c *Client) Heartbeat(ctx context.Context, modCount int) error {
	type hb struct {
		ServerID  string `json:"serverId"`
		ModCount  int    `json:"modCount"`
		Timestamp string `json:"timestamp"`
	}
	body := hb{
		ServerID:  c.serverID,
		ModCount:  modCount,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	data, _ := json.Marshal(body)
	url := fmt.Sprintf("%s/api/v1/agents/heartbeat", c.backendURL)

	return retry.HeartbeatPolicy.Do(ctx, "heartbeat", log.Printf, func() error {
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		c.setAuthHeader(req)
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("heartbeat: %w", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			return fmt.Errorf("heartbeat: backend returned %s", resp.Status)
		}
		return nil
	})
}

// setAuthHeader attaches the X-Agent-Token header if a token is set.
func (c *Client) setAuthHeader(req *http.Request) {
	if c.token != "" {
		req.Header.Set("X-Agent-Token", c.token)
	}
}

// openModContent returns a ReadCloser over the mod's content.
// For a file mod this is the file itself; for a directory we create a tar stream.
func openModContent(mod discover.Mod) (io.ReadCloser, error) {
	fi, err := os.Stat(mod.Path)
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		return tarDir(mod.Path)
	}
	return os.Open(mod.Path)
}
