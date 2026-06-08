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
)

// Mod represents a locally discovered mod file.
type Mod struct {
	// ID is derived from the filename without extension.
	ID string
	// Name is the human-readable name (same as ID for Phase A).
	Name string
	// Path is the absolute path to the mod file/directory.
	Path string
	// SHA256 is the hex-encoded SHA256 of the mod content.
	SHA256 string
	// SizeBytes is the byte length of the content.
	SizeBytes int64
	// Version is extracted from a version file if present, otherwise "unknown".
	Version string
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

	return Mod{
		ID:        id,
		Name:      id,
		Path:      path,
		SHA256:    sha256hex,
		SizeBytes: size,
		Version:   version,
	}, nil
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

// hashDir computes a stable SHA256 over all files in a directory tree by
// hashing each file path+content in sorted order.
func hashDir(dir string) (string, int64, error) {
	h := sha256.New()
	var total int64

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		// Include relative path so renames change the hash
		rel, _ := filepath.Rel(dir, path)
		fmt.Fprintf(h, "file:%s\n", rel)

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		n, err := io.Copy(h, f)
		total += n
		return err
	})
	if err != nil {
		return "", 0, fmt.Errorf("hashDir %q: %w", dir, err)
	}
	return hex.EncodeToString(h.Sum(nil)), total, nil
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
