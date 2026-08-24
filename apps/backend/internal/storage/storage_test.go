package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func TestPutGet_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	store, err := NewDiskStore(dir)
	if err != nil {
		t.Fatalf("new disk store: %v", err)
	}

	content := "hello mod content"
	hash := sha256Hex(content)
	if err := store.Put(hash, strings.NewReader(content)); err != nil {
		t.Fatalf("put: %v", err)
	}

	rc, size, err := store.Get(hash)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer rc.Close()
	if size != int64(len(content)) {
		t.Fatalf("expected size %d, got %d", len(content), size)
	}
}

func TestGet_MissingReturnsErrNotFound(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewDiskStore(dir)
	_, _, err := store.Get(sha256Hex("never stored"))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestPut_ChecksumMismatchRejected(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewDiskStore(dir)
	wrongHash := sha256Hex("something else entirely")

	err := store.Put(wrongHash, strings.NewReader("actual content"))
	if err == nil {
		t.Fatal("expected checksum mismatch error")
	}
	if store.Has(wrongHash) {
		t.Fatal("expected mismatched blob not to be stored")
	}
}

func TestHas_ReflectsStoredState(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewDiskStore(dir)
	hash := sha256Hex("x")

	if store.Has(hash) {
		t.Fatal("expected Has to be false before Put")
	}
	if err := store.Put(hash, strings.NewReader("x")); err != nil {
		t.Fatalf("put: %v", err)
	}
	if !store.Has(hash) {
		t.Fatal("expected Has to be true after Put")
	}
}

func TestSize_ReturnsStoredByteLength(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewDiskStore(dir)
	content := "twelve bytes"
	hash := sha256Hex(content)
	_ = store.Put(hash, strings.NewReader(content))

	if got := store.Size(hash); got != int64(len(content)) {
		t.Fatalf("expected size %d, got %d", len(content), got)
	}
}

func TestSize_MissingReturnsZero(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewDiskStore(dir)
	if got := store.Size(sha256Hex("nope")); got != 0 {
		t.Fatalf("expected 0 for missing blob, got %d", got)
	}
}

func TestPut_SurvivesAcrossStoreInstances(t *testing.T) {
	dir := t.TempDir()
	content := "persisted content"
	hash := sha256Hex(content)

	store1, _ := NewDiskStore(dir)
	if err := store1.Put(hash, strings.NewReader(content)); err != nil {
		t.Fatalf("put: %v", err)
	}

	store2, err := NewDiskStore(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if !store2.Has(hash) {
		t.Fatal("expected blob to survive across DiskStore instances (same root dir)")
	}
}

func TestBlobPath_UsesTwoLevelPrefix(t *testing.T) {
	d := &DiskStore{root: "/root"}
	hash := "abcdef0123456789"
	got := d.blobPath(hash)
	want := "/root/ab/abcdef0123456789"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
