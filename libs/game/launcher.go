package game

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"pzlauncher/libs/contracts"
)

type simpleFinder struct{}

func NewSimpleFinder() InstallationFinder { return &simpleFinder{} }

// FindInstallation returns a best-effort installation discovery using common paths or environment.
func (s *simpleFinder) FindInstallation() (contracts.GameInstallation, error) {
	// Minimal, non-invasive discovery: check env PZ_PATH, else return a dummy installation
	if p := os.Getenv("PZ_PATH"); p != "" {
		return contracts.GameInstallation{Path: p, Version: "unknown", Platform: "local", IsSteamInstall: false}, nil
	}
	// fallback dummy
	return contracts.GameInstallation{Path: "./pz_executable", Version: "unknown", Platform: "local", IsSteamInstall: false}, nil
}

type simpleLauncher struct{}

func NewSimpleLauncher() GameLauncher { return &simpleLauncher{} }

// Launch simulates launching the game by recording a launched file in the profile.
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
	fn := filepath.Join(profilePath, "launched.txt")
	f, err := os.Create(fn)
	if err != nil {
		return contracts.LaunchResult{Success: false, Error: fmt.Sprintf("create launch record: %v", err)}, err
	}
	defer f.Close()
	f.WriteString(fmt.Sprintf("launched at %s\ninstallation=%s\nargs=%s\n", time.Now().Format(time.RFC3339), installation.Path, request.LaunchArgs))
	return contracts.LaunchResult{Success: true, ProfileID: profilePath, LaunchArgs: request.LaunchArgs}, nil
}
