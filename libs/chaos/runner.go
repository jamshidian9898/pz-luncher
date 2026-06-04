package chaos

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"pzlauncher/libs/contracts"
	"pzlauncher/libs/session"
)

// Runner executes chaos test scenarios against the session/executor system
type Runner struct {
	cacheDir   string
	executor   session.Executor
	resultsDir string
	eventLog   []ChaosEvent
}

// NewRunner creates a new chaos test runner
func NewRunner(cacheDir string, executor session.Executor) *Runner {
	return &Runner{
		cacheDir:   cacheDir,
		executor:   executor,
		resultsDir: filepath.Join(cacheDir, "..", "chaos-results"),
		eventLog:   []ChaosEvent{},
	}
}

// WithResultsDir configures where to write test results
func (r *Runner) WithResultsDir(dir string) *Runner {
	r.resultsDir = dir
	return r
}

// RunScenario executes a single chaos test scenario
func (r *Runner) RunScenario(ctx context.Context, scenario *Scenario) (*ScenarioResult, error) {
	// Validate scenario
	if err := scenario.Validate(); err != nil {
		return nil, fmt.Errorf("invalid scenario: %w", err)
	}

	log.Printf("[Chaos] Starting scenario: %s - %s", scenario.Name, scenario.Description)

	// Prepare result
	result := &ScenarioResult{
		ScenarioName:   scenario.Name,
		StartTime:      time.Now(),
		PackageResults: []PackageResult{},
		Events:         []ChaosEvent{},
		Stats:          make(map[string]interface{}),
	}

	// Clear event log for this run
	r.eventLog = []ChaosEvent{}

	// Setup failure injection if configured
	injector := r.setupFailureInjection(scenario)

	// Create session manager for this test
	sessionsDir := filepath.Join(r.cacheDir, "..", "chaos-sessions")
	os.MkdirAll(sessionsDir, 0755)
	sessionMgr := session.NewSimpleManager(sessionsDir)

	// Convert test packages to provider decisions
	decisions := r.createDecisions(scenario.Packages)

	// Create session
	sess, err := sessionMgr.CreateSession("chaos-"+scenario.Name, "chaos-profile", decisions)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	// Configure executor with failure injector if present
	if steamExec, ok := r.executor.(*session.SteamExecutor); ok && injector != nil {
		steamExec.WithFailureInjector(injector)
	}

	// Execute session
	if err := sessionMgr.Execute(ctx, sess, r.executor); err != nil {
		log.Printf("[Chaos] Session execution error: %v", err)
	}

	// Collect results
	for _, exec := range sess.Executions {
		pkgResult := PackageResult{
			PackageID:   exec.PackageID,
			ExpectedSHA: exec.ProviderDecision.PackageSHA256,
			Success:     exec.State == contracts.PackageStateComplete,
			Attempts:    exec.Attempts,
			DurationMs:  exec.DurationMs,
			States:      []contracts.PackageExecutionState{exec.State},
		}
		if exec.Error != "" {
			pkgResult.Errors = []string{exec.Error}
		}
		result.PackageResults = append(result.PackageResults, pkgResult)
	}

	// Record completion
	result.EndTime = time.Now()
	result.DurationMs = result.EndTime.Sub(result.StartTime).Milliseconds()
	result.Events = r.eventLog

	// Validate against expectations
	result.Success = r.validateExpectations(result, &scenario.Expectation)

	// Calculate stats
	result.Stats["totalPackages"] = len(scenario.Packages)
	result.Stats["successfulPackages"] = r.countSuccesses(result.PackageResults)
	result.Stats["failedPackages"] = r.countFailures(result.PackageResults)
	result.Stats["totalAttempts"] = r.sumAttempts(result.PackageResults)
	result.Stats["avgDurationMs"] = result.DurationMs / int64(len(scenario.Packages))

	// Save results
	if err := r.saveResults(result); err != nil {
		log.Printf("[Chaos] Failed to save results: %v", err)
	}

	log.Printf("[Chaos] Scenario %s completed: success=%v, duration=%dms",
		scenario.Name, result.Success, result.DurationMs)

	return result, nil
}

// RunSuite executes multiple scenarios and returns aggregated results
func (r *Runner) RunSuite(ctx context.Context, scenarios []*Scenario) (*SuiteResult, error) {
	suiteResult := &SuiteResult{
		StartTime:      time.Now(),
		Scenarios:      []ScenarioResult{},
		Passed:         0,
		Failed:         0,
		TotalScenarios: len(scenarios),
	}

	for _, scenario := range scenarios {
		result, err := r.RunScenario(ctx, scenario)
		if err != nil {
			log.Printf("[Chaos] Scenario %s failed with error: %v", scenario.Name, err)
			// Mark as failed but continue
			result = &ScenarioResult{
				ScenarioName: scenario.Name,
				Success:      false,
				Events:       []ChaosEvent{{Type: "runner_error", Details: err.Error()}},
			}
		}

		suiteResult.Scenarios = append(suiteResult.Scenarios, *result)
		if result.Success {
			suiteResult.Passed++
		} else {
			suiteResult.Failed++
		}
	}

	suiteResult.EndTime = time.Now()
	suiteResult.DurationMs = suiteResult.EndTime.Sub(suiteResult.StartTime).Milliseconds()

	// Calculate pass rate
	if suiteResult.TotalScenarios > 0 {
		suiteResult.PassRate = float64(suiteResult.Passed) / float64(suiteResult.TotalScenarios)
	}

	// Save suite results
	r.saveSuiteResults(suiteResult)

	log.Printf("[Chaos] Suite completed: %d/%d passed (%.1f%%), duration=%dms",
		suiteResult.Passed, suiteResult.TotalScenarios,
		suiteResult.PassRate*100, suiteResult.DurationMs)

	return suiteResult, nil
}

