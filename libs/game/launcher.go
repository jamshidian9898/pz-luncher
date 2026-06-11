package game

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"pzlauncher/libs/contracts"
)

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

	// Don't wait for process - let it run independently
	return contracts.LaunchResult{Success: true, ProfileID: profilePath, LaunchArgs: request.LaunchArgs}, nil
}
