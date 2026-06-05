package ingest

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
)

// tarDir creates an in-memory gzipped tar archive of the directory and returns
// it as a ReadCloser. Phase A: we buffer the entire archive in a pipe so we
// don't need to materialize it on disk.
func tarDir(dir string) (io.ReadCloser, error) {
	pr, pw := io.Pipe()

	go func() {
		gz := gzip.NewWriter(pw)
		tw := tar.NewWriter(gz)

		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(dir, path)
			if rel == "." {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			hdr, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}
			hdr.Name = rel
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
			if !d.IsDir() {
				f, err := os.Open(path)
				if err != nil {
					return err
				}
				defer f.Close()
				_, err = io.Copy(tw, f)
				return err
			}
			return nil
		})

		tw.Close()
		gz.Close()
		pw.CloseWithError(err)
	}()

	return pr, nil
}
