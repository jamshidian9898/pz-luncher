package fixtures

import (
	"fmt"
	"os"
	"path/filepath"

	"pzlauncher/libs/contracts"
)

// SeedCache writes demo blobs for the provided manifest into cacheDir for the
// requested seedIDs. It uses deterministic content: <modID>-demo. Returns nil
// if successful.
func SeedCache(manifest contracts.Manifest, cacheDir string, seedIDs []string) error {
	want := map[string]bool{}
	for _, id := range seedIDs {
		want[id] = true
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return fmt.Errorf("mkdir cacheDir: %w", err)
	}
	for _, m := range manifest.Mods {
		if !want[m.ID] {
			continue
		}
		blobPath := filepath.Join(cacheDir, m.SHA256)
		// If file already exists, skip
		if _, err := os.Stat(blobPath); err == nil {
			continue
		}
		content := []byte(m.ID + "-demo")
		if err := os.WriteFile(blobPath, content, 0o644); err != nil {
			return fmt.Errorf("write blob %s: %w", blobPath, err)
		}
	}
	return nil
}
