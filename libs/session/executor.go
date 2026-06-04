package session

import (
	"context"
	"fmt"
	"time"

	"pzlauncher/libs/contracts"
)

// SimpleExecutor is a stub executor for testing and development
// In production, this would integrate with actual download providers
type SimpleExecutor struct {
	CacheDir string
}

// NewSimpleExecutor creates a new simple executor
func NewSimpleExecutor(cacheDir string) *SimpleExecutor {
	return &SimpleExecutor{CacheDir: cacheDir}
}

// Execute processes a single package
// For now, this is a stub that simulates success for any package
func (e *SimpleExecutor) Execute(ctx context.Context, exec *contracts.PackageExecution) (*contracts.PackageExecution, error) {
	// In a real implementation, this would:
	// 1. Look up the provider from ProviderDecision
	// 2. Call the provider's Download method
	// 3. Verify the downloaded content against SHA256
	// 4. Store in cache directory
	// 5. Return updated execution state

	// Simulate work
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(10 * time.Millisecond): // Fast stub
	}

	// Mark as complete (stub behavior)
	exec.State = contracts.PackageStateComplete
	exec.CompletedAt = time.Now()
	exec.CachePath = fmt.Sprintf("%s/%s", e.CacheDir, exec.ProviderDecision.PackageSHA256)

	return exec, nil
}
