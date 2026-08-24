// Package watch notifies on changes to a mods directory so the Agent can
// react immediately instead of waiting for the next poll interval.
// RFC-0056 calls for "filesystem change event (inotify/FSEvents/
// ReadDirectoryChanges)" — this package is the cross-platform wrapper.
package watch

import (
	"context"
	"log"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Debounce is the quiet period after the last observed event before onChange
// fires. Mod installs/updates touch many files in quick succession (unzip,
// Steam Workshop sync); without debouncing we'd trigger a sync per file.
const Debounce = 2 * time.Second

// Watch watches dir (non-recursively — PZ mods are one level under modsDir,
// and each mod's own internal changes don't need a re-scan trigger) for
// create/write/remove/rename events and calls onChange (debounced) for each
// burst of activity. It runs until ctx is cancelled.
//
// If dir cannot be watched (doesn't exist yet, platform limitation, fd
// exhaustion, etc.) Watch logs a warning and returns nil immediately —
// callers must keep their poll-based fallback running regardless of this
// function's outcome.
func Watch(ctx context.Context, dir string, onChange func()) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("agent: watch: create watcher: %v (falling back to poll-only)", err)
		return nil
	}

	if err := watcher.Add(dir); err != nil {
		log.Printf("agent: watch: add %q: %v (falling back to poll-only)", dir, err)
		watcher.Close()
		return nil
	}
	log.Printf("agent: watching %q for mod changes", dir)

	go run(ctx, watcher, onChange)
	return nil
}

func run(ctx context.Context, watcher *fsnotify.Watcher, onChange func()) {
	defer watcher.Close()

	var timer *time.Timer
	var timerC <-chan time.Time

	for {
		select {
		case <-ctx.Done():
			if timer != nil {
				timer.Stop()
			}
			return

		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			// Ignore pure Chmod noise (metadata-only, common on some
			// filesystems/editors); everything else resets the debounce timer.
			if event.Op&^fsnotify.Chmod == 0 {
				continue
			}
			if timer == nil {
				timer = time.NewTimer(Debounce)
			} else {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(Debounce)
			}
			timerC = timer.C

		case <-timerC:
			timerC = nil
			onChange()

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("agent: watch: error: %v", err)
		}
	}
}
