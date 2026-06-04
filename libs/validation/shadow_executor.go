package validation

import (
	"context"
	"fmt"
	"time"

	"pzlauncher/libs/contracts"
	"pzlauncher/libs/session"
)

// ExecutionMode defines how the executor should behave
type ExecutionMode int

const (
	ModeChaos ExecutionMode = iota    // Use failure injection
	ModeLive                        // Use real APIs, no injection
	ModeShadow                      // Run both, compare results
)

// ShadowExecutor wraps a real executor and adds live/chaos comparison capabilities
type ShadowExecutor struct {
	realExecutor  session.Executor
	chaosExecutor session.Executor
	mode          ExecutionMode
	comparator    *DriftComparator
	telemetry     *TelemetryCollector
}

// NewShadowExecutor creates a dual-mode executor for validation
func NewShadowExecutor(real, chaos session.Executor) *ShadowExecutor {
	return &ShadowExecutor{
		realExecutor:  real,
		chaosExecutor: chaos,
		mode:          ModeLive, // Default to live
		comparator:    NewDriftComparator(),
		telemetry:     NewTelemetryCollector(),
	}
}

// WithMode sets the execution mode
func (s *ShadowExecutor) WithMode(mode ExecutionMode) *ShadowExecutor {
	s.mode = mode
	return s
}

// Execute runs according to the configured mode
func (s *ShadowExecutor) Execute(ctx context.Context, exec *contracts.PackageExecution) (*contracts.PackageExecution, error) {
	switch s.mode {
	case ModeChaos:
		return s.executeChaos(ctx, exec)
	case ModeLive:
		return s.executeLive(ctx, exec)
	case ModeShadow:
		return s.executeShadow(ctx, exec)
	default:
		return nil, fmt.Errorf("unknown execution mode: %d", s.mode)
	}
}

// executeChaos runs with failure injection
func (s *ShadowExecutor) executeChaos(ctx context.Context, exec *contracts.PackageExecution) (*contracts.PackageExecution, error) {
	start := time.Now()
	result, err := s.chaosExecutor.Execute(ctx, exec)
	duration := time.Since(start)
	
	// Record telemetry
	if result != nil {
		s.telemetry.RecordChaosRun(result.PackageID, duration, err, result.State)
	}
	
	return result, err
}

// executeLive runs against real APIs
func (s *ShadowExecutor) executeLive(ctx context.Context, exec *contracts.PackageExecution) (*contracts.PackageExecution, error) {
	start := time.Now()
	result, err := s.realExecutor.Execute(ctx, exec)
	duration := time.Since(start)
	
	// Record telemetry
	if result != nil {
		s.telemetry.RecordLiveRun(result.PackageID, duration, err, result.State)
	}
	
	return result, err
}

// executeShadow runs both modes and compares
func (s *ShadowExecutor) executeShadow(ctx context.Context, exec *contracts.PackageExecution) (*contracts.PackageExecution, error) {
	// Clone the execution for chaos run
	chaosExec := cloneExecution(exec)
	
	// Run live (non-blocking for comparison)
	liveStart := time.Now()
	liveResult, liveErr := s.realExecutor.Execute(ctx, exec)
	liveDuration := time.Since(liveStart)
	
	// Run chaos
	chaosStart := time.Now()
	chaosResult, chaosErr := s.chaosExecutor.Execute(ctx, chaosExec)
	chaosDuration := time.Since(chaosStart)
	
	// Record both
	if liveResult != nil {
		s.telemetry.RecordLiveRun(liveResult.PackageID, liveDuration, liveErr, liveResult.State)
	}
	if chaosResult != nil {
		s.telemetry.RecordChaosRun(chaosResult.PackageID, chaosDuration, chaosErr, chaosResult.State)
	}
	
	// Compare and detect drift
	drift := s.comparator.Compare(liveResult, chaosResult, liveDuration, chaosDuration)
	if drift.HasDrift {
		s.telemetry.RecordDrift(drift)
	}
	
	// Return live result as "source of truth"
	return liveResult, liveErr
}

// GetDriftReport returns accumulated drift detections
func (s *ShadowExecutor) GetDriftReport() *DriftReport {
	return s.comparator.GenerateReport()
}

// GetTelemetry returns collected telemetry
func (s *ShadowExecutor) GetTelemetry() *TelemetryReport {
	return s.telemetry.GenerateReport()
}

// cloneExecution creates a deep copy for shadow comparison
func cloneExecution(exec *contracts.PackageExecution) *contracts.PackageExecution {
	return &contracts.PackageExecution{
		PackageID:        exec.PackageID,
		ProviderDecision:   exec.ProviderDecision,
		State:              exec.State,
		StartedAt:          exec.StartedAt,
		CompletedAt:        exec.CompletedAt,
		DurationMs:         exec.DurationMs,
		BytesDownloaded:    exec.BytesDownloaded,
		BytesTotal:         exec.BytesTotal,
		Error:              exec.Error,
		Attempts:           exec.Attempts,
		CachePath:          exec.CachePath,
	}
}
