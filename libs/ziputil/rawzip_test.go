package ziputil

import (
	"archive/zip"
	"io"
	"testing"
)

// newRawZip writes entries with unvalidated names, for testing Extract's
// path checks against archives our own writer would never produce.
func newRawZip(t *testing.T, w io.Writer, entries map[string]string) *zip.Writer {
	t.Helper()
	zw := zip.NewWriter(w)
	for name, content := range entries {
		f, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Store})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return zw
}
