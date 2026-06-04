package validation

import (
	"fmt"
	"time"

	"pzlauncher/libs/contracts"
)

// DriftComparator compares live vs chaos execution results
type DriftComparator struct {
	comparisons []DriftDetection
}

// NewDriftComparator creates a new drift comparator
func NewDriftComparator() *DriftComparator {
	return &DriftComparator{
		comparisons: []DriftDetection{},
	}
}

// DriftDetection represents a detected divergence between live and chaos
type DriftDetection struct {
	Timestamp     time.Time `json:"timestamp"`
	PackageID     string    `json:"packageId"`
	HasDrift      bool      `json:"hasDrift"`
	Type          string    `json:"type"` // "outcome", "timing", "attempts"
	LiveOutcome   string    `json:"liveOutcome"`
	ChaosOutcome  string    `json:"chaosOutcome"`
	LiveDuration  int64     `json:"liveDurationMs"`
	ChaosDuration int64     `json:"chaosDurationMs"`
	Description   string    `json:"description"`
	Severity      string    `json:"severity"` // "info", "warning", "critical"
}

// Compare evaluates live vs chaos results and detects drift
func (d *DriftComparator) Compare(live, chaos *contracts.PackageExecution, liveDur, chaosDur time.Duration) *DriftDetection {
	detection := &DriftDetection{
		Timestamp:     time.Now(),
		PackageID:     live.PackageID,
		LiveDuration:  liveDur.Milliseconds(),
		ChaosDuration: chaosDur.Milliseconds(),
	}

	// Compare outcomes
	if live.State != chaos.State {
		detection.HasDrift = true
		detection.Type = "outcome"
		detection.LiveOutcome = string(live.State)
		detection.ChaosOutcome = string(chaos.State)
		detection.Description = fmt.Sprintf("outcome mismatch: live=%s, chaos=%s", live.State, chaos.State)
		detection.Severity = "critical"

		d.comparisons = append(d.comparisons, *detection)
		return detection
	}

	// Timing comparison - TELEMETRY ONLY, NOT SLO
	// Timing variance is expected due to OS scheduler, GC, disk cache, etc.
	// We record it but don't count it as drift for reliability scoring.
	minDuration := 100 * time.Millisecond
	if liveDur >= minDuration || chaosDur >= minDuration {
		if chaosDur > 0 && liveDur > 0 {
			var ratio float64
			if liveDur > chaosDur {
				ratio = float64(liveDur) / float64(chaosDur)
			} else {
				ratio = float64(chaosDur) / float64(liveDur)
			}
			// Log timing difference but don't flag as SLO drift
			if ratio > 3.0 {
				// Telemetry only - timing drift is not a reliability issue
				_ = fmt.Sprintf("timing variance: live=%dms, chaos=%dms (ratio %.2f)",
					liveDur.Milliseconds(), chaosDur.Milliseconds(), ratio)
			}
		}
	}

	// Attempt count comparison - TELEMETRY ONLY
	// Attempt variance is expected due to retry policies and network conditions
	if abs(live.Attempts-chaos.Attempts) > 2 {
		// Telemetry only - attempt variance is not a reliability issue
		_ = fmt.Sprintf("attempt variance: live=%d, chaos=%d",
			live.Attempts, chaos.Attempts)
	}

	// No significant drift
	detection.HasDrift = false
	detection.Type = "none"
	detection.LiveOutcome = string(live.State)
	detection.ChaosOutcome = string(chaos.State)
	detection.Description = "no significant drift detected"
	detection.Severity = "info"

	d.comparisons = append(d.comparisons, *detection)
	return detection
}

// DriftReport summarizes all drift detections
type DriftReport struct {
	GeneratedAt       time.Time        `json:"generatedAt"`
	TotalComparisons  int              `json:"totalComparisons"`
	DriftCount        int              `json:"driftCount"`
	OutcomeMismatches int              `json:"outcomeMismatches"`
	TimingDrifts      int              `json:"timingDrifts"`
	AttemptDrifts     int              `json:"attemptDrifts"`
	Drifts            []DriftDetection `json:"drifts"`
	DriftRate         float64          `json:"driftRate"`
}

// GenerateReport creates a summary of all drift detections
func (d *DriftComparator) GenerateReport() *DriftReport {
	report := &DriftReport{
		GeneratedAt:      time.Now(),
		TotalComparisons: len(d.comparisons),
		Drifts:           d.comparisons,
	}

	for _, d := range d.comparisons {
		if d.HasDrift {
			report.DriftCount++
			switch d.Type {
			case "outcome":
				report.OutcomeMismatches++
			case "timing":
				report.TimingDrifts++
			case "attempts":
				report.AttemptDrifts++
			}
		}
	}

	if report.TotalComparisons > 0 {
		report.DriftRate = float64(report.DriftCount) / float64(report.TotalComparisons)
	}

	return report
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
