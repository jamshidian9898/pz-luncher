package validation

import (
	"sync"
	"time"

	"pzlauncher/libs/contracts"
)

// TelemetryCollector gathers real-world execution metrics
// This bridges the gap between simulation and reality
type TelemetryCollector struct {
	mu          sync.RWMutex
	liveRuns    []ExecutionRecord
	chaosRuns   []ExecutionRecord
	drifts      []DriftDetection
	startedAt   time.Time
}

// ExecutionRecord captures a single execution
 type ExecutionRecord struct {
	Timestamp    time.Time                     `json:"timestamp"`
	PackageID    string                        `json:"packageId"`
	Mode         string                        `json:"mode"`         // "live", "chaos"
	Duration     time.Duration                 `json:"duration"`
	Error        error                         `json:"error,omitempty"`
	State        contracts.PackageExecutionState `json:"state"`
	WorkshopID   string                        `json:"workshopId,omitempty"`
	DownloadSize int64                         `json:"downloadSize,omitempty"`
}

// NewTelemetryCollector creates a new telemetry collector
func NewTelemetryCollector() *TelemetryCollector {
	return &TelemetryCollector{
		liveRuns:  []ExecutionRecord{},
		chaosRuns: []ExecutionRecord{},
		drifts:    []DriftDetection{},
		startedAt: time.Now(),
	}
}

// RecordLiveRun captures a live execution
func (t *TelemetryCollector) RecordLiveRun(packageID string, duration time.Duration, err error, state contracts.PackageExecutionState) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.liveRuns = append(t.liveRuns, ExecutionRecord{
		Timestamp: time.Now(),
		PackageID: packageID,
		Mode:      "live",
		Duration:  duration,
		Error:     err,
		State:     state,
	})
}

// RecordChaosRun captures a chaos execution
func (t *TelemetryCollector) RecordChaosRun(packageID string, duration time.Duration, err error, state contracts.PackageExecutionState) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.chaosRuns = append(t.chaosRuns, ExecutionRecord{
		Timestamp: time.Now(),
		PackageID: packageID,
		Mode:      "chaos",
		Duration:  duration,
		Error:     err,
		State:     state,
	})
}

// RecordDrift captures a detected drift
func (t *TelemetryCollector) RecordDrift(drift *DriftDetection) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.drifts = append(t.drifts, *drift)
}

// TelemetryReport summarizes collected data
type TelemetryReport struct {
	GeneratedAt      time.Time         `json:"generatedAt"`
	SessionDuration  time.Duration     `json:"sessionDuration"`
	LiveRuns         int               `json:"liveRuns"`
	ChaosRuns        int               `json:"chaosRuns"`
	Drifts           int               `json:"drifts"`
	LiveSuccessRate  float64           `json:"liveSuccessRate"`
	ChaosSuccessRate float64           `json:"chaosSuccessRate"`
	AvgLiveDuration  time.Duration     `json:"avgLiveDuration"`
	AvgChaosDuration time.Duration     `json:"avgChaosDuration"`
	LatencyProfile   map[string]int64  `json:"latencyProfile"`
	TopDrifts        []DriftDetection  `json:"topDrifts,omitempty"`
}

// GenerateReport creates a telemetry summary
func (t *TelemetryCollector) GenerateReport() *TelemetryReport {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	report := &TelemetryReport{
		GeneratedAt:     time.Now(),
		SessionDuration: time.Since(t.startedAt),
		LiveRuns:        len(t.liveRuns),
		ChaosRuns:       len(t.chaosRuns),
		Drifts:          len(t.drifts),
		LatencyProfile:  make(map[string]int64),
	}
	
	// Calculate success rates
	liveSuccesses := 0
	var totalLiveDuration time.Duration
	for _, r := range t.liveRuns {
		if r.State == contracts.PackageStateComplete {
			liveSuccesses++
		}
		totalLiveDuration += r.Duration
	}
	if report.LiveRuns > 0 {
		report.LiveSuccessRate = float64(liveSuccesses) / float64(report.LiveRuns)
		report.AvgLiveDuration = totalLiveDuration / time.Duration(report.LiveRuns)
	}
	
	chaosSuccesses := 0
	var totalChaosDuration time.Duration
	for _, r := range t.chaosRuns {
		if r.State == contracts.PackageStateComplete {
			chaosSuccesses++
		}
		totalChaosDuration += r.Duration
	}
	if report.ChaosRuns > 0 {
		report.ChaosSuccessRate = float64(chaosSuccesses) / float64(report.ChaosRuns)
		report.AvgChaosDuration = totalChaosDuration / time.Duration(report.ChaosRuns)
	}
	
	// Build latency profile (bucket durations)
	for _, r := range t.liveRuns {
		bucket := durationBucket(r.Duration)
		report.LatencyProfile[bucket]++
	}
	
	// Include top drifts
	if len(t.drifts) > 0 {
		report.TopDrifts = t.drifts[:min(10, len(t.drifts))]
	}
	
	return report
}

// durationBucket categorizes duration into buckets
func durationBucket(d time.Duration) string {
	ms := d.Milliseconds()
	switch {
	case ms < 100:
		return "<100ms"
	case ms < 500:
		return "100-500ms"
	case ms < 1000:
		return "500ms-1s"
	case ms < 5000:
		return "1-5s"
	case ms < 30000:
		return "5-30s"
	default:
		return ">30s"
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetRawRecords returns all collected records (for detailed analysis)
func (t *TelemetryCollector) GetRawRecords() ([]ExecutionRecord, []ExecutionRecord, []DriftDetection) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	return t.liveRuns, t.chaosRuns, t.drifts
}

// Reset clears all collected data
func (t *TelemetryCollector) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.liveRuns = []ExecutionRecord{}
	t.chaosRuns = []ExecutionRecord{}
	t.drifts = []DriftDetection{}
	t.startedAt = time.Now()
}
