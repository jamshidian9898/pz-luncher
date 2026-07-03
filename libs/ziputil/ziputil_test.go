package ziputil

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func makeTree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"mod.info":          "name=TestMod\nid=TestMod\n",
		"media/lua/a.lua":   `print("hello")`,
		"media/maps/b.data": "binary-ish content",
	}
	for rel, content := range files {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestHashDirIsDeterministic(t *testing.T) {
	dir := makeTree(t)

	h1, n1, err := HashDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	h2, n2, err := HashDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 || n1 != n2 {
		t.Fatalf("hash not stable: (%s,%d) vs (%s,%d)", h1, n1, h2, n2)
	}
	if n1 == 0 {
		t.Fatal("archive size is zero")
	}
}

func TestHashMatchesStreamedBytes(t *testing.T) {
	dir := makeTree(t)

	declared, size, err := HashDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	rc := StreamDeterministic(dir)
	defer rc.Close()
	h := sha256.New()
	n, err := io.Copy(h, rc)
	if err != nil {
		t.Fatal(err)
	}
	got := hex.EncodeToString(h.Sum(nil))

	if got != declared {
		t.Fatalf("streamed bytes hash %s != declared %s", got, declared)
	}
	if n != size {
		t.Fatalf("streamed %d bytes, declared size %d", n, size)
	}
}

func TestExtractRoundTrip(t *testing.T) {
	dir := makeTree(t)

	var buf bytes.Buffer
	if err := WriteDeterministic(&buf, dir, nil); err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(t.TempDir(), "mod.zip")
	if err := os.WriteFile(zipPath, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if !IsZipFile(zipPath) {
		t.Fatal("IsZipFile = false for a zip archive")
	}

	out := t.TempDir()
	if err := Extract(zipPath, out); err != nil {
		t.Fatal(err)
	}

	want, err := os.ReadFile(filepath.Join(dir, "media/lua/a.lua"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(out, "media/lua/a.lua"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(want, got) {
		t.Fatalf("extracted content differs: %q vs %q", got, want)
	}
}

func TestExtractRejectsZipSlip(t *testing.T) {
	// Hand-build an archive with an escaping path.
	var buf bytes.Buffer
	zw := newRawZip(t, &buf, map[string]string{"../evil.txt": "pwned"})
	_ = zw
	zipPath := filepath.Join(t.TempDir(), "evil.zip")
	if err := os.WriteFile(zipPath, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Extract(zipPath, t.TempDir()); err == nil {
		t.Fatal("expected zip-slip rejection, got nil error")
	}
}
