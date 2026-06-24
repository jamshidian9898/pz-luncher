package game

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"pzlauncher/libs/contracts"
)

// ProcessTracker keeps track of running game processes
type ProcessTracker struct {
	mu        sync.RWMutex
	process   *os.Process
	startTime time.Time
	running   bool
}

var (
	globalTracker = &ProcessTracker{}
)

// IsGameRunning returns true if the tracked game process is still running.
// The tracker is updated by the goroutine that calls cmd.Wait() after launch.
func IsGameRunning() bool {
	globalTracker.mu.RLock()
	defer globalTracker.mu.RUnlock()
	return globalTracker.running && globalTracker.process != nil
}

// StopGame terminates the running game process.
// The tracker is reset regardless of whether Kill succeeds, so the UI never
// gets stuck believing the game is still running.
func StopGame() error {
	globalTracker.mu.Lock()
	defer globalTracker.mu.Unlock()

	if !globalTracker.running || globalTracker.process == nil {
		return fmt.Errorf("no game process running")
	}

	if err := globalTracker.process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		// Log but still reset state; the process may already be gone or may be a zombie.
		fmt.Fprintf(os.Stderr, "stop game warning: %v\n", err)
	}

	globalTracker.running = false
	globalTracker.process = nil
	return nil
}

// GetGameRuntime returns how long the game has been running
func GetGameRuntime() time.Duration {
	globalTracker.mu.RLock()
	defer globalTracker.mu.RUnlock()

	if !globalTracker.running {
		return 0
	}
	return time.Since(globalTracker.startTime)
}

type simpleFinder struct{}

func NewSimpleFinder() InstallationFinder { return &simpleFinder{} }

// FindInstallation returns a best-effort installation discovery using common paths or environment.
func (s *simpleFinder) FindInstallation() (contracts.GameInstallation, error) {
	// Check settings/env for game path (PZ_GAME_PATH is set by settings.ApplyGamePathEnv)
	if p := os.Getenv("PZ_GAME_PATH"); p != "" {
		return contracts.GameInstallation{Path: p, Version: "unknown", Platform: "local", IsSteamInstall: false}, nil
	}
	// Legacy PZ_PATH fallback
	if p := os.Getenv("PZ_PATH"); p != "" {
		return contracts.GameInstallation{Path: p, Version: "unknown", Platform: "local", IsSteamInstall: false}, nil
	}
	// fallback dummy
	return contracts.GameInstallation{Path: "./pz_executable", Version: "unknown", Platform: "local", IsSteamInstall: false}, nil
}

type simpleLauncher struct{}

func NewSimpleLauncher() GameLauncher { return &simpleLauncher{} }

// Launch actually starts the game executable with the given launch args.
func (l *simpleLauncher) Launch(installation contracts.GameInstallation, request contracts.LaunchRequest) (contracts.LaunchResult, error) {
	profilePath := request.ProfileID
	// if ProfileID looks like a path, use it; otherwise assume profiles/{id}
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		// try profiles/<id>
		alt := filepath.Join("profiles", profilePath)
		if _, err2 := os.Stat(alt); err2 == nil {
			profilePath = alt
		}
	}

	// Determine the game executable based on platform
	var exePath string
	switch runtime.GOOS {
	case "windows":
		// Try 64-bit first, fall back to 32-bit
		exePath = filepath.Join(installation.Path, "ProjectZomboid64.exe")
		if _, err := os.Stat(exePath); os.IsNotExist(err) {
			exePath = filepath.Join(installation.Path, "ProjectZomboid.exe")
		}
	case "darwin":
		exePath = filepath.Join(installation.Path, "ProjectZomboid.app", "Contents", "MacOS", "ProjectZomboid")
	default: // linux
		exePath = filepath.Join(installation.Path, "ProjectZomboid64")
		if _, err := os.Stat(exePath); os.IsNotExist(err) {
			exePath = filepath.Join(installation.Path, "ProjectZomboid")
		}
	}

	// Verify executable exists
	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		return contracts.LaunchResult{Success: false, Error: fmt.Sprintf("Game executable not found at %s", exePath)}, err
	}

	// Build command args: use launch args from manifest if provided
	args := []string{}
	if request.LaunchArgs != "" {
		args = append(args, request.LaunchArgs)
	}

	// Create launch record file for tracking
	fn := filepath.Join(profilePath, "launched.txt")
	f, err := os.Create(fn)
	if err == nil {
		defer f.Close()
		f.WriteString(fmt.Sprintf("launched at %s\ninstallation=%s\nexe=%s\nargs=%s\n", time.Now().Format(time.RFC3339), installation.Path, exePath, request.LaunchArgs))
	}

	// Actually start the game process
	cmd := exec.Command(exePath, args...)
	cmd.Dir = installation.Path

	if err := cmd.Start(); err != nil {
		return contracts.LaunchResult{Success: false, Error: fmt.Sprintf("Failed to start game: %v", err)}, err
	}

	// Track the process
	globalTracker.mu.Lock()
	globalTracker.process = cmd.Process
	globalTracker.startTime = time.Now()
	globalTracker.running = true
	globalTracker.mu.Unlock()

	// Start goroutine to wait for process exit
	go func() {
		cmd.Wait()
		globalTracker.mu.Lock()
		globalTracker.running = false
		globalTracker.process = nil
		globalTracker.mu.Unlock()
	}()

	// Don't wait for process - let it run independently
	return contracts.LaunchResult{Success: true, ProfileID: profilePath, LaunchArgs: request.LaunchArgs}, nil
}
