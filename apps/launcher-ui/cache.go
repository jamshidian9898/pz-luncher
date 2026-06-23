package main

// cache.go implements RFC-0061: Cache Manager.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type CacheEntry struct {
	Type         string `json:"type"` // "version" or "mod"
	Key          string `json:"key"`
	Platform     string `json:"platform,omitempty"`
	GameVersion  string `json:"gameVersion,omitempty"`
	ModID        string `json:"modId,omitempty"`
	SizeBytes    int64  `json:"sizeBytes"`
	DownloadedAt string `json:"downloadedAt"`
	LastUsedAt   string `json:"lastUsedAt"`
	UsedByProfiles []string `json:"usedByProfiles"`
}

type CacheStats struct {
	TotalBytes     int64        `json:"totalBytes"`
	VersionBytes   int64        `json:"versionBytes"`
	ModBytes       int64        `json:"modBytes"`
	Entries        []CacheEntry `json:"entries"`
	DeletableBytes int64        `json:"deletableBytes"`
}

// GetCacheStats returns current cache usage and suggests deletable entries.
func (a *App) GetCacheStats() (*CacheStats, error) {
	root := a.service.getWorkspaceRoot()

	stats := &CacheStats{
		Entries: []CacheEntry{},
	}

	// Scan ~/.pz-launcher/versions/
	versionsDir := filepath.Join(root, "versions")
	if entries, err := os.ReadDir(versionsDir); err == nil {
		for _, d := range entries {
			if !d.IsDir() {
				continue
			}
			versionDir := filepath.Join(versionsDir, d.Name())
			size, _ := dirSize(versionDir)

			entry := CacheEntry{
				Type:        "version",
				Key:         d.Name(),
				GameVersion: d.Name(),
				SizeBytes:   size,
				DownloadedAt: fileModTime(versionDir),
				LastUsedAt:  fileModTime(versionDir),
			}

			// Check which profiles use this version
			entry.UsedByProfiles = versionsUsedBy(root, d.Name())

			stats.Entries = append(stats.Entries, entry)
			stats.VersionBytes += size
		}
	}

	// Scan ~/.pz-launcher/mods/
	modsDir := filepath.Join(root, "mods")
	if entries, err := os.ReadDir(modsDir); err == nil {
		for _, d := range entries {
			if !d.IsDir() {
				continue
			}
			modDir := filepath.Join(modsDir, d.Name())
			size, _ := dirSize(modDir)

			entry := CacheEntry{
				Type:        "mod",
				Key:         d.Name(),
				ModID:       d.Name(),
				SizeBytes:   size,
				DownloadedAt: fileModTime(modDir),
				LastUsedAt:  fileModTime(modDir),
			}

			entry.UsedByProfiles = modsUsedBy(root, d.Name())
			stats.Entries = append(stats.Entries, entry)
			stats.ModBytes += size
		}
	}

	// Calculate deletable entries
	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30).Unix()

	for i := range stats.Entries {
		e := &stats.Entries[i]
		if e.Type == "version" && len(e.UsedByProfiles) == 0 {
			if t := parseTime(e.LastUsedAt); t.Unix() < thirtyDaysAgo {
				stats.DeletableBytes += e.SizeBytes
			}
		}
		if e.Type == "mod" && len(e.UsedByProfiles) == 0 {
			if t := parseTime(e.LastUsedAt); t.Unix() < thirtyDaysAgo {
				stats.DeletableBytes += e.SizeBytes
			}
		}
	}

	stats.TotalBytes = stats.VersionBytes + stats.ModBytes
	return stats, nil
}

// DeleteCacheEntry deletes a version or mod by key.
// Returns error if profiles still reference it.
func (a *App) DeleteCacheEntry(entryType, key string) error {
	root := a.service.getWorkspaceRoot()

	var dir string
	var profiles []string

	if entryType == "version" {
		dir = filepath.Join(root, "versions", key)
		profiles = versionsUsedBy(root, key)
	} else if entryType == "mod" {
		dir = filepath.Join(root, "mods", key)
		profiles = modsUsedBy(root, key)
	} else {
		return fmt.Errorf("unknown entry type: %s", entryType)
	}

	if len(profiles) > 0 {
		return fmt.Errorf("cannot delete: used by profiles: %v", profiles)
	}

	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}

	a.emitCacheEvent("deleted", map[string]interface{}{
		"type": entryType,
		"key":  key,
	})
	return nil
}

// --- helpers ---

func dirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return err
		}
		size += fi.Size()
		return nil
	})
	return size, err
}

func fileModTime(path string) string {
	if fi, err := os.Stat(path); err == nil {
		return fi.ModTime().UTC().Format(time.RFC3339)
	}
	return time.Now().UTC().Format(time.RFC3339)
}

func parseTime(s string) time.Time {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	return time.Now()
}

func (a *App) emitCacheEvent(eventType string, data map[string]interface{}) {
	if a.service.ctx == nil {
		return
	}
	data["type"] = eventType
	a.service.ctx.Value("wailsRuntime").(interface{})
}

func versionsUsedBy(root, version string) []string {
	var used []string
	profilesDir := filepath.Join(root, "profiles")
	if entries, err := os.ReadDir(profilesDir); err == nil {
		for _, d := range entries {
			if !d.IsDir() {
				continue
			}
			// Check if profile symlink points to this version
			versionLink := filepath.Join(profilesDir, d.Name(), ".version")
			if data, err := os.ReadFile(versionLink); err == nil {
				var meta struct {
					GameVersion string `json:"gameVersion"`
				}
				if err := json.Unmarshal(data, &meta); err == nil && meta.GameVersion == version {
					used = append(used, d.Name())
				}
			}
		}
	}
	return used
}

func modsUsedBy(root, modSHA256 string) []string {
	var used []string
	profilesDir := filepath.Join(root, "profiles")
	if entries, err := os.ReadDir(profilesDir); err == nil {
		for _, d := range entries {
			if !d.IsDir() {
				continue
			}
			modsDir := filepath.Join(profilesDir, d.Name(), "mods")
			if entries2, err := os.ReadDir(modsDir); err == nil {
				for _, m := range entries2 {
					if m.Name() == modSHA256 {
						used = append(used, d.Name())
						break
					}
				}
			}
		}
	}
	return used
}
