package main

// contribute.go implements RFC-0060: Community Upload & Cross-Validation.
//
// These methods are bound to the frontend via Wails and let players
// contribute game client binaries to the Content Registry.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"pzlauncher/libs/settings"
	"pzlauncher/libs/ziputil"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ContributeEntry describes one local game version and its registry trust status.
type ContributeEntry struct {
	GameVersion string `json:"gameVersion"`
	Platform    string `json:"platform"`
	LocalPath   string `json:"localPath"`
	SizeBytes   int64  `json:"sizeBytes"`
	SHA256      string `json:"sha256,omitempty"` // populated after hashing
	TrustLevel  string `json:"trustLevel"`       // "unknown"|"pending"|"verified"|"rejected"
	UploadCount int    `json:"uploadCount"`
	Source      string `json:"source"` // "cache"|"gamePath"
}

// ContributeStatus is returned by GetContributeStatus.
type ContributeStatus struct {
	Entries    []ContributeEntry `json:"entries"`
	BackendURL string            `json:"backendUrl"`
}

// SubmitHashResult mirrors the backend POST .../submit response.
type SubmitHashResult struct {
	Status      string `json:"status"`
	TrustLevel  string `json:"trustLevel"`
	UploadCount int    `json:"uploadCount"`
	UploadURL   string `json:"uploadUrl,omitempty"`
	Error       string `json:"error,omitempty"`
}

// GetContributeStatus scans the local version cache and the user's gamePath
// to build a list of game versions available to contribute.
func (a *App) GetContributeStatus() (*ContributeStatus, error) {
	root := a.ui.getWorkspaceRoot()
	st, _ := settings.Load(root)
	backendURL := ""
	if st != nil {
		backendURL = st.BackendURL
	}
	if backendURL == "" {
		backendURL = "http://localhost:8080"
	}

	var entries []ContributeEntry

	// 1. Scan ~/.pz-launcher/versions/ for already-downloaded versions.
	versionsDir := filepath.Join(root, "versions")
	if dirs, err := os.ReadDir(versionsDir); err == nil {
		for _, d := range dirs {
			if !d.IsDir() {
				continue
			}
			versionDir := filepath.Join(versionsDir, d.Name())
			entry, err := entryFromVersionDir(versionDir, d.Name(), "cache")
			if err != nil {
				continue
			}
			entries = append(entries, entry)
		}
	}

	// 2. Check settings.gamePath as a candidate (user's existing installation).
	if st != nil && st.GamePath != "" {
		if entry, err := entryFromGamePath(st.GamePath); err == nil {
			// Only add if not already present from cache.
			alreadyListed := false
			for _, e := range entries {
				if e.GameVersion == entry.GameVersion {
					alreadyListed = true
					break
				}
			}
			if !alreadyListed {
				entries = append(entries, entry)
			}
		}
	}

	// 3. Enrich entries with registry status from backend.
	for i := range entries {
		enrichFromRegistry(backendURL, &entries[i])
	}

	return &ContributeStatus{Entries: entries, BackendURL: backendURL}, nil
}

