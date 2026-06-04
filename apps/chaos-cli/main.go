package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"pzlauncher/libs/chaos"
	"pzlauncher/libs/session"
)

func main() {
	var (
		scenarioName = flag.String("scenario", "", "Specific scenario to run (empty = run all)")
		cacheDir     = flag.String("cache", "cache/chaos", "Cache directory for test artifacts")
		record       = flag.Bool("record", false, "Record results as new baseline")
		replay       = flag.Bool("replay", false, "Compare against existing baseline")
		list         = flag.Bool("list", false, "List available scenarios")
		verbose      = flag.Bool("v", false, "Verbose output")
	)
	flag.Parse()

	// Ensure cache directory exists
	os.MkdirAll(*cacheDir, 0755)

	// List scenarios and exit
	if *list {
		printScenarios()
		return
	}

	// Setup executor
	executor := session.NewSteamExecutor(*cacheDir)

	// Try to auto-detect steamcmd
	if steamcmdPath := session.FindSteamCMD(); steamcmdPath != "" {
		executor.WithSteamCMD(steamcmdPath)
		if *verbose {
			fmt.Printf("[Chaos] SteamCMD found: %s\n", steamcmdPath)
		}
	}

	// Create runner
	runner := chaos.NewRunner(*cacheDir, executor).
		WithResultsDir(*cacheDir + "/results")

	// Create replay engine if needed
	var replayEngine *chaos.ReplayEngine
	if *replay {
		replayEngine = chaos.NewReplayEngine(*cacheDir + "/baselines")
	}

	// Run specific scenario or full suite
	ctx := context.Background()

	if *scenarioName != "" {
		// Run single scenario
		scenario := findScenario(*scenarioName)
		if scenario == nil {
			log.Fatalf("Unknown scenario: %s", *scenarioName)
		}

		result, err := runner.RunScenario(ctx, scenario)
		if err != nil {
			log.Fatalf("Scenario failed: %v", err)
		}

		printResult(result, *verbose)

		// Record baseline
		if *record {
			if replayEngine == nil {
				replayEngine = chaos.NewReplayEngine(*cacheDir + "/baselines")
			}
			if err := replayEngine.RecordBaseline(result); err != nil {
				log.Printf("Failed to record baseline: %v", err)
			} else {
				fmt.Printf("[Chaos] Baseline recorded for %s\n", scenario.Name)
			}
		}

		// Replay comparison
		if *replay && replayEngine != nil {
			comparison, err := replayEngine.ReplayResult(result)
			if err != nil {
				fmt.Printf("[Chaos] No baseline for comparison: %v\n", err)
			} else {
				printComparison(comparison)
			}
		}

	} else {
		// Run full suite
		scenarios := chaos.PresetScenarios()

		fmt.Printf("[Chaos] Running %d scenarios...\n\n", len(scenarios))

		suiteResult, err := runner.RunSuite(ctx, scenarios)
		if err != nil {
			log.Fatalf("Suite failed: %v", err)
		}

		printSuiteResult(suiteResult)

		// Record baselines for all
		if *record {
			if replayEngine == nil {
				replayEngine = chaos.NewReplayEngine(*cacheDir + "/baselines")
			}
			for _, scenarioResult := range suiteResult.Scenarios {
				if err := replayEngine.RecordBaseline(&scenarioResult); err != nil {
					log.Printf("Failed to record baseline for %s: %v", scenarioResult.ScenarioName, err)
				}
			}
			fmt.Printf("\n[Chaos] Recorded %d baselines\n", len(suiteResult.Scenarios))
		}

		// Replay comparison for all
		if *replay && replayEngine != nil {
			report, err := replayEngine.GenerateReport(convertToPointers(suiteResult.Scenarios))
			if err != nil {
				log.Printf("Failed to generate determinism report: %v", err)
			} else {
				printDeterminismReport(report)

				// Save report
				if err := replayEngine.SaveReport(report); err != nil {
					log.Printf("Failed to save report: %v", err)
				}
			}
		}
	}
}

