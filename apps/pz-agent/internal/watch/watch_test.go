package watch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatch_FiresOnFileCreate(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	changed := make(chan struct{}, 1)
	if err := Watch(ctx, dir, func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	}); err != nil {
		t.Fatalf("watch: %v", err)
	}

	// Give the watcher goroutine a moment to register with the OS before
	// generating the event.
	time.Sleep(100 * time.Millisecond)
	if err := os.WriteFile(filepath.Join(dir, "NewMod.zip"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	select {
	case <-changed:
		// onChange fired as expected.
	case <-time.After(Debounce + 3*time.Second):
		t.Fatal("timed out waiting for onChange after file create")
	}
}

func TestWatch_MissingDirDegradesGracefully(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := Watch(ctx, filepath.Join(t.TempDir(), "does-not-exist"), func() {})
	if err != nil {
		t.Fatalf("expected nil error (graceful degradation), got %v", err)
	}
}
