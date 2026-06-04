package session

import (
	"context"
	"fmt"

	"pzlauncher/libs/contracts"
)

// CompositeExecutor routes package executions to provider-specific executors
// This allows the Session Manager to remain agnostic while supporting multiple providers
// All routing complexity lives here — Session Manager just calls Execute()
type CompositeExecutor struct {
	executors map[string]Executor // provider name -> executor
	fallback  Executor            // used when no specific executor found
}

// NewCompositeExecutor creates a router with provider-specific executors
func NewCompositeExecutor() *CompositeExecutor {
	return &CompositeExecutor{
		executors: make(map[string]Executor),
	}
}

// Register adds an executor for a specific provider
func (c *CompositeExecutor) Register(providerName string, executor Executor) {
	c.executors[providerName] = executor
}

// SetFallback sets the executor used when no specific executor is registered
func (c *CompositeExecutor) SetFallback(executor Executor) {
	c.fallback = executor
}

// Execute routes the package execution to the appropriate provider executor
func (c *CompositeExecutor) Execute(ctx context.Context, exec *contracts.PackageExecution) (*contracts.PackageExecution, error) {
	provider := exec.ProviderDecision.ChosenProvider
	
	// Look up provider-specific executor
	e, ok := c.executors[provider]
	if !ok {
		// No specific executor — use fallback
		if c.fallback != nil {
			return c.fallback.Execute(ctx, exec)
		}
		return nil, fmt.Errorf("no executor registered for provider: %s", provider)
	}
	
	return e.Execute(ctx, exec)
}

// DefaultExecutor creates a pre-configured composite executor for production use
// - LocalCache: uses SimpleExecutor (already cached, just verify)
// - Steam: uses SteamExecutor (real download)
// - Others: uses SimpleExecutor (stub for development)
func DefaultExecutor(cacheDir string) Executor {
	composite := NewCompositeExecutor()
	
	// LocalCache: simple executor (just verifies existing files)
	composite.Register("LocalCache", NewSimpleExecutor(cacheDir))
	
	// Steam: real executor (actual download from Steam Workshop)
	composite.Register("Steam", NewSteamExecutor(cacheDir))
	
	// Fallback: simple executor for any other providers
	composite.SetFallback(NewSimpleExecutor(cacheDir))
	
	return composite
}
