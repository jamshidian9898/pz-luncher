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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ErrNotFound is returned by Get when the blob is not in the store.
var ErrNotFound = errors.New("blob not found")

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
}

// DiskStore is the Phase-A implementation: plain files on disk.
type DiskStore struct {
	root string
}

// NewDiskStore creates (or opens) a DiskStore at the given root directory.
func NewDiskStore(root string) (*DiskStore, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("storage: create root %q: %w", root, err)
	}
	return &DiskStore{root: root}, nil
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
