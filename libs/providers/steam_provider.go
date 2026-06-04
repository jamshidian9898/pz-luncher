package providers

import (
	"context"
	"errors"

	"pzlauncher/libs/contracts"
)

// SteamProvider is a placeholder provider for Steam Workshop content.
type SteamProvider struct{}

func NewSteamProvider() *SteamProvider { return &SteamProvider{} }

func (s *SteamProvider) Name() string  { return "Steam" }
func (s *SteamProvider) Priority() int { return 1 }

func (s *SteamProvider) Exists(ctx context.Context, pkg contracts.PackageMetadata) (bool, error) {
	// Stub: report false; real implementation should query Steam APIs or local cache
	return false, nil
}

func (s *SteamProvider) Download(ctx context.Context, pkg contracts.PackageMetadata, destination string) error {
	return errors.New("SteamProvider.Download not implemented")
}