// SuiteResult aggregates multiple scenario results
type SuiteResult struct {
	StartTime      time.Time        `json:"startTime"`
	EndTime        time.Time        `json:"endTime"`
	DurationMs     int64            `json:"durationMs"`
	Scenarios      []ScenarioResult `json:"scenarios"`
	Passed         int              `json:"passed"`
	Failed         int              `json:"failed"`
	TotalScenarios int              `json:"totalScenarios"`
	PassRate       float64          `json:"passRate"`
}

// setupFailureInjection configures the failure injector based on scenario
func (r *Runner) setupFailureInjection(scenario *Scenario) *session.FailureInjector {
	if len(scenario.Failures) == 0 {
		return nil
	}

	injector := session.NewFailureInjector()

	for _, config := range scenario.Failures {
		mode := FailureModeToSessionMode(config.FailureMode)
		if mode != session.FailureNone {
			injector.SetFailureProbability(mode, config.Probability)
		}
	}

	injector.Enable()
	return injector
}

// createDecisions converts test packages to provider decisions
func (r *Runner) createDecisions(packages []TestPackage) []contracts.ProviderDecision {
	decisions := []contracts.ProviderDecision{}

	for _, pkg := range packages {
		decision := contracts.ProviderDecision{
			PackageID:      pkg.PackageID,
			ChosenProvider: "Steam",
			PackageSHA256:  pkg.SHA256,
			DecisionAt:     time.Now(),
			FinalReason:    "chaos test",
			Attempts: []contracts.ProviderAttempt{
				{
					ProviderName: "Steam",
					CheckedAt:    time.Now(),
					Exists:       false,
				},
			},
		}
		decisions = append(decisions, decision)
	}

	return decisions
}

// validateExpectations checks if results match expected outcomes
func (r *Runner) validateExpectations(result *ScenarioResult, exp *ExpectationConfig) bool {
	// Check duration
	if result.DurationMs > exp.MaxDuration {
		log.Printf("[Chaos] FAIL: Duration %dms exceeds max %dms", result.DurationMs, exp.MaxDuration)
		return false
	}

	// Count successes and failures
	successes := r.countSuccesses(result.PackageResults)
	failures := r.countFailures(result.PackageResults)

	// Check expected counts
	if exp.TotalSuccesses >= 0 && successes != exp.TotalSuccesses {
		log.Printf("[Chaos] FAIL: Expected %d successes, got %d", exp.TotalSuccesses, successes)
		return false
	}

	if exp.TotalFailures >= 0 && failures != exp.TotalFailures {
		log.Printf("[Chaos] FAIL: Expected %d failures, got %d", exp.TotalFailures, failures)
		return false
	}

	// Check if partial is allowed
	if !exp.AllowPartial && failures > 0 {
		log.Printf("[Chaos] FAIL: Partial success not allowed, but %d failures occurred", failures)
		return false
	}

	return true
}

func (r *Runner) countSuccesses(results []PackageResult) int {
	count := 0
	for _, r := range results {
		if r.Success {
			count++
		}
	}
	return count
}

func (r *Runner) countFailures(results []PackageResult) int {
	count := 0
	for _, r := range results {
		if !r.Success {
			count++
		}
	}
	return count
}

func (r *Runner) sumAttempts(results []PackageResult) int {
	total := 0
	for _, r := range results {
		total += r.Attempts
	}
	return total
}

// saveResults persists scenario results to disk
func (r *Runner) saveResults(result *ScenarioResult) error {
	os.MkdirAll(r.resultsDir, 0755)

	filename := fmt.Sprintf("%s-%d.json", result.ScenarioName, result.StartTime.Unix())
	_ = filepath.Join(r.resultsDir, filename)

	return nil // JSON serialization would go here
}

// saveSuiteResults persists suite results
func (r *Runner) saveSuiteResults(result *SuiteResult) error {
	os.MkdirAll(r.resultsDir, 0755)

	filename := fmt.Sprintf("suite-%d.json", result.StartTime.Unix())
	_ = filepath.Join(r.resultsDir, filename)

	return nil
}

// LogEvent records a chaos event
func (r *Runner) LogEvent(eventType, packageID, details string) {
	event := ChaosEvent{
		Timestamp: time.Now(),
		Type:      eventType,
		PackageID: packageID,
		Details:   details,
	}
	r.eventLog = append(r.eventLog, event)
}
