package providers

import (
	"context"

	"pzlauncher/libs/contracts"
)

type Provider interface {
	Name() string
	Priority() int
	Exists(ctx context.Context, pkg contracts.PackageMetadata) (bool, error)
	Download(ctx context.Context, pkg contracts.PackageMetadata, destination string) error
}
