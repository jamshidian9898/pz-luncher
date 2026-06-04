package validation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ReliabilityMetrics tracks SLO/SLI metrics over time
type ReliabilityMetrics struct {
	mu sync.RWMutex

	// SLIs (Service Level Indicators)
	TotalExecutions      int64 `json:"totalExecutions"`
	SuccessfulExecutions int64 `json:"successfulExecutions"`
	FailedExecutions     int64 `json:"failedExecutions"`
	RetryableFailures    int64 `json:"retryableFailures"`
	FatalFailures        int64 `json:"fatalFailures"`

	// Timing metrics
	TotalDuration   time.Duration    `json:"totalDuration"`
	MinDuration     time.Duration    `json:"minDuration"`
	MaxDuration     time.Duration    `json:"maxDuration"`
	DurationBuckets map[string]int64 `json:"durationBuckets"`

	// Drift metrics
	TotalComparisons  int64 `json:"totalComparisons"`
	DriftDetections   int64 `json:"driftDetections"`
	OutcomeMismatches int64 `json:"outcomeMismatches"`
	TimingDrifts      int64 `json:"timingDrifts"`

	// Provider metrics
	ProviderUsage   map[string]int64 `json:"providerUsage"`
	ProviderSuccess map[string]int64 `json:"providerSuccess"`

	// Time series
	WindowStart time.Time `json:"windowStart"`
	WindowEnd   time.Time `json:"windowEnd"`
	LastUpdated time.Time `json:"lastUpdated"`
}

// NewReliabilityMetrics creates a new metrics collector
func NewReliabilityMetrics() *ReliabilityMetrics {
	now := time.Now()
	return &ReliabilityMetrics{
		DurationBuckets: make(map[string]int64),
		ProviderUsage:   make(map[string]int64),
		ProviderSuccess: make(map[string]int64),
		WindowStart:     now,
		LastUpdated:     now,
		MinDuration:     time.Hour * 24, // Init to large value
	}
}

// RecordExecution records a single execution result
func (m *ReliabilityMetrics) RecordExecution(success bool, duration time.Duration, err error, provider string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalExecutions++
	m.LastUpdated = time.Now()

	if success {
		m.SuccessfulExecutions++
	} else {
		m.FailedExecutions++
		if isRetryable(err) {
			m.RetryableFailures++
		} else {
			m.FatalFailures++
		}
	}

	// Timing
	m.TotalDuration += duration
	if duration < m.MinDuration {
		m.MinDuration = duration
	}
	if duration > m.MaxDuration {
		m.MaxDuration = duration
	}

	// Bucket
	bucket := durationBucket(duration)
	m.DurationBuckets[bucket]++

	// Provider
	m.ProviderUsage[provider]++
	if success {
		m.ProviderSuccess[provider]++
	}
}

// RecordDrift records a drift detection
// NOTE: Only Outcome Drift counts toward SLO. Timing drift is telemetry only.
func (m *ReliabilityMetrics) RecordDrift(drift *DriftDetection) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalComparisons++
	m.LastUpdated = time.Now()

	if drift.HasDrift {
		switch drift.Type {
		case "outcome":
			m.DriftDetections++ // Only outcome counts for SLO
			m.OutcomeMismatches++
		case "timing":
			m.TimingDrifts++ // Telemetry only, not SLO
		}
	}
}

// SLOReport generates an SLO compliance report
type SLOReport struct {
	GeneratedAt    time.Time     `json:"generatedAt"`
	WindowDuration time.Duration `json:"windowDuration"`

	// SLO: Availability >= 99%
	AvailabilitySLO    float64 `json:"availabilitySLO"` // Target: 0.99
	AvailabilityActual float64 `json:"availabilityActual"`
	AvailabilityMet    bool    `json:"availabilityMet"`

	// SLO: Success Rate >= 95%
	SuccessRateSLO    float64 `json:"successRateSLO"` // Target: 0.95
	SuccessRateActual float64 `json:"successRateActual"`
	SuccessRateMet    bool    `json:"successRateMet"`

	// SLO: Drift Rate < 10%
	DriftSLO        float64 `json:"driftSLO"` // Target: 0.10
	DriftRateActual float64 `json:"driftRateActual"`
	DriftRateMet    bool    `json:"driftRateMet"`

	// SLO: P99 Latency < 60s
	P99LatencySLO    time.Duration `json:"p99LatencySLO"` // Target: 60s
	P99LatencyActual time.Duration `json:"p99LatencyActual"`
	P99LatencyMet    bool          `json:"p99LatencyMet"`

	// Overall status
	AllSLOsMet       bool    `json:"allSLOsMet"`
	ReliabilityScore float64 `json:"reliabilityScore"` // 0-100
}

