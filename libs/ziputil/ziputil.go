// Package ziputil provides deterministic zip archiving so that directory
// content can be identified by the SHA256 of its archive bytes.
//
// The same directory tree always produces the same byte stream: entries are
// sorted, timestamps fixed, and no compression is applied (zip.Store). This
// lets the Agent declare sha256(archive) as a mod's content identity, upload
// exactly those bytes, and have the Backend's checksum verification pass.
package ziputil

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Magic is the 4-byte signature of a non-empty zip archive.
var Magic = []byte{'P', 'K', 0x03, 0x04}

// WriteDeterministic packs root into a zip archive with deterministic ordering
// and fixed metadata, so the same directory always produces the same byte stream.
// onFile, if non-nil, is called after each file is added.
func WriteDeterministic(w io.Writer, root string, onFile func()) error {
	zw := zip.NewWriter(w)
	defer zw.Close()

	var entries []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		entries = append(entries, rel)
		return nil
	})
	if err != nil {
		return err
	}

	sort.Strings(entries)
	fixedTime := time.Unix(0, 0)

	for _, rel := range entries {
		full := filepath.Join(root, rel)
		header := &zip.FileHeader{
			Name:     filepath.ToSlash(rel),
			Method:   zip.Store,
			Modified: fixedTime,
		}
		f, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		file, err := os.Open(full)
		if err != nil {
			return err
		}
		_, err = io.Copy(f, file)
		file.Close()
		if err != nil {
			return err
		}
		if onFile != nil {
			onFile()
		}
	}
	return zw.Close()
}

// HashDir returns the SHA256 hex and byte size of the deterministic zip
// archive of root, without materializing the archive.
func HashDir(root string) (string, int64, error) {
	h := sha256.New()
	cw := &countingWriter{w: h}
	if err := WriteDeterministic(cw, root, nil); err != nil {
		return "", 0, fmt.Errorf("ziputil: hash %q: %w", root, err)
	}
	return hex.EncodeToString(h.Sum(nil)), cw.n, nil
}

// StreamDeterministic returns a ReadCloser over the deterministic zip archive
// of root, produced on the fly via a pipe.
func StreamDeterministic(root string) io.ReadCloser {
	pr, pw := io.Pipe()
	go func() {
		pw.CloseWithError(WriteDeterministic(pw, root, nil))
	}()
	return pr
}

// IsZipFile reports whether the file at path starts with the zip magic bytes.
func IsZipFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	var buf [4]byte
	if _, err := io.ReadFull(f, buf[:]); err != nil {
		return false
	}
	return string(buf[:]) == string(Magic)
}

// Extract unpacks the zip archive at src into destDir. Entry paths are
// validated against zip-slip (no escaping destDir).
func Extract(src, destDir string) error {
	zr, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("ziputil: open %q: %w", src, err)
	}
	defer zr.Close()

	cleanDest := filepath.Clean(destDir)
	for _, f := range zr.File {
		target := filepath.Join(cleanDest, filepath.FromSlash(f.Name))
		if !strings.HasPrefix(target, cleanDest+string(os.PathSeparator)) {
			return fmt.Errorf("ziputil: illegal entry path %q", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(target)
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return fmt.Errorf("ziputil: extract %q: %w", f.Name, err)
		}
	}
	return nil
}

type countingWriter struct {
	w io.Writer
	n int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}
