package main

// seed.go — Phase A demo blob seeding.
//
// For Phase A, real mod files don't exist (no SteamCMD). Instead we seed the
// store with a synthetic placeholder blob whose content we know in advance and
// whose SHA256 is pre-computed. The fixture manifests declare the SHA256 of
// these known placeholder blobs so the pipeline can complete end-to-end.
//
// A5 (Agent Minimal) will replace placeholder blobs with real content.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"pzlauncher/apps/backend/internal/storage"
)

// seedDemoBlobs walks fixture manifests and seeds placeholder blobs for any
// mod whose sha256 is not already in the store.
func seedDemoBlobs(store storage.Store, fixturesDir string) {
	manifestsDir := filepath.Join(fixturesDir, "manifests")
	entries, err := os.ReadDir(manifestsDir)
	if err != nil {
		log.Printf("seed: no fixture manifests at %q (%v) — skipping", manifestsDir, err)
		return
	}

	seeded, skipped := 0, 0
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		path := filepath.Join(manifestsDir, e.Name())
		n, s := seedManifest(store, path)
		seeded += n
		skipped += s
	}
	log.Printf("seed: %d blobs seeded, %d already present", seeded, skipped)
}

type seedManifestFile struct {
	Mods []struct {
		ID     string `json:"id"`
		SHA256 string `json:"sha256"`
	} `json:"mods"`
}

func seedManifest(store storage.Store, path string) (seeded, skipped int) {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("seed: read %q: %v", path, err)
		return
	}
	var m seedManifestFile
	if err := json.Unmarshal(data, &m); err != nil {
		log.Printf("seed: parse %q: %v", path, err)
		return
	}
	for _, mod := range m.Mods {
		if mod.SHA256 == "" {
			continue
		}
		if store.Has(mod.SHA256) {
			skipped++
			continue
		}
		// Build a placeholder whose actual SHA256 we compute and then update
		// the manifest to match — but manifests are static files on disk in
		// Phase A. Instead: generate placeholder, compute its hash, skip if it
		// doesn't match the manifest's declared hash (mis-match means the real
		// file needs to come from the Agent).
		blob := makePlaceholderBlob(mod.ID)
		got := computeSHA256(blob)
		if got != mod.SHA256 {
			// Manifest declares a different hash — this blob was hand-authored
			// or came from a real source.  We can't fake it; skip.
			log.Printf("seed: skipping mod=%s — manifest sha256 %s not producible as placeholder (need Agent)", mod.ID, mod.SHA256[:12])
			continue
		}
		if err := store.Put(mod.SHA256, bytes.NewReader(blob)); err != nil {
			log.Printf("seed: put %s: %v", mod.SHA256[:12], err)
			continue
		}
		log.Printf("seed: stored placeholder blob for mod=%s sha256=%s…", mod.ID, mod.SHA256[:12])
		seeded++
	}
	return
}

// makePlaceholderBlob returns the canonical Phase-A placeholder content for a
// given mod ID.  The content is deterministic so its SHA256 is stable.
func makePlaceholderBlob(modID string) []byte {
	return []byte(fmt.Sprintf("PZ-PLACEHOLDER-v1 mod=%s\n", modID))
}

func computeSHA256(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
