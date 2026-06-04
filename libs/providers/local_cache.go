package providers

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"pzlauncher/libs/contracts"
)

// LocalCacheProvider checks for package blobs in a local cache directory.
type LocalCacheProvider struct {
	CacheDir string // e.g. ./cache/sha256
}

func NewLocalCacheProvider(cacheDir string) *LocalCacheProvider {
	return &LocalCacheProvider{CacheDir: cacheDir}
}

func (l *LocalCacheProvider) Name() string  { return "LocalCache" }
func (l *LocalCacheProvider) Priority() int { return 10 }

func (l *LocalCacheProvider) Exists(ctx context.Context, pkg contracts.PackageMetadata) (bool, error) {
	path := filepath.Join(l.CacheDir, pkg.SHA256)
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (l *LocalCacheProvider) Download(ctx context.Context, pkg contracts.PackageMetadata, destination string) error {
	src := filepath.Join(l.CacheDir, pkg.SHA256)
	fsrc, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer fsrc.Close()
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	fdst, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("create dest: %w", err)
	}
	defer fdst.Close()
	_, err = io.Copy(fdst, fsrc)
	if err != nil {
		return fmt.Errorf("copy: %w", err)
	}
	return nil
}
