package chaos

import (
	"encoding/json"
	"fmt"
	"time"

	"pzlauncher/libs/contracts"
	"pzlauncher/libs/session"
)

// Scenario represents a chaos test case with controlled failure injection
type Scenario struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Packages    []TestPackage     `json:"packages"`
	Failures    []FailureConfig   `json:"failures"`
	Expectation ExpectationConfig `json:"expectation"`
}

// TestPackage represents a package to download in the test
type TestPackage struct {
	PackageID     string `json:"packageId"`
	WorkshopID    string `json:"workshopId"`    // Steam Workshop ID
	SHA256        string `json:"sha256"`        // Expected hash (for deterministic tests)
	ShouldSucceed bool   `json:"shouldSucceed"` // Expected outcome
}

// FailureConfig configures when and how to inject failures
type FailureConfig struct {
	TargetPackage  string  `json:"targetPackage"`  // "" = all packages
	FailureMode    string  `json:"failureMode"`    // "timeout", "http_error", "hash_mismatch", etc.
	Probability    float64 `json:"probability"`    // 0-1
	TriggerAttempt int     `json:"triggerAttempt"` // Which attempt to trigger on (0=any)
}

// ExpectationConfig defines expected test outcomes
type ExpectationConfig struct {
	TotalSuccesses int   `json:"totalSuccesses"` // How many packages should succeed
	TotalFailures  int   `json:"totalFailures"`  // How many should fail
	MaxDuration    int64 `json:"maxDurationMs"`  // Test must complete within
	AllowPartial   bool  `json:"allowPartial"`   // Is partial success acceptable?
}

// ScenarioResult captures the outcome of a chaos test
type ScenarioResult struct {
	ScenarioName   string                 `json:"scenarioName"`
	StartTime      time.Time              `json:"startTime"`
	EndTime        time.Time              `json:"endTime"`
	DurationMs     int64                  `json:"durationMs"`
	Success        bool                   `json:"success"`
	PackageResults []PackageResult        `json:"packageResults"`
	Events         []ChaosEvent           `json:"events"`
	Stats          map[string]interface{} `json:"stats"`
}

// PackageResult tracks outcome for a single package
type PackageResult struct {
	PackageID   string                            `json:"packageId"`
	ExpectedSHA string                            `json:"expectedSha256"`
	ActualSHA   string                            `json:"actualSha256,omitempty"`
	Success     bool                              `json:"success"`
	Attempts    int                               `json:"attempts"`
	DurationMs  int64                             `json:"durationMs"`
	States      []contracts.PackageExecutionState `json:"states"`
	Errors      []string                          `json:"errors,omitempty"`
}

// ChaosEvent records significant events during test execution
type ChaosEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"` // "injected_failure", "retry", "state_change"
	PackageID string    `json:"packageId"`
	Details   string    `json:"details"`
}

// NewScenario creates a basic chaos test scenario
func NewScenario(name, description string) *Scenario {
	return &Scenario{
		Name:        name,
		Description: description,
		Packages:    []TestPackage{},
		Failures:    []FailureConfig{},
		Expectation: ExpectationConfig{
			MaxDuration:  300000, // 5 minutes default
			AllowPartial: false,
		},
	}
}

// AddPackage adds a test package to the scenario
func (s *Scenario) AddPackage(pkg TestPackage) *Scenario {
	s.Packages = append(s.Packages, pkg)
	return s
}

// AddFailure adds a failure injection config
func (s *Scenario) AddFailure(config FailureConfig) *Scenario {
	s.Failures = append(s.Failures, config)
	return s
}

// SetExpectation sets the expected outcome
func (s *Scenario) SetExpectation(exp ExpectationConfig) *Scenario {
	s.Expectation = exp
	return s
}

// Validate checks if the scenario is well-formed
func (s *Scenario) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("scenario must have a name")
	}
	if len(s.Packages) == 0 {
		return fmt.Errorf("scenario must have at least one package")
	}
	for _, f := range s.Failures {
		if f.Probability < 0 || f.Probability > 1 {
			return fmt.Errorf("failure probability must be 0-1, got %f", f.Probability)
		}
	}
	return nil
}

// ToJSON serializes the scenario
func (s *Scenario) ToJSON() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}

