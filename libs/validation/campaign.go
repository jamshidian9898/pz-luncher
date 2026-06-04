package validation

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"pzlauncher/libs/contracts"
	"pzlauncher/libs/session"
)

// CampaignTestPackage represents a test package in a campaign
type CampaignTestPackage struct {
	PackageID     string `json:"packageId"`
	WorkshopID    string `json:"workshopId"`
	SHA256        string `json:"sha256"`
	ShouldSucceed bool   `json:"shouldSucceed"`
}

// Campaign represents an extended validation run
type Campaign struct {
	Name        string
	Description string
	Config      CampaignConfig

	// State
	metrics    *ReliabilityMetrics
	shadowExec *ShadowExecutor
	sessionMgr *session.SimpleManager
	cacheDir   string

	// Control
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	mu      sync.RWMutex
	running bool
}

// CampaignConfig defines validation parameters
type CampaignConfig struct {
	TotalRuns       int           // Total number of sessions to execute (0 = infinite)
	Interval        time.Duration // Time between runs
	Mode            ExecutionMode // live, chaos, or shadow
	PackagesPerRun  int           // Packages per session
	MaxConcurrent   int           // Max concurrent sessions
	MetricsInterval time.Duration // How often to report metrics
	DriftThreshold  float64       // Max acceptable drift rate
}

// DefaultCampaignConfig returns sensible defaults
func DefaultCampaignConfig() CampaignConfig {
	return CampaignConfig{
		TotalRuns:       100,
		Interval:        30 * time.Second,
		Mode:            ModeShadow,
		PackagesPerRun:  2,
		MaxConcurrent:   3,
		MetricsInterval: 5 * time.Minute,
		DriftThreshold:  0.10,
	}
}

// NewCampaign creates a new validation campaign
func NewCampaign(name, description string, config CampaignConfig, cacheDir string) *Campaign {
	ctx, cancel := context.WithCancel(context.Background())

	return &Campaign{
		Name:        name,
		Description: description,
		Config:      config,
		metrics:     NewReliabilityMetrics(),
		cacheDir:    cacheDir,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start begins the campaign
func (c *Campaign) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return fmt.Errorf("campaign already running")
	}

	c.running = true

	// Setup infrastructure
	if err := c.setup(); err != nil {
		return fmt.Errorf("setup: %w", err)
	}

	// Start workers
	for i := 0; i < c.Config.MaxConcurrent; i++ {
		c.wg.Add(1)
		go c.worker(i)
	}

	// Start metrics reporter
	c.wg.Add(1)
	go c.metricsReporter()

	log.Printf("[Campaign] Started: %s - %s", c.Name, c.Description)
	log.Printf("[Campaign] Config: runs=%d, interval=%s, mode=%v",
		c.Config.TotalRuns, c.Config.Interval, c.Config.Mode)

	return nil
}

// Stop gracefully shuts down the campaign
func (c *Campaign) Stop() {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return
	}
	c.running = false
	c.mu.Unlock()

	log.Printf("[Campaign] Stopping: %s", c.Name)

	// Signal cancellation
	c.cancel()

	// Wait for workers
	c.wg.Wait()

	// Save final metrics
	c.saveFinalReport()

	log.Printf("[Campaign] Stopped: %s", c.Name)
}

// IsRunning returns campaign status
func (c *Campaign) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

// GetMetrics returns current metrics
func (c *Campaign) GetMetrics() *ReliabilityMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.metrics
}

// setup initializes infrastructure
func (c *Campaign) setup() error {
	// Ensure cache directory
	os.MkdirAll(c.cacheDir, 0755)

	// Create session manager
	sessionsDir := filepath.Join(c.cacheDir, "campaign-sessions")
	os.MkdirAll(sessionsDir, 0755)
	c.sessionMgr = session.NewSimpleManager(sessionsDir)

	// Create executors
	realExec := session.NewSteamExecutor(c.cacheDir)
	if steamcmdPath := session.FindSteamCMD(); steamcmdPath != "" {
		realExec.WithSteamCMD(steamcmdPath)
	}

	chaosExec := session.NewSteamExecutor(c.cacheDir)
	injector := session.NewFailureInjector()
	injector.PresetChaosMode()
	chaosExec.WithFailureInjector(injector)

	// Create shadow executor
	c.shadowExec = NewShadowExecutor(realExec, chaosExec).
		WithMode(c.Config.Mode)

	return nil
}

// worker executes sessions
func (c *Campaign) worker(id int) {
	defer c.wg.Done()

	runCount := 0
	ticker := time.NewTicker(c.Config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			log.Printf("[Campaign] Worker %d: shutting down", id)
			return

		case <-ticker.C:
			// Check if we've reached total runs
			if c.Config.TotalRuns > 0 && runCount >= c.Config.TotalRuns/c.Config.MaxConcurrent {
				return
			}

			// Execute a session
			if err := c.executeSession(runCount); err != nil {
				log.Printf("[Campaign] Worker %d: session error: %v", id, err)
			}

			runCount++
		}
	}
}