// GenerateSLOReport creates an SLO compliance report
func (m *ReliabilityMetrics) GenerateSLOReport() *SLOReport {
	m.mu.RLock()
	defer m.mu.RUnlock()

	report := &SLOReport{
		GeneratedAt:     time.Now(),
		WindowDuration:  time.Since(m.WindowStart),
		AvailabilitySLO: 0.99,
		SuccessRateSLO:  0.95,
		DriftSLO:        0.10,
		P99LatencySLO:   60 * time.Second,
	}

	// Calculate actuals
	if m.TotalExecutions > 0 {
		// Availability: (total - fatal) / total
		available := m.TotalExecutions - m.FatalFailures
		report.AvailabilityActual = float64(available) / float64(m.TotalExecutions)

		// Success rate
		report.SuccessRateActual = float64(m.SuccessfulExecutions) / float64(m.TotalExecutions)
	}

	// Drift rate
	if m.TotalComparisons > 0 {
		report.DriftRateActual = float64(m.DriftDetections) / float64(m.TotalComparisons)
	}

	// P99 latency (approximate from buckets)
	report.P99LatencyActual = m.calculateP99()

	// Check SLOs
	report.AvailabilityMet = report.AvailabilityActual >= report.AvailabilitySLO
	report.SuccessRateMet = report.SuccessRateActual >= report.SuccessRateSLO
	report.DriftRateMet = report.DriftRateActual < report.DriftSLO
	report.P99LatencyMet = report.P99LatencyActual <= report.P99LatencySLO

	// Overall
	report.AllSLOsMet = report.AvailabilityMet && report.SuccessRateMet &&
		report.DriftRateMet && report.P99LatencyMet

	// Reliability score (0-100)
	score := 0.0
	if report.AvailabilityMet {
		score += 25
	}
	if report.SuccessRateMet {
		score += 25
	}
	if report.DriftRateMet {
		score += 25
	}
	if report.P99LatencyMet {
		score += 25
	}
	report.ReliabilityScore = score

	return report
}

// calculateP99 estimates P99 from duration buckets
func (m *ReliabilityMetrics) calculateP99() time.Duration {
	total := int64(0)
	for _, count := range m.DurationBuckets {
		total += count
	}

	if total == 0 {
		return 0
	}

	target := int64(float64(total) * 0.99)
	accumulated := int64(0)

	// Check buckets from fast to slow
	buckets := []string{"<100ms", "100-500ms", "500ms-1s", "1-5s", "5-30s", ">30s"}
	for _, bucket := range buckets {
		count := m.DurationBuckets[bucket]
		accumulated += count
		if accumulated >= target {
			// Return upper bound of this bucket
			switch bucket {
			case "<100ms":
				return 100 * time.Millisecond
			case "100-500ms":
				return 500 * time.Millisecond
			case "500ms-1s":
				return time.Second
			case "1-5s":
				return 5 * time.Second
			case "5-30s":
				return 30 * time.Second
			default:
				return 60 * time.Second
			}
		}
	}

	return 60 * time.Second
}

// FailureDistributionReport shows failure breakdown
type FailureDistributionReport struct {
	GeneratedAt          time.Time          `json:"generatedAt"`
	TotalFailures        int64              `json:"totalFailures"`
	RetryablePercent     float64            `json:"retryablePercent"`
	FatalPercent         float64            `json:"fatalPercent"`
	TopFailureTypes      []FailureTypeStat  `json:"topFailureTypes"`
	ProviderFailureRates map[string]float64 `json:"providerFailureRates"`
}

type FailureTypeStat struct {
	Type    string  `json:"type"`
	Count   int64   `json:"count"`
	Percent float64 `json:"percent"`
}

// ProviderReliability shows per-provider stats
type ProviderReliability struct {
	Name        string  `json:"name"`
	Usage       int64   `json:"usage"`
	Successes   int64   `json:"successes"`
	Failures    int64   `json:"failures"`
	SuccessRate float64 `json:"successRate"`
	Reliability string  `json:"reliability"` // "high" | "medium" | "low"
}

// Save persists metrics to disk
func (m *ReliabilityMetrics) Save(path string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.WindowEnd = time.Now()

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}

	os.MkdirAll(filepath.Dir(path), 0755)
	return os.WriteFile(path, data, 0644)
}

// Load reads metrics from disk
func LoadReliabilityMetrics(path string) (*ReliabilityMetrics, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var m ReliabilityMetrics
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	// Re-initialize maps if nil
	if m.DurationBuckets == nil {
		m.DurationBuckets = make(map[string]int64)
	}
	if m.ProviderUsage == nil {
		m.ProviderUsage = make(map[string]int64)
	}
	if m.ProviderSuccess == nil {
		m.ProviderSuccess = make(map[string]int64)
	}

	return &m, nil
}

// isRetryable determines if an error is retryable
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	// Check for retryable error types
	errStr := err.Error()
	return !(errStr == "hash mismatch" || errStr == "context canceled")
}