// HashResult is returned by HashGameVersion.
type HashResult struct {
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"sizeBytes"`
}

// HashGameVersion computes the SHA256 of the game binary for a given entry.
// It accepts either a single file or a directory. Directories are packed into a
// deterministic zip archive so the same hash is produced for identical content.
// This is a slow operation for large files (~10 GB); progress events are emitted.
func (a *App) HashGameVersion(localPath string) (*HashResult, error) {
	info, err := os.Stat(localPath)
	if err != nil {
		return nil, fmt.Errorf("cannot stat %q: %w", localPath, err)
	}

	if info.IsDir() {
		return a.hashGameDirectory(localPath)
	}
	return a.hashGameFile(localPath, info.Size())
}

func (a *App) hashGameFile(localPath string, total int64) (*HashResult, error) {
	f, err := os.Open(localPath)
	if err != nil {
		return nil, fmt.Errorf("cannot open %q: %w", localPath, err)
	}
	defer f.Close()

	h := sha256.New()
	buf := make([]byte, 4*1024*1024) // 4 MB chunks
	var done int64

	lastEmit := time.Now()
	for {
		n, err := f.Read(buf)
		if n > 0 {
			h.Write(buf[:n])
			done += int64(n)
			if time.Since(lastEmit) > 300*time.Millisecond {
				pct := int(float64(done) / float64(total) * 100)
				a.emitContributeEvent("hashing", map[string]interface{}{
					"percent": pct,
					"done":    done,
					"total":   total,
				})
				lastEmit = time.Now()
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("hash read error: %w", err)
		}
	}
	a.emitContributeEvent("hashing", map[string]interface{}{"percent": 100, "done": done, "total": total})
	return &HashResult{
		SHA256:    hex.EncodeToString(h.Sum(nil)),
		SizeBytes: total,
	}, nil
}

func (a *App) hashGameDirectory(localPath string) (*HashResult, error) {
	h := sha256.New()
	cw := &countingWriter{w: h}

	// Size is not known until the archive is written; emit a streaming progress
	// event based on the number of files processed.
	fileCount, totalFiles, err := countFiles(localPath)
	if err != nil {
		return nil, fmt.Errorf("cannot count directory files: %w", err)
	}
	_ = fileCount

	var processed int
	lastEmit := time.Now()
	emitProgress := func() {
		if time.Since(lastEmit) > 500*time.Millisecond {
			pct := 0
			if totalFiles > 0 {
				pct = int(float64(processed) / float64(totalFiles) * 100)
			}
			a.emitContributeEvent("hashing", map[string]interface{}{
				"percent": pct,
				"done":    processed,
				"total":   totalFiles,
			})
			lastEmit = time.Now()
		}
	}

	if err := ziputil.WriteDeterministic(cw, localPath, func() {
		processed++
		emitProgress()
	}); err != nil {
		return nil, fmt.Errorf("hash archive error: %w", err)
	}
	a.emitContributeEvent("hashing", map[string]interface{}{"percent": 100, "done": processed, "total": totalFiles})
	return &HashResult{
		SHA256:    hex.EncodeToString(h.Sum(nil)),
		SizeBytes: cw.n,
	}, nil
}

// countFiles returns the total number of regular files in a directory tree.
func countFiles(root string) (int, int, error) {
	var count int
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info != nil && !info.IsDir() {
			count++
		}
		return nil
	})
	return count, count, err
}

// countingWriter counts bytes written to an underlying writer.
type countingWriter struct {
	w io.Writer
	n int64
}

func (cw *countingWriter) Write(p []byte) (int, error) {
	n, err := cw.w.Write(p)
	cw.n += int64(n)
	return n, err
}

// SubmitVersionHash calls POST /api/v1/registry/versions/{g}/{v}/{p}/submit.
// It does NOT upload the file — only submits the hash for cross-validation.
func (a *App) SubmitVersionHash(gameID, version, platform, sha256hex string, sizeBytes int64) (*SubmitHashResult, error) {
	root := a.ui.getWorkspaceRoot()
	st, _ := settings.Load(root)
	backendURL := "http://localhost:8080"
	if st != nil && st.BackendURL != "" {
		backendURL = st.BackendURL
	}

	url := fmt.Sprintf("%s/api/v1/registry/versions/%s/%s/%s/submit",
		strings.TrimRight(backendURL, "/"), gameID, version, platform)

	body := fmt.Sprintf(`{"sha256":%q,"sizeBytes":%d}`, sha256hex, sizeBytes)
	resp, err := http.Post(url, "application/json", strings.NewReader(body)) //nolint:noctx
	if err != nil {
		return &SubmitHashResult{Error: err.Error()}, nil
	}
	defer resp.Body.Close()

	var result SubmitHashResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return &SubmitHashResult{Error: "invalid response from backend"}, nil
	}
	return &result, nil
}

// UploadVersionBinary streams the local file to the registry.
// Emits progress events during upload.
func (a *App) UploadVersionBinary(gameID, version, platform, localPath, sha256hex string, sizeBytes int64) error {
	root := a.ui.getWorkspaceRoot()
	st, _ := settings.Load(root)
	backendURL := "http://localhost:8080"
	if st != nil && st.BackendURL != "" {
		backendURL = st.BackendURL
	}

	info, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("cannot stat %q: %w", localPath, err)
	}

	var body io.Reader
	var total int64

	if info.IsDir() {
		if sizeBytes <= 0 {
			return fmt.Errorf("directory upload requires a known archive size; hash the directory first")
		}
		pr, pw := io.Pipe()
		go func() {
			defer pw.Close()
			if err := ziputil.WriteDeterministic(pw, localPath, nil); err != nil {
				_ = pw.CloseWithError(err)
			}
		}()
		body = pr
		total = sizeBytes
	} else {
		f, err := os.Open(localPath)
		if err != nil {
			return fmt.Errorf("cannot open %q: %w", localPath, err)
		}
		defer f.Close()
		body = f
		if sizeBytes > 0 {
			total = sizeBytes
		} else {
			total = info.Size()
		}
	}

	pr := &progressReader{
		r:     body,
		total: total,
		onProgress: func(done int64) {
			pct := 0
			if total > 0 {
				pct = int(float64(done) / float64(total) * 100)
			}
			a.emitContributeEvent("uploading", map[string]interface{}{
				"percent": pct,
				"done":    done,
				"total":   total,
			})
		},
	}

	url := fmt.Sprintf("%s/api/v1/registry/versions/%s/%s/%s",
		strings.TrimRight(backendURL, "/"), gameID, version, platform)

	req, err := http.NewRequest(http.MethodPut, url, pr)
	if err != nil {
		return err
	}
	req.Header.Set("X-Content-SHA256", sha256hex)
	req.ContentLength = total

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload rejected (%d): %s", resp.StatusCode, b)
	}
	a.emitContributeEvent("upload_done", map[string]interface{}{"version": version})
	return nil
}

// --- helpers ---

func (a *App) emitContributeEvent(eventType string, data map[string]interface{}) {
	if a.ui.ctx == nil {
		return
	}
	data["type"] = eventType
	wailsRuntime.EventsEmit(a.ui.ctx, "contribute:event", data)
}

func currentPlatform() string {
	switch runtime.GOOS {
	case "windows":
		if runtime.GOARCH == "amd64" {
			return "windows-x64"
		}
		return "windows-x86"
	case "linux":
		return "linux-x64"
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return "macos-arm64"
		}
		return "macos-x64"
	default:
		return runtime.GOOS + "-" + runtime.GOARCH
	}
}

func entryFromVersionDir(dir, version, source string) (ContributeEntry, error) {
	metaPath := filepath.Join(dir, ".version")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return ContributeEntry{}, err
	}
	var meta struct {
		GameVersion string `json:"gameVersion"`
		Platform    string `json:"platform"`
		SHA256      string `json:"sha256"`
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return ContributeEntry{}, err
	}
	// Estimate directory size
	size, _ := dirSize(dir)
	return ContributeEntry{
		GameVersion: meta.GameVersion,
		Platform:    meta.Platform,
		LocalPath:   dir,
		SizeBytes:   size,
		SHA256:      meta.SHA256,
		TrustLevel:  "unknown",
		Source:      source,
	}, nil
}

func entryFromGamePath(gamePath string) (ContributeEntry, error) {
	if _, err := os.Stat(gamePath); err != nil {
		return ContributeEntry{}, err
	}
	version := detectGameVersion(gamePath)
	size, _ := dirSize(gamePath)
	return ContributeEntry{
		GameVersion: version,
		Platform:    currentPlatform(),
		LocalPath:   gamePath,
		SizeBytes:   size,
		TrustLevel:  "unknown",
		Source:      "gamePath",
	}, nil
}

// detectGameVersion tries several heuristics to determine a game version from a path.
func detectGameVersion(gamePath string) string {
	// 1. Version hint file placed by our launcher.
	if data, err := os.ReadFile(filepath.Join(gamePath, ".pz-version")); err == nil {
		if v := strings.TrimSpace(string(data)); v != "" {
			return v
		}
	}

	// 2. Generic version file inside the directory.
	if data, err := os.ReadFile(filepath.Join(gamePath, "version.txt")); err == nil {
		if v := strings.TrimSpace(string(data)); v != "" {
			return v
		}
	}

	// 3. Extract version from directory names in the path (e.g. ProjectZomboid-42.16.3).
	parts := strings.Split(filepath.ToSlash(gamePath), "/")
	versionRe := regexp.MustCompile(`(?i)(?:ProjectZomboid|PZ|ProjectZomboid64)[-_\s]?v?(\d+(?:\.\d+)*\w*)`)
	for i := len(parts) - 1; i >= 0; i-- {
		if m := versionRe.FindStringSubmatch(parts[i]); len(m) > 1 {
			if v := normalizeVersion(m[1]); v != "" {
				return v
			}
		}
	}

	// 4. Fallback: any path component that looks like a version number.
	fallbackRe := regexp.MustCompile(`(\d+(?:\.\d+)+\w*)`)
	for i := len(parts) - 1; i >= 0; i-- {
		if m := fallbackRe.FindStringSubmatch(parts[i]); len(m) > 1 {
			if v := normalizeVersion(m[1]); v != "" {
				return v
			}
		}
	}

	return "unknown"
}

func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if regexp.MustCompile(`^\d+(\.\d+)*$`).MatchString(v) {
		return v
	}
	return ""
}

func enrichFromRegistry(backendURL string, entry *ContributeEntry) {
	if entry.GameVersion == "unknown" || backendURL == "" {
		return
	}
	url := fmt.Sprintf("%s/api/v1/registry/versions?game=pz&platform=%s",
		strings.TrimRight(backendURL, "/"), entry.Platform)
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return
	}
	defer resp.Body.Close()
	var result struct {
		Versions []struct {
			GameVersion string `json:"gameVersion"`
			TrustLevel  string `json:"trustLevel"`
			UploadCount int    `json:"uploadCount"`
		} `json:"versions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return
	}
	for _, v := range result.Versions {
		if v.GameVersion == entry.GameVersion {
			entry.TrustLevel = v.TrustLevel
			entry.UploadCount = v.UploadCount
			return
		}
	}
}

// progressReader wraps an io.Reader and calls onProgress every 2 MB.
type progressReader struct {
	r          io.Reader
	total      int64
	done       int64
	lastNotify int64
	onProgress func(done int64)
}

func (p *progressReader) Read(buf []byte) (int, error) {
	n, err := p.r.Read(buf)
	p.done += int64(n)
	if p.done-p.lastNotify >= 2*1024*1024 {
		p.onProgress(p.done)
		p.lastNotify = p.done
	}
	return n, err
}
