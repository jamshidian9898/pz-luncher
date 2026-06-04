package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"pzlauncher/libs/contracts"
	"pzlauncher/libs/session"
	"pzlauncher/libs/validation"
)

func main() {
	var (
		mode      = flag.String("mode", "live", "Execution mode: live, chaos, shadow")
		packageID = flag.String("package", "test-mod", "Package ID to test")
		workshopID = flag.String("workshop", "123456789", "Steam Workshop ID")
		sha256    = flag.String("sha256", strings.Repeat("aa", 32), "Expected SHA256")
		cacheDir  = flag.String("cache", "cache/validation", "Cache directory")
		compare   = flag.Bool("compare", false, "Compare live vs chaos results")
		verbose   = flag.Bool("v", false, "Verbose output")
	)
	flag.Parse()

	// Ensure cache directory exists
	os.MkdirAll(*cacheDir, 0755)

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

	// Create decision
	decision := contracts.ProviderDecision{
		PackageID:       *packageID,
		ChosenProvider:  "Steam",
		PackageSHA256:  *sha256,
		DecisionAt:     time.Now(),
		FinalReason:    "validation test",
	}

	// Create execution
	exec := &contracts.PackageExecution{
		PackageID:      *packageID,
		ProviderDecision: decision,
		State:          contracts.PackageStatePending,
	}

	// Create executors
	realExecutor := createRealExecutor(*cacheDir)
	chaosExecutor := createChaosExecutor(*cacheDir)

	// Create shadow executor
	shadowExec := validation.NewShadowExecutor(realExecutor, chaosExecutor).
		WithMode(execMode)

	fmt.Printf("[Validation] Mode: %s\n", *mode)
	fmt.Printf("[Validation] Package: %s (Workshop: %s)\n", *packageID, *workshopID)
	fmt.Println()

	// Execute
	ctx := context.Background()
	start := time.Now()
	
	result, err := shadowExec.Execute(ctx, exec)
	
	duration := time.Since(start)

	// Print results
	if err != nil {
		fmt.Printf("✗ Execution failed: %v\n", err)
	} else {
		fmt.Printf("✓ Execution completed\n")
		fmt.Printf("  State: %s\n", result.State)
		fmt.Printf("  Duration: %dms\n", result.DurationMs)
		fmt.Printf("  Attempts: %d\n", result.Attempts)
		if result.CachePath != "" {
			fmt.Printf("  Cache: %s\n", result.CachePath)
		}
	}
	
	fmt.Printf("\nTotal time: %dms\n", duration.Milliseconds())

	// If shadow mode, print drift report
	if execMode == validation.ModeShadow {
		driftReport := shadowExec.GetDriftReport()
		printDriftReport(driftReport)
	}

	// Print telemetry
	if *verbose || *compare {
		telemetry := shadowExec.GetTelemetry()
		printTelemetryReport(telemetry)
	}

	// Compare mode: run both separately and show comparison
	if *compare {
		runComparison(ctx, realExecutor, chaosExecutor, exec)
	}
}

func createRealExecutor(cacheDir string) session.Executor {
	exec := session.NewSteamExecutor(cacheDir)
	
	// Try to auto-detect steamcmd
	if steamcmdPath := session.FindSteamCMD(); steamcmdPath != "" {
		exec.WithSteamCMD(steamcmdPath)
	}
	
	return exec
}

func createChaosExecutor(cacheDir string) session.Executor {
	exec := session.NewSteamExecutor(cacheDir)
	
	// Enable failure injection for chaos mode
	injector := session.NewFailureInjector()
	injector.PresetChaosMode() // 10% timeout, 5% HTTP errors
	exec.WithFailureInjector(injector)
	
	return exec
}