// FromJSON deserializes a scenario
func ScenarioFromJSON(data []byte) (*Scenario, error) {
	var s Scenario
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// PresetScenarios returns common chaos test scenarios
func PresetScenarios() []*Scenario {
	return []*Scenario{
		PresetFlakyNetwork(),
		PresetSteamAPIDown(),
		PresetHashMismatch(),
		PresetPartialDownload(),
		PresetRetryExhaustion(),
	}
}

// PresetFlakyNetwork simulates unreliable network conditions
func PresetFlakyNetwork() *Scenario {
	return NewScenario("flaky_network", "30% timeout, 20% delay - validates recovery").
		AddPackage(TestPackage{
			PackageID:     "test-mod-a",
			WorkshopID:    "123456789",
			SHA256:        "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			ShouldSucceed: true,
		}).
		AddFailure(FailureConfig{
			FailureMode:    "timeout",
			Probability:    0.3,
			TriggerAttempt: 0,
		}).
		AddFailure(FailureConfig{
			FailureMode:    "http_error",
			Probability:    0.2,
			TriggerAttempt: 0,
		}).
		SetExpectation(ExpectationConfig{
			TotalSuccesses: 1,
			TotalFailures:  0,
			MaxDuration:    60000,
			AllowPartial:   true,
		})
}

// PresetSteamAPIDown simulates complete Steam API failure (tests SteamCMD fallback)
func PresetSteamAPIDown() *Scenario {
	return NewScenario("steam_api_down", "100% Steam API failure - tests fallback to SteamCMD").
		AddPackage(TestPackage{
			PackageID:     "test-mod-b",
			WorkshopID:    "987654321",
			SHA256:        "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			ShouldSucceed: true,
		}).
		AddFailure(FailureConfig{
			TargetPackage:  "test-mod-b",
			FailureMode:    "steam_api_unavailable",
			Probability:    1.0,
			TriggerAttempt: 0,
		}).
		SetExpectation(ExpectationConfig{
			TotalSuccesses: 1,
			TotalFailures:  0,
			MaxDuration:    120000,
			AllowPartial:   false,
		})
}

// PresetHashMismatch simulates data corruption (non-retryable failure)
func PresetHashMismatch() *Scenario {
	return NewScenario("hash_mismatch", "100% hash mismatch - validates fail-fast behavior").
		AddPackage(TestPackage{
			PackageID:     "test-mod-c",
			WorkshopID:    "111111111",
			SHA256:        "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			ShouldSucceed: false, // Expected to fail
		}).
		AddFailure(FailureConfig{
			TargetPackage:  "test-mod-c",
			FailureMode:    "hash_mismatch",
			Probability:    1.0,
			TriggerAttempt: 1,
		}).
		SetExpectation(ExpectationConfig{
			TotalSuccesses: 0,
			TotalFailures:  1,
			MaxDuration:    30000,
			AllowPartial:   true, // Partial is ok because this is expected failure
		})
}

// PresetPartialDownload simulates connection drops mid-download
func PresetPartialDownload() *Scenario {
	return NewScenario("partial_download", "50% partial download - tests resume/retry").
		AddPackage(TestPackage{
			PackageID:     "test-mod-d",
			WorkshopID:    "222222222",
			SHA256:        "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
			ShouldSucceed: true,
		}).
		AddFailure(FailureConfig{
			FailureMode:    "partial_download",
			Probability:    0.5,
			TriggerAttempt: 0,
		}).
		SetExpectation(ExpectationConfig{
			TotalSuccesses: 1,
			TotalFailures:  0,
			MaxDuration:    120000,
			AllowPartial:   true,
		})
}

// PresetRetryExhaustion tests behavior when retry budget is depleted
func PresetRetryExhaustion() *Scenario {
	return NewScenario("retry_exhaustion", "100% failure with limited budget - tests graceful degradation").
		AddPackage(TestPackage{
			PackageID:     "test-mod-e",
			WorkshopID:    "333333333",
			SHA256:        "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			ShouldSucceed: false, // Should fail after budget exhausted
		}).
		AddFailure(FailureConfig{
			FailureMode:    "timeout",
			Probability:    1.0,
			TriggerAttempt: 0,
		}).
		SetExpectation(ExpectationConfig{
			TotalSuccesses: 0,
			TotalFailures:  1,
			MaxDuration:    60000,
			AllowPartial:   true,
		})
}

// FailureModeToSessionMode maps scenario failure modes to session failure modes
func FailureModeToSessionMode(mode string) session.FailureMode {
	switch mode {
	case "timeout":
		return session.FailureNetworkTimeout
	case "http_error":
		return session.FailureHTTPError
	case "hash_mismatch":
		return session.FailureHashMismatch
	case "partial_download":
		return session.FailurePartialDownload
	case "steam_api_unavailable":
		return session.FailureSteamAPIUnavailable
	default:
		return session.FailureNone
	}
}
