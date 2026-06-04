package session

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// FailureMode defines types of failures that can be injected
type FailureMode int

const (
	FailureNone FailureMode = iota
	FailureNetworkTimeout
	FailureHTTPError
	FailureHashMismatch
	FailureCorruptData
	FailurePartialDownload
	FailureSteamAPIUnavailable
	FailureSteamCMDNotFound
)

// FailureInjector simulates real-world failure scenarios for testing
// This is critical for validating robustness before production use
type FailureInjector struct {
	mu        sync.RWMutex
	enabled   bool
	modes     map[FailureMode]float64 // mode -> probability (0-1)
	delayMin  time.Duration
	delayMax  time.Duration
}

// NewFailureInjector creates a new failure injector (disabled by default)
func NewFailureInjector() *FailureInjector {
	return &FailureInjector{
		enabled:  false,
		modes:    make(map[FailureMode]float64),
		delayMin: 0,
		delayMax: 0,
	}
}

// Enable activates failure injection
func (f *FailureInjector) Enable() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.enabled = true
}

// Disable deactivates failure injection
func (f *FailureInjector) Disable() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.enabled = false
}

// IsEnabled returns current state
func (f *FailureInjector) IsEnabled() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.enabled
}

// SetFailureProbability sets probability for a failure mode (0-1)
func (f *FailureInjector) SetFailureProbability(mode FailureMode, probability float64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.modes[mode] = probability
}

// SetNetworkDelay sets min/max delay for simulating slow networks
func (f *FailureInjector) SetNetworkDelay(min, max time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.delayMin = min
	f.delayMax = max
}

// MaybeFail checks if a failure should be injected and returns appropriate error
func (f *FailureInjector) MaybeFail(ctx context.Context, operation string) error {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if !f.enabled {
		return nil
	}

	// Apply network delay first
	if f.delayMax > 0 {
		delay := f.delayMin + time.Duration(rand.Int63n(int64(f.delayMax-f.delayMin)))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	// Check each failure mode
	for mode, prob := range f.modes {
		if rand.Float64() < prob {
			return f.generateError(mode, operation)
		}
	}

	return nil
}

// generateError creates realistic errors for each failure mode
func (f *FailureInjector) generateError(mode FailureMode, operation string) error {
	switch mode {
	case FailureNetworkTimeout:
		return &FailureError{
			Mode:    mode,
			Message: fmt.Sprintf("network timeout during %s: connection to Steam API timed out after 30s", operation),
			Retryable: true,
		}

	case FailureHTTPError:
		statusCodes := []int{500, 502, 503, 504, 429}
		code := statusCodes[rand.Intn(len(statusCodes))]
		return &FailureError{
			Mode:    mode,
			Message: fmt.Sprintf("http error %d during %s: Steam API temporarily unavailable", code, operation),
			Retryable: code == 429 || code >= 500,
		}

	case FailureHashMismatch:
		return &FailureError{
			Mode:    mode,
			Message: "hash mismatch: downloaded content does not match expected SHA256",
			Retryable: false, // Non-retryable - source is corrupt
		}

	case FailureCorruptData:
		return &FailureError{
			Mode:    mode,
			Message: "corrupt data: unexpected EOF while reading download stream",
			Retryable: true,
		}

	case FailurePartialDownload:
		return &FailureError{
			Mode:    mode,
			Message: "partial download: connection closed before completion (downloaded 45%)",
			Retryable: true,
		}

	case FailureSteamAPIUnavailable:
		return &FailureError{
			Mode:    mode,
			Message: "steam api unavailable: unable to resolve workshop item metadata",
			Retryable: true,
		}

	case FailureSteamCMDNotFound:
		return &FailureError{
			Mode:    mode,
			Message: "steamcmd not found: executable not available for fallback download",
			Retryable: false, // Non-retryable - configuration issue
		}

	default:
		return errors.New("unknown failure mode")
	}
}

// FailureError is an error with metadata about the failure
type FailureError struct {
	Mode      FailureMode
	Message   string
	Retryable bool
}

func (e *FailureError) Error() string {
	return e.Message
}

// IsRetryable returns whether this error is retryable
func (e *FailureError) IsRetryable() bool {
	return e.Retryable
}

// GetActiveModes returns list of currently active failure modes
func (f *FailureInjector) GetActiveModes() []FailureMode {
	f.mu.RLock()
	defer f.mu.RUnlock()

	var active []FailureMode
	for mode, prob := range f.modes {
		if prob > 0 {
			active = append(active, mode)
		}
	}
	return active
}

// PresetChaosMode configures common chaos testing scenario
// Simulates a flaky but not completely broken network
func (f *FailureInjector) PresetChaosMode() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.enabled = true
	f.modes[FailureNetworkTimeout] = 0.1      // 10% timeout
	f.modes[FailureHTTPError] = 0.05          // 5% HTTP errors
	f.modes[FailurePartialDownload] = 0.05    // 5% partial downloads
	f.delayMin = 100 * time.Millisecond
	f.delayMax = 2 * time.Second
}

// PresetCorruptMode configures data corruption testing
// Simulates rare but serious corruption scenarios
func (f *FailureInjector) PresetCorruptMode() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.enabled = true
	f.modes[FailureHashMismatch] = 0.02       // 2% hash mismatch (rare but serious)
	f.modes[FailureCorruptData] = 0.05        // 5% stream corruption
}

// PresetSteamDownMode configures Steam API unavailability
// Simulates when Steam API is completely unavailable
func (f *FailureInjector) PresetSteamDownMode() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.enabled = true
	f.modes[FailureSteamAPIUnavailable] = 1.0 // 100% API failure
	f.modes[FailureSteamCMDNotFound] = 0.0    // But steamcmd might still work
}

// Stats returns current failure injection statistics
func (f *FailureInjector) Stats() map[string]interface{} {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return map[string]interface{}{
		"enabled":   f.enabled,
		"modes":     f.modes,
		"delayMin":  f.delayMin.String(),
		"delayMax":  f.delayMax.String(),
	}
}