// executeSession runs a single validation session
func (c *Campaign) executeSession(index int) error {
	sessionID := fmt.Sprintf("%s-%d-%d", c.Name, time.Now().Unix(), index)

	// Generate test packages
	packages := c.generateTestPackages()
	decisions := c.createDecisions(packages)

	// Create session
	sess, err := c.sessionMgr.CreateSession(sessionID, "campaign-profile", decisions)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	// Execute
	start := time.Now()
	err = c.sessionMgr.Execute(c.ctx, sess, c.shadowExec)
	duration := time.Since(start)

	// Collect results
	for _, exec := range sess.Executions {
		success := exec.State == contracts.PackageStateComplete
		c.metrics.RecordExecution(success, duration,
			fmt.Errorf("%s", exec.Error),
			exec.ProviderDecision.ChosenProvider)
	}

	// Collect drift if shadow mode
	if c.Config.Mode == ModeShadow {
		driftReport := c.shadowExec.GetDriftReport()
		for _, drift := range driftReport.Drifts {
			c.metrics.RecordDrift(&drift)
		}
	}

	return nil
}

// generateTestPackages creates test data
func (c *Campaign) generateTestPackages() []CampaignTestPackage {
	packages := []CampaignTestPackage{}

	for i := 0; i < c.Config.PackagesPerRun; i++ {
		pkg := CampaignTestPackage{
			PackageID:     fmt.Sprintf("campaign-pkg-%d", i),
			WorkshopID:    fmt.Sprintf("%d", 100000000+i),
			SHA256:        fmt.Sprintf("%064d", i),
			ShouldSucceed: true,
		}
		packages = append(packages, pkg)
	}

	return packages
}

// createDecisions converts packages to provider decisions
func (c *Campaign) createDecisions(packages []CampaignTestPackage) []contracts.ProviderDecision {
	decisions := []contracts.ProviderDecision{}

	for _, pkg := range packages {
		decision := contracts.ProviderDecision{
			PackageID:      pkg.PackageID,
			ChosenProvider: "Steam",
			PackageSHA256:  pkg.SHA256,
			DecisionAt:     time.Now(),
			FinalReason:    "campaign test",
		}
		decisions = append(decisions, decision)
	}

	return decisions
}

// metricsReporter periodically reports metrics
func (c *Campaign) metricsReporter() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.Config.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.reportMetrics()
		}
	}
}

// reportMetrics prints current metrics
func (c *Campaign) reportMetrics() {
	report := c.metrics.GenerateSLOReport()

	log.Printf("[Campaign] Metrics: executions=%d, availability=%.2f%%, success=%.2f%%, drift=%.2f%%, score=%.0f/100",
		c.metrics.TotalExecutions,
		report.AvailabilityActual*100,
		report.SuccessRateActual*100,
		report.DriftRateActual*100,
		report.ReliabilityScore)

	// Check if campaign should stop due to excessive drift
	if report.DriftRateActual > c.Config.DriftThreshold {
		log.Printf("[Campaign] WARNING: Drift rate %.2f%% exceeds threshold %.2f%%",
			report.DriftRateActual*100, c.Config.DriftThreshold*100)
	}

	// Save metrics
	c.saveMetrics()
}

// saveMetrics persists current metrics
func (c *Campaign) saveMetrics() {
	path := filepath.Join(c.cacheDir, "campaign-metrics.json")
	if err := c.metrics.Save(path); err != nil {
		log.Printf("[Campaign] Failed to save metrics: %v", err)
	}
}

// saveFinalReport writes final campaign report
func (c *Campaign) saveFinalReport() {
	// Save metrics
	c.saveMetrics()

	// Generate SLO report
	sloReport := c.metrics.GenerateSLOReport()

	// Write report
	reportPath := filepath.Join(c.cacheDir, "campaign-final-report.json")

	// Summary
	summary := map[string]interface{}{
		"campaign":         c.Name,
		"description":      c.Description,
		"config":           c.Config,
		"metrics":          c.metrics,
		"sloReport":        sloReport,
		"allSLOsMet":       sloReport.AllSLOsMet,
		"reliabilityScore": sloReport.ReliabilityScore,
		"completedAt":      time.Now(),
	}

	// Simple JSON output would go here
	_ = summary
	_ = reportPath

	log.Printf("[Campaign] Final report: score=%.0f/100, allSLOsMet=%v",
		sloReport.ReliabilityScore, sloReport.AllSLOsMet)
}

// CampaignReport summarizes a completed campaign
type CampaignReport struct {
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	TotalExecutions  int64     `json:"totalExecutions"`
	SuccessRate      float64   `json:"successRate"`
	DriftRate        float64   `json:"driftRate"`
	ReliabilityScore float64   `json:"reliabilityScore"`
	AllSLOsMet       bool      `json:"allSLOsMet"`
	CompletedAt      time.Time `json:"completedAt"`
}