func printScenarios() {
	scenarios := chaos.PresetScenarios()

	fmt.Println("Available Chaos Test Scenarios:")
	fmt.Println("================================")

	for _, s := range scenarios {
		fmt.Printf("\n%s\n", s.Name)
		fmt.Printf("  Description: %s\n", s.Description)
		fmt.Printf("  Packages: %d\n", len(s.Packages))
		fmt.Printf("  Failures: %d configured\n", len(s.Failures))
		fmt.Printf("  Expected: %d success, %d failure\n",
			s.Expectation.TotalSuccesses, s.Expectation.TotalFailures)
	}

	fmt.Println("\n\nUsage:")
	fmt.Println("  go run apps/chaos-cli/main.go -scenario=<name>    # Run single scenario")
	fmt.Println("  go run apps/chaos-cli/main.go                      # Run full suite")
	fmt.Println("  go run apps/chaos-cli/main.go -record              # Record baselines")
	fmt.Println("  go run apps/chaos-cli/main.go -replay              # Compare to baselines")
}

func findScenario(name string) *chaos.Scenario {
	scenarios := chaos.PresetScenarios()
	for _, s := range scenarios {
		if s.Name == name {
			return s
		}
	}
	return nil
}

func printResult(result *chaos.ScenarioResult, verbose bool) {
	status := "PASS"
	if !result.Success {
		status = "FAIL"
	}

	fmt.Printf("\n[%s] %s\n", status, result.ScenarioName)
	fmt.Printf("Duration: %dms\n", result.DurationMs)
	fmt.Printf("Packages: %d total\n", len(result.PackageResults))

	for _, pkg := range result.PackageResults {
		icon := "✓"
		if !pkg.Success {
			icon = "✗"
		}
		fmt.Printf("  %s %s (attempts: %d, duration: %dms)\n",
			icon, pkg.PackageID, pkg.Attempts, pkg.DurationMs)

		if verbose && len(pkg.Errors) > 0 {
			for _, err := range pkg.Errors {
				fmt.Printf("    Error: %s\n", err)
			}
		}
	}

	if verbose && len(result.Events) > 0 {
		fmt.Println("\nEvents:")
		for _, event := range result.Events {
			fmt.Printf("  [%s] %s: %s\n",
				event.Timestamp.Format("15:04:05"), event.Type, event.Details)
		}
	}
}

func printSuiteResult(result *chaos.SuiteResult) {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("CHAOS TEST SUITE RESULTS")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Duration: %dms\n", result.DurationMs)
	fmt.Printf("Scenarios: %d total\n", result.TotalScenarios)
	fmt.Printf("Passed: %d (%.1f%%)\n", result.Passed, result.PassRate*100)
	fmt.Printf("Failed: %d\n", result.Failed)

	if result.Failed > 0 {
		fmt.Println("\nFailed Scenarios:")
		for _, s := range result.Scenarios {
			if !s.Success {
				fmt.Printf("  ✗ %s\n", s.ScenarioName)
			}
		}
	}

	if result.PassRate >= 0.8 {
		fmt.Println("\n✓ Suite PASSED (>=80% scenarios passed)")
	} else {
		fmt.Println("\n✗ Suite FAILED (<80% scenarios passed)")
		os.Exit(1)
	}
}

func printComparison(comp *chaos.ReplayComparison) {
	fmt.Printf("\n[Replay] %s\n", comp.ScenarioName)

	if comp.Deterministic {
		fmt.Println("  ✓ Deterministic (matches baseline)")
	} else {
		fmt.Println("  ✗ Non-deterministic (differs from baseline)")
		for _, diff := range comp.Differences {
			fmt.Printf("    - %s\n", diff)
		}
	}

	fmt.Printf("  Duration delta: %.1f%%\n", comp.DurationDelta)
}

func printDeterminismReport(report *chaos.DeterminismReport) {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("DETERMINISM REPORT")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Total: %d\n", report.TotalScenarios)
	fmt.Printf("Deterministic: %d\n", report.Deterministic)
	fmt.Printf("Non-deterministic: %d\n", report.NonDeterministic)
	fmt.Printf("Pass Rate: %.1f%%\n", report.PassRate*100)

	if report.NonDeterministic > 0 {
		fmt.Println("\nNon-deterministic scenarios:")
		for _, comp := range report.Comparisons {
			if !comp.Deterministic {
				fmt.Printf("  ✗ %s\n", comp.ScenarioName)
				for _, diff := range comp.Differences {
					fmt.Printf("      %s\n", diff)
				}
			}
		}
	}
}

func convertToPointers(scenarios []chaos.ScenarioResult) []*chaos.ScenarioResult {
	pointers := make([]*chaos.ScenarioResult, len(scenarios))
	for i := range scenarios {
		pointers[i] = &scenarios[i]
	}
	return pointers
}