func runComparison(ctx context.Context, real, chaos session.Executor, exec *contracts.PackageExecution) {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("LIVE VS CHAOS COMPARISON")
	fmt.Println(strings.Repeat("=", 50))

	// Run live
	liveStart := time.Now()
	liveResult, liveErr := real.Execute(ctx, exec)
	liveDuration := time.Since(liveStart)
	
	// Clone for chaos
	chaosExec := &contracts.PackageExecution{
		PackageID:        exec.PackageID,
		ProviderDecision: exec.ProviderDecision,
		State:            exec.State,
	}
	
	// Run chaos
	chaosStart := time.Now()
	chaosResult, chaosErr := chaos.Execute(ctx, chaosExec)
	chaosDuration := time.Since(chaosStart)
	
	// Print comparison
	fmt.Printf("\nLive Run:\n")
	if liveErr != nil {
		fmt.Printf("  Error: %v\n", liveErr)
	} else {
		fmt.Printf("  State: %s\n", liveResult.State)
		fmt.Printf("  Duration: %dms\n", liveDuration.Milliseconds())
		fmt.Printf("  Attempts: %d\n", liveResult.Attempts)
	}
	
	fmt.Printf("\nChaos Run:\n")
	if chaosErr != nil {
		fmt.Printf("  Error: %v\n", chaosErr)
	} else {
		fmt.Printf("  State: %s\n", chaosResult.State)
		fmt.Printf("  Duration: %dms\n", chaosDuration.Milliseconds())
		fmt.Printf("  Attempts: %d\n", chaosResult.Attempts)
	}
	
	// Compare
	fmt.Printf("\nComparison:\n")
	if liveResult != nil && chaosResult != nil {
		if liveResult.State == chaosResult.State {
			fmt.Printf("  ✓ Outcome matches: %s\n", liveResult.State)
		} else {
			fmt.Printf("  ✗ Outcome mismatch: live=%s, chaos=%s\n", 
				liveResult.State, chaosResult.State)
		}
		
		// Timing ratio
		if chaosDuration > 0 {
			ratio := float64(liveDuration) / float64(chaosDuration)
			fmt.Printf("  Timing ratio (live/chaos): %.2f\n", ratio)
			if ratio > 3.0 || ratio < 0.33 {
				fmt.Printf("  ⚠ Significant timing drift detected\n")
			}
		}
		
		// Attempts
		if abs(liveResult.Attempts-chaosResult.Attempts) > 2 {
			fmt.Printf("  ⚠ Attempt count differs significantly: live=%d, chaos=%d\n",
				liveResult.Attempts, chaosResult.Attempts)
		}
	}
}

func printDriftReport(report *validation.DriftReport) {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("DRIFT DETECTION REPORT")
	fmt.Println(strings.Repeat("=", 50))
	
	fmt.Printf("Total comparisons: %d\n", report.TotalComparisons)
	fmt.Printf("Drifts detected: %d (%.1f%%)\n", report.DriftCount, report.DriftRate*100)
	
	if report.OutcomeMismatches > 0 {
		fmt.Printf("⚠ Outcome mismatches: %d\n", report.OutcomeMismatches)
	}
	if report.TimingDrifts > 0 {
		fmt.Printf("⚠ Timing drifts: %d\n", report.TimingDrifts)
	}
	if report.AttemptDrifts > 0 {
		fmt.Printf("⚠ Attempt count drifts: %d\n", report.AttemptDrifts)
	}
	
	if report.DriftCount == 0 {
		fmt.Println("✓ No drift detected between live and chaos execution")
	} else if report.DriftRate < 0.1 {
		fmt.Println("✓ Low drift rate - within acceptable bounds")
	} else {
		fmt.Println("✗ High drift rate - system may need adjustment")
	}
}

func printTelemetryReport(report *validation.TelemetryReport) {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("TELEMETRY REPORT")
	fmt.Println(strings.Repeat("=", 50))
	
	fmt.Printf("Session duration: %s\n", report.SessionDuration)
	fmt.Printf("Live runs: %d (success: %.1f%%)\n", report.LiveRuns, report.LiveSuccessRate*100)
	fmt.Printf("Chaos runs: %d (success: %.1f%%)\n", report.ChaosRuns, report.ChaosSuccessRate*100)
	fmt.Printf("Drifts: %d\n", report.Drifts)
	
	if report.AvgLiveDuration > 0 {
		fmt.Printf("Avg live duration: %s\n", report.AvgLiveDuration)
	}
	if report.AvgChaosDuration > 0 {
		fmt.Printf("Avg chaos duration: %s\n", report.AvgChaosDuration)
	}
	
	if len(report.LatencyProfile) > 0 {
		fmt.Println("\nLatency profile (live):")
		for bucket, count := range report.LatencyProfile {
			fmt.Printf("  %s: %d\n", bucket, count)
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
