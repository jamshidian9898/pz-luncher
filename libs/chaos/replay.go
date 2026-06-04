package chaos

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ReplayEngine validates deterministic behavior by replaying scenarios
// Same chaos input should produce same output (accounting for non-deterministic factors)
type ReplayEngine struct {
	baselineDir string
}

// NewReplayEngine creates a new replay engine
func NewReplayEngine(baselineDir string) *ReplayEngine {
	return &ReplayEngine{baselineDir: baselineDir}
}

// RecordBaseline saves a scenario result as the baseline for future comparison
func (r *ReplayEngine) RecordBaseline(result *ScenarioResult) error {
	os.MkdirAll(r.baselineDir, 0755)
	
	filename := fmt.Sprintf("baseline-%s.json", result.ScenarioName)
	path := filepath.Join(r.baselineDir, filename)
	
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(path, data, 0644)
}

// ReplayResult compares a new result against the baseline
// Returns true if results are equivalent (accounting for acceptable variance)
func (r *ReplayEngine) ReplayResult(result *ScenarioResult) (*ReplayComparison, error) {
	filename := fmt.Sprintf("baseline-%s.json", result.ScenarioName)
	path := filepath.Join(r.baselineDir, filename)
	
	// Load baseline
	baselineData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("no baseline found for %s: %w", result.ScenarioName, err)
	}
	
	var baseline ScenarioResult
	if err := json.Unmarshal(baselineData, &baseline); err != nil {
		return nil, fmt.Errorf("parse baseline: %w", err)
	}
	
	// Compare
	comparison := r.compareResults(&baseline, result)
	
	return comparison, nil
}

// ReplayComparison holds the comparison between baseline and new result
type ReplayComparison struct {
	ScenarioName     string            `json:"scenarioName"`
	Deterministic    bool              `json:"deterministic"`
	Differences      []string          `json:"differences,omitempty"`
	AcceptableVariance map[string]string `json:"acceptableVariance,omitempty"`
	BaselineDuration int64             `json:"baselineDurationMs"`
	NewDuration      int64             `json:"newDurationMs"`
	DurationDelta    float64           `json:"durationDeltaPercent"`
}

// compareResults checks if two results are equivalent
func (r *ReplayEngine) compareResults(baseline, current *ScenarioResult) *ReplayComparison {
	comp := &ReplayComparison{
		ScenarioName:       baseline.ScenarioName,
		AcceptableVariance: make(map[string]string),
		BaselineDuration:   baseline.DurationMs,
		NewDuration:        current.DurationMs,
	}
	
	// Calculate duration variance (timing is non-deterministic, but should be within range)
	if baseline.DurationMs > 0 {
		delta := float64(current.DurationMs-baseline.DurationMs) / float64(baseline.DurationMs) * 100
		comp.DurationDelta = delta
		
		// Acceptable variance: ±50% of baseline duration
		if delta > 50 || delta < -50 {
			comp.Differences = append(comp.Differences, 
				fmt.Sprintf("duration variance too large: %.1f%%", delta))
		} else {
			comp.AcceptableVariance["duration"] = fmt.Sprintf("%.1f%%", delta)
		}
	}
	
	// Success/failure should be deterministic
	if baseline.Success != current.Success {
		comp.Differences = append(comp.Differences, 
			fmt.Sprintf("success mismatch: baseline=%v, current=%v", baseline.Success, current.Success))
	}
	
	// Package count should match
	if len(baseline.PackageResults) != len(current.PackageResults) {
		comp.Differences = append(comp.Differences,
			fmt.Sprintf("package count mismatch: baseline=%d, current=%d", 
				len(baseline.PackageResults), len(current.PackageResults)))
	}
	
	// Compare individual package results
	for i := 0; i < len(baseline.PackageResults) && i < len(current.PackageResults); i++ {
		basePkg := baseline.PackageResults[i]
		currPkg := current.PackageResults[i]
		
		// Success should match
		if basePkg.Success != currPkg.Success {
			comp.Differences = append(comp.Differences,
				fmt.Sprintf("package %s success mismatch: baseline=%v, current=%v",
					basePkg.PackageID, basePkg.Success, currPkg.Success))
		}
		
		// Attempt count should be similar (allowing for variance in retry count)
		if abs(basePkg.Attempts-currPkg.Attempts) > 2 {
			comp.Differences = append(comp.Differences,
				fmt.Sprintf("package %s attempts vary too much: baseline=%d, current=%d",
					basePkg.PackageID, basePkg.Attempts, currPkg.Attempts))
		}
	}
	
	// Deterministic if no significant differences
	comp.Deterministic = len(comp.Differences) == 0
	
	return comp
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// DeterminismReport summarizes replay tests across multiple scenarios
type DeterminismReport struct {
	Timestamp        time.Time           `json:"timestamp"`
	TotalScenarios   int                 `json:"totalScenarios"`
	Deterministic    int                 `json:"deterministicCount"`
	NonDeterministic int                 `json:"nonDeterministicCount"`
	Comparisons      []ReplayComparison  `json:"comparisons"`
	PassRate         float64             `json:"passRate"`
}

// GenerateReport creates a determinism report for all baselines
func (r *ReplayEngine) GenerateReport(results []*ScenarioResult) (*DeterminismReport, error) {
	report := &DeterminismReport{
		Timestamp:   time.Now(),
		Comparisons: []ReplayComparison{},
	}
	
	for _, result := range results {
		comparison, err := r.ReplayResult(result)
		if err != nil {
			// No baseline for this scenario
			continue
		}
		
		report.TotalScenarios++
		report.Comparisons = append(report.Comparisons, *comparison)
		
		if comparison.Deterministic {
			report.Deterministic++
		} else {
			report.NonDeterministic++
		}
	}
	
	if report.TotalScenarios > 0 {
		report.PassRate = float64(report.Deterministic) / float64(report.TotalScenarios)
	}
	
	return report, nil
}

// SaveReport writes the determinism report to disk
func (r *ReplayEngine) SaveReport(report *DeterminismReport) error {
	filename := fmt.Sprintf("determinism-report-%d.json", report.Timestamp.Unix())
	path := filepath.Join(r.baselineDir, filename)
	
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(path, data, 0644)
}
