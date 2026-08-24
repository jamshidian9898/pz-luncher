// Package storage implements the Backend Content-Addressable Storage (RFC-0053/A4).
//
// Design:
//   - Store is a pure interface — future backends (S3, R2, Agent proxy) implement it
//     without any Launcher changes.
//   - DiskStore lays blobs out as: <root>/<sha256[:2]>/<sha256>
//     This two-level prefix tree avoids directory entry explosion at scale.
//   - All content is addressed by lowercase hex SHA256.
package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ErrNotFound is returned by Get when the blob is not in the store.
var ErrNotFound = errors.New("blob not found")

// BlobMeta describes who uploaded a blob and when (operator-facing metadata).
type BlobMeta struct {
	SourceServer string    `json:"sourceServer,omitempty"`
	UploadedAt   time.Time `json:"uploadedAt,omitempty"`
}

// BlobInfo is one entry in a store listing.
type BlobInfo struct {
	SHA256     string `json:"sha256"`
	SizeBytes  int64  `json:"sizeBytes"`
	SourceServer string `json:"sourceServer,omitempty"`
	FirstSeenAt  time.Time `json:"firstSeenAt,omitempty"`
}

// Store is the content-addressable blob store interface.
// Every method is keyed by the hex-encoded SHA256 of the content.
type Store interface {
	// Get opens a blob for streaming. Caller must close the returned ReadCloser.
	// Returns (nil, 0, ErrNotFound) when the blob is absent.
	Get(sha256hex string) (io.ReadCloser, int64, error)

	// Put stores a blob. The implementation MUST verify the sha256 after writing
	// and return an error if it doesn't match.
	Put(sha256hex string, r io.Reader) error

	// Has reports whether the blob exists without opening it.
	Has(sha256hex string) bool

	// Size returns the stored byte size of a blob, or 0 if absent.
	Size(sha256hex string) int64

	// Annotate records operator-facing metadata for a blob. Best-effort:
	// implementations may ignore it; missing metadata must never break ingestion.
	Annotate(sha256hex string, meta BlobMeta) error

	// List returns every blob in the store with whatever metadata is known.
	List() ([]BlobInfo, error)
}

// DiskStore is the Phase-A implementation: plain files on disk plus an
// index.json sidecar for operator metadata.
type DiskStore struct {
	root string

	mu    sync.Mutex
	index map[string]BlobMeta // sha256 → metadata
}

// NewDiskStore creates (or opens) a DiskStore at the given root directory.
func NewDiskStore(root string) (*DiskStore, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("storage: create root %q: %w", root, err)
	}
	d := &DiskStore{root: root, index: map[string]BlobMeta{}}
	d.loadIndex()
	return d, nil
}

func (d *DiskStore) indexPath() string {
	return filepath.Join(d.root, "index.json")
}

// loadIndex reads the metadata sidecar. Missing or corrupt index is not fatal —
// the store still works, it just lacks operator metadata.
func (d *DiskStore) loadIndex() {
	data, err := os.ReadFile(d.indexPath())
	if err != nil {
		return
	}
	var idx map[string]BlobMeta
	if json.Unmarshal(data, &idx) == nil {
		d.index = idx
	}
}

// saveIndex atomically writes the sidecar. Caller must hold d.mu.
func (d *DiskStore) saveIndex() error {
	data, err := json.MarshalIndent(d.index, "", "  ")
	if err != nil {
		return err
	}
	tmp := d.indexPath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, d.indexPath())
}

func (d *DiskStore) blobPath(sha256hex string) string {
	if len(sha256hex) < 4 {
		return filepath.Join(d.root, sha256hex)
	}
	return filepath.Join(d.root, sha256hex[:2], sha256hex)
}

// Get opens the blob for reading. Returns ErrNotFound if absent.
func (d *DiskStore) Get(sha256hex string) (io.ReadCloser, int64, error) {
	p := d.blobPath(sha256hex)
	fi, err := os.Stat(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, 0, ErrNotFound
	}
	if err != nil {
		return nil, 0, fmt.Errorf("storage: stat %q: %w", sha256hex, err)
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, 0, fmt.Errorf("storage: open %q: %w", sha256hex, err)
	}
	return f, fi.Size(), nil
}

// Put writes a blob, verifies its SHA256, and moves it to the permanent path.
// If the computed hash does not match sha256hex the data is discarded and an
// error is returned.
func (d *DiskStore) Put(sha256hex string, r io.Reader) error {
	dest := d.blobPath(sha256hex)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("storage: mkdir: %w", err)
	}

	tmp := dest + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("storage: create tmp: %w", err)
	}

	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(f, h), r); err != nil {
		f.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("storage: write: %w", err)
	}
	f.Close()

	got := hex.EncodeToString(h.Sum(nil))
	if got != sha256hex {
		_ = os.Remove(tmp)
		return fmt.Errorf("storage: CHECKSUM_MISMATCH: got %s want %s", got, sha256hex)
	}

	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("storage: rename: %w", err)
	}
	return nil
}

// Has reports whether the blob exists.
func (d *DiskStore) Has(sha256hex string) bool {
	_, err := os.Stat(d.blobPath(sha256hex))
	return err == nil
}

// Size returns the stored byte size, or 0 if absent.
func (d *DiskStore) Size(sha256hex string) int64 {
	fi, err := os.Stat(d.blobPath(sha256hex))
	if err != nil {
		return 0
	}
	return fi.Size()
}

// Annotate records operator metadata for a blob (best-effort).
func (d *DiskStore) Annotate(sha256hex string, meta BlobMeta) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if prev, ok := d.index[sha256hex]; ok {
		if meta.SourceServer == "" {
			meta.SourceServer = prev.SourceServer
		}
		if meta.UploadedAt.IsZero() {
			meta.UploadedAt = prev.UploadedAt
		}
	}
	d.index[sha256hex] = meta
	return d.saveIndex()
}

// List returns every blob on disk merged with any known metadata,
// sorted by SHA256 for stable output.
func (d *DiskStore) List() ([]BlobInfo, error) {
	d.mu.Lock()
	idx := make(map[string]BlobMeta, len(d.index))
	for k, v := range d.index {
		idx[k] = v
	}
	d.mu.Unlock()

	var out []BlobInfo
	err := filepath.WalkDir(d.root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !entry.Type().IsRegular() {
			return nil
		}
		name := entry.Name()
		if len(name) != 64 || filepath.Dir(path) == d.root {
			return nil // skip index.json, tmp files, and non-blob entries
		}
		for _, c := range name {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				return nil // not a hex sha256 name
			}
		}
		info := BlobInfo{SHA256: name, SizeBytes: d.Size(name)}
		if meta, ok := idx[name]; ok {
			info.SourceServer = meta.SourceServer
			info.FirstSeenAt = meta.UploadedAt
		}
		out = append(out, info)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("storage: list: %w", err)
	}
	return out, nil
}
