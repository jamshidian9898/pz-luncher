package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"strings"

	"pzlauncher/libs/validation"
)

const (
	version     = "v1.0.0"
	versionDate = "2026-06-04"
)

func main() {
	var (
		runs        = flag.Int("runs", 100, "Total number of sessions to execute")
		interval    = flag.Duration("interval", 30*time.Second, "Interval between runs")
		mode        = flag.String("mode", "shadow", "Execution mode: live, chaos, shadow")
		packages    = flag.Int("packages", 2, "Packages per session")
		concurrent  = flag.Int("concurrent", 3, "Max concurrent sessions")
		drift       = flag.Float64("drift", 0.10, "Max acceptable drift rate (0-1)")
		cacheDir    = flag.String("cache", "cache/campaign", "Cache directory")
		name        = flag.String("name", "default", "Campaign name")
		infinite    = flag.Bool("infinite", false, "Run until manually stopped (overrides -runs)")
		showVersion = flag.Bool("version", false, "Show version")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("campaign-cli %s (built %s)\n", version, versionDate)
		fmt.Println("Platform: pzlauncher v1.0.0")
		fmt.Println("Status: Core FROZEN, Validation STABLE")
		os.Exit(0)
	}

	// Parse mode
	var execMode validation.ExecutionMode
	switch *mode {
	case "live":
		execMode = validation.ModeLive
	case "chaos":
		execMode = validation.ModeChaos
	case "shadow":
		execMode = validation.ModeShadow
	default:
		log.Fatalf("Unknown mode: %s (use: live, chaos, shadow)", *mode)
	}

	// Calculate total runs
	totalRuns := *runs
	if *infinite {
		totalRuns = 0 // 0 means infinite
	}

	// Create config
	config := validation.CampaignConfig{
		TotalRuns:       totalRuns,
		Interval:        *interval,
		Mode:            execMode,
		PackagesPerRun:  *packages,
		MaxConcurrent:   *concurrent,
		MetricsInterval: 1 * time.Minute,
		DriftThreshold:  *drift,
	}

	// Create campaign
	description := fmt.Sprintf("Long-run validation: %d runs, %s interval, %s mode",
		totalRuns, *interval, *mode)
	if *infinite {
		description = fmt.Sprintf("Continuous validation: %s interval, %s mode until stopped",
			*interval, *mode)
	}

	campaign := validation.NewCampaign(*name, description, config, *cacheDir)

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Print campaign info
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("EXTENDED VALIDATION CAMPAIGN")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Name: %s\n", *name)
	fmt.Printf("Description: %s\n", description)
	fmt.Printf("Mode: %s\n", *mode)
	if *infinite {
		fmt.Printf("Runs: infinite (until stopped)\n")
	} else {
		fmt.Printf("Runs: %d\n", totalRuns)
	}
	fmt.Printf("Interval: %s\n", *interval)
	fmt.Printf("Concurrent: %d\n", *concurrent)
	fmt.Printf("Packages/Run: %d\n", *packages)
	fmt.Printf("Drift Threshold: %.1f%%\n", *drift*100)
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()

	// Start campaign
	if err := campaign.Start(); err != nil {
		log.Fatalf("Failed to start campaign: %v", err)
	}

	// Wait for completion or signal
	if *infinite {
		fmt.Println("Campaign running infinitely. Press Ctrl+C to stop...")
		<-sigChan
		fmt.Println("\nStopping campaign...")
		campaign.Stop()
	} else {
		// Wait for completion or signal
		go func() {
			<-sigChan
			fmt.Println("\nReceived interrupt, stopping campaign...")
			campaign.Stop()
		}()

		// Wait for campaign to complete
		for campaign.IsRunning() {
			time.Sleep(1 * time.Second)
		}
	}

	// Print final report
	printFinalReport(campaign)
}

func printFinalReport(campaign *validation.Campaign) {
	metrics := campaign.GetMetrics()
	report := metrics.GenerateSLOReport()

	fmt.Println()
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("CAMPAIGN FINAL REPORT")
	fmt.Println(strings.Repeat("=", 60))

	fmt.Printf("\nExecution Summary:\n")
	fmt.Printf("  Total Executions: %d\n", metrics.TotalExecutions)
	fmt.Printf("  Successful: %d (%.2f%%)\n",
		metrics.SuccessfulExecutions,
		float64(metrics.SuccessfulExecutions)/float64(metrics.TotalExecutions)*100)
	fmt.Printf("  Failed: %d (%.2f%%)\n",
		metrics.FailedExecutions,
		float64(metrics.FailedExecutions)/float64(metrics.TotalExecutions)*100)

	fmt.Printf("\nSLO Compliance:\n")
	fmt.Printf("  Availability: %.2f%% (target: 99%%) %s\n",
		report.AvailabilityActual*100,
		checkmark(report.AvailabilityMet))
	fmt.Printf("  Success Rate: %.2f%% (target: 95%%) %s\n",
		report.SuccessRateActual*100,
		checkmark(report.SuccessRateMet))
	fmt.Printf("  Drift Rate: %.2f%% (target: <10%%) %s\n",
		report.DriftRateActual*100,
		checkmark(report.DriftRateMet))
	fmt.Printf("  P99 Latency: %s (target: <60s) %s\n",
		report.P99LatencyActual,
		checkmark(report.P99LatencyMet))

	fmt.Printf("\nReliability Score: %.0f/100\n", report.ReliabilityScore)

	if report.AllSLOsMet {
		fmt.Println("\n✓ ALL SLOs MET - Platform validated for production")
	} else {
		fmt.Println("\n✗ Some SLOs not met - Review before production")
	}

	fmt.Println(strings.Repeat("=", 60))

	// Exit with appropriate code
	if report.AllSLOsMet {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}

func checkmark(met bool) string {
	if met {
		return "✓"
	}
	return "✗"
}
