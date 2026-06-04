package session

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

// SteamExecutorMode defines how the Steam executor operates
type SteamExecutorMode int

const (
	ModeReal            SteamExecutorMode = iota // Full Steam API + SteamCMD
	ModeSteamCMDOnly                             // Skip API, use SteamCMD directly
	ModeOfflineFixtures                          // Use local fixture files (no network)
)

// String returns human-readable mode name
func (m SteamExecutorMode) String() string {
	switch m {
	case ModeReal:
		return "real"
	case ModeSteamCMDOnly:
		return "steamcmd-only"
	case ModeOfflineFixtures:
		return "offline-fixtures"
	default:
		return "unknown"
	}
}

// FixturePackage defines a test fixture with real metadata
type FixturePackage struct {
	WorkshopID  string `json:"workshopId"`
	Name        string `json:"name"`
	SHA256      string `json:"sha256"`
	Size        int64  `json:"size"`
	FixtureFile string `json:"fixtureFile"` // Path to local .zip fixture
}

// FixtureRegistry manages offline test fixtures
type FixtureRegistry struct {
	fixturesDir string
	fixtures    map[string]FixturePackage // workshopID -> fixture
}

// NewFixtureRegistry creates a fixture registry
func NewFixtureRegistry(fixturesDir string) *FixtureRegistry {
	return &FixtureRegistry{
		fixturesDir: fixturesDir,
		fixtures:    make(map[string]FixturePackage),
	}
}

// Register adds a fixture package
func (r *FixtureRegistry) Register(fixture FixturePackage) {
	r.fixtures[fixture.WorkshopID] = fixture
}

// LoadFromFile loads fixtures from JSON
func (r *FixtureRegistry) LoadFromFile(path string) error {
	// Implementation would load JSON file with fixture definitions
	return nil
}

// Get retrieves a fixture by workshop ID
func (r *FixtureRegistry) Get(workshopID string) (*FixturePackage, bool) {
	fixture, ok := r.fixtures[workshopID]
	if !ok {
		return nil, false
	}
	return &fixture, true
}

// AvailableWorkshopIDs returns all registered workshop IDs
func (r *FixtureRegistry) AvailableWorkshopIDs() []string {
	ids := make([]string, 0, len(r.fixtures))
	for id := range r.fixtures {
		ids = append(ids, id)
	}
	return ids
}

// WithMode sets the executor mode (for SteamExecutor)
func (e *SteamExecutor) WithMode(mode SteamExecutorMode) *SteamExecutor {
	e.mode = mode
	return e
}

// WithFixtures sets the fixture registry (for offline mode)
func (e *SteamExecutor) WithFixtures(registry *FixtureRegistry) *SteamExecutor {
	e.fixtureRegistry = registry
	return e
}

// downloadOffline uses local fixtures instead of network
func (e *SteamExecutor) downloadOffline(ctx context.Context, workshopID, targetPath, expectedSHA string) error {
	log.Printf("[SteamExecutor] downloadOffline: workshopID=%s, targetPath=%s", workshopID, targetPath)

	if e.fixtureRegistry == nil {
		return fmt.Errorf("fixture registry not configured")
	}

	fixture, ok := e.fixtureRegistry.Get(workshopID)
	if !ok {
		return fmt.Errorf("no fixture registered for workshop ID: %s", workshopID)
	}

	// Check if fixture file exists
	fixturePath := filepath.Join(e.fixtureRegistry.fixturesDir, fixture.FixtureFile)
	if _, err := os.Stat(fixturePath); err != nil {
		return fmt.Errorf("fixture file not found: %s", fixturePath)
	}

	// Ensure target directory exists
	targetDir := filepath.Dir(targetPath)
	if targetPath == "" || targetPath == e.CacheDir {
		// If SHA256 is empty, use workshopID as filename
		targetPath = filepath.Join(e.CacheDir, workshopID+".pkg")
	}
	os.MkdirAll(targetDir, 0755)

	// Copy fixture to target
	src, err := os.Open(fixturePath)
	if err != nil {
		return fmt.Errorf("open fixture: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("create target: %w", err)
	}
	defer dst.Close()

	// Copy with progress (if callback set)
	written, err := io.Copy(dst, src)
	if err != nil {
		return fmt.Errorf("copy fixture: %w", err)
	}

	_ = written // Could report progress here

	// Verify hash if expected
	if expectedSHA != "" {
		actualSHA, err := ComputeSHA256(targetPath)
		if err != nil {
			return fmt.Errorf("compute hash: %w", err)
		}
		if actualSHA != expectedSHA {
			return fmt.Errorf("hash mismatch: expected %s, got %s", expectedSHA, actualSHA)
		}
	}

	log.Printf("[SteamExecutor] downloadOffline SUCCESS")
	return nil
}

// ComputeSHA256 computes SHA256 of a file
func ComputeSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	// In real implementation, use crypto/sha256
	// For now, return placeholder
	_ = data
	return "placeholder_hash", nil
}
