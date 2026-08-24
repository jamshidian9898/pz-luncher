// Package discover scans a local mods directory and computes SHA256 for each
// mod file found. This is the only "intelligence" the Agent has about content;
// all orchestration decisions stay in the Backend.
package discover

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"pzlauncher/libs/ziputil"
)

// Mod represents a locally discovered mod file.
type Mod struct {
	// ID is derived from mod.info's "id=" key when present, otherwise the
	// filename without extension.
	ID string
	// Name is from mod.info's "name=" key when present, otherwise same as ID.
	Name string
	// Path is the absolute path to the mod file/directory.
	Path string
	// SHA256 is the hex-encoded SHA256 of the mod content.
	SHA256 string
	// SizeBytes is the byte length of the content.
	SizeBytes int64
	// Version is extracted from a version file/mod.info if present, otherwise "unknown".
	Version string
	// WorkshopID is the Steam Workshop item ID, if this mod was discovered
	// under a Workshop content directory (numeric folder name). Empty for
	// locally-installed (non-Workshop) mods.
	WorkshopID string
}

// Scanner scans a mods directory for installable mod content.
type Scanner struct {
	modsDir string
}

// NewScanner creates a Scanner for the given directory.
func NewScanner(modsDir string) *Scanner {
	return &Scanner{modsDir: modsDir}
}

// Scan walks modsDir and returns a Mod entry for each discovered item.
// Phase A: each top-level file or directory in modsDir is treated as one mod.
func (s *Scanner) Scan() ([]Mod, error) {
	entries, err := os.ReadDir(s.modsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("discover: read %q: %w", s.modsDir, err)
	}

	var mods []Mod
	for _, e := range entries {
		if shouldSkip(e.Name()) {
			continue
		}
		path := filepath.Join(s.modsDir, e.Name())
		mod, err := scanEntry(path, e)
		if err != nil {
			// Non-fatal: log and continue
			continue
		}
		mods = append(mods, mod)
	}
	return mods, nil
}

func scanEntry(path string, e os.DirEntry) (Mod, error) {
	id := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
	name := id

	var sha256hex string
	var size int64
	var err error

	if e.IsDir() {
		sha256hex, size, err = hashDir(path)
	} else {
		sha256hex, size, err = hashFile(path)
	}
	if err != nil {
		return Mod{}, err
	}

	version := readVersionFile(path)

	var workshopID string
	if isNumeric(e.Name()) {
		// Steam Workshop content dirs are named after the numeric workshop
		// item ID (e.g. workshop/content/108600/2470650615/).
		workshopID = e.Name()
	}

	if info := readModInfo(path); info != nil {
		if v, ok := info["id"]; ok && v != "" {
			id = v
		}
		if v, ok := info["name"]; ok && v != "" {
			name = v
		} else {
			name = id
		}
		if v, ok := info["version"]; ok && v != "" {
			version = v
		}
		if v, ok := info["workshopid"]; ok && v != "" {
			workshopID = v
		}
	}

	return Mod{
		ID:         id,
		Name:       name,
		Path:       path,
		SHA256:     sha256hex,
		SizeBytes:  size,
		Version:    version,
		WorkshopID: workshopID,
	}, nil
}

// isNumeric reports whether s consists only of ASCII digits (and is non-empty).
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// readModInfo locates and parses a PZ mod.info file for this entry.
// It checks, in order:
//  1. <path>/mod.info                  — mod installed directly
//  2. <path>/mods/<first>/mod.info     — Workshop layout: content/108600/<id>/mods/<Name>/mod.info
//
// Returns nil if no mod.info is found. Keys are lower-cased.
func readModInfo(path string) map[string]string {
	if info := parseModInfoFile(filepath.Join(path, "mod.info")); info != nil {
		return info
	}

	nested := filepath.Join(path, "mods")
	entries, err := os.ReadDir(nested)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if info := parseModInfoFile(filepath.Join(nested, e.Name(), "mod.info")); info != nil {
			return info
		}
	}
	return nil
}

// parseModInfoFile parses a single mod.info file's "key=value" lines.
// Returns nil if the file does not exist or cannot be read.
func parseModInfoFile(infoPath string) map[string]string {
	data, err := os.ReadFile(infoPath)
	if err != nil {
		return nil
	}
	result := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)
		if key != "" && value != "" {
			// mod.info can repeat keys (e.g. multiple "id=" for legacy multi-mod
			// packs); keep the first occurrence, it's the primary mod entry.
			if _, exists := result[key]; !exists {
				result[key] = value
			}
		}
	}
	return result
}

// hashFile computes the SHA256 and size of a single file.
func hashFile(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

// hashDir identifies a directory mod by the SHA256 and size of its
// deterministic zip archive — the exact bytes the Agent uploads, so the
// Backend's checksum verification matches.
func hashDir(dir string) (string, int64, error) {
	return ziputil.HashDir(dir)
}

// shouldSkip returns true for files/directories that are not actual mods.
func shouldSkip(name string) bool {
	// Hidden files/dirs.
	if strings.HasPrefix(name, ".") {
		return true
	}
	// Steam metadata files (appmanifest_*.acf).
	if strings.HasPrefix(name, "appmanifest_") {
		return true
	}
	// Common non-mod files.
	lower := strings.ToLower(name)
	skip := []string{
		"downloading", "temp", "tmp", "shadercache",
		"steam_autocloud.vdf", "libraryfolders.vdf",
	}
	for _, s := range skip {
		if lower == s {
			return true
		}
	}
	return false
}

// readVersionFile tries to read a version from known version file locations.
func readVersionFile(modPath string) string {
	candidates := []string{
		filepath.Join(modPath, "version.txt"),
		filepath.Join(modPath, "mod.info"),
	}
	for _, c := range candidates {
		data, err := os.ReadFile(c)
		if err == nil {
			line := strings.TrimSpace(strings.SplitN(string(data), "\n", 2)[0])
			if line != "" {
				return line
			}
		}
	}
	return "unknown"
}
