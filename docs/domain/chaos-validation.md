# Chaos Validation Suite

System-level testing framework for validating execution behavior under controlled chaos.

## Purpose

Proves that the execution kernel behaves correctly under stress — not just in happy path, but in real-world failure scenarios.

## Philosophy

> "If the system doesn't survive chaos in testing, it won't survive reality in production."

## Architecture

```
Scenario Definition
       ↓
Failure Injection (controlled)
       ↓
Session Execution (with chaos)
       ↓
Outcome Validation (expectations)
       ↓
Baseline Recording (optional)
       ↓
Replay Comparison (determinism)
```

## Components

### 1. Scenario Definition (`libs/chaos/scenario.go`)

Defines test cases with:
- **Packages**: What to download
- **Failures**: When/how to inject failures
- **Expectations**: What success looks like

```go
scenario := chaos.NewScenario("flaky_network", "Validates recovery from network issues").
    AddPackage(chaos.TestPackage{
        PackageID:     "test-mod-a",
        WorkshopID:  "123456789",
        SHA256:        "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
        ShouldSucceed: true,
    }).
    AddFailure(chaos.FailureConfig{
        FailureMode:  "timeout",
        Probability:  0.3,  // 30% timeout rate
    }).
    SetExpectation(chaos.ExpectationConfig{
        TotalSuccesses: 1,
        MaxDuration:    60000,
    })
```

### 2. Failure Injection (`libs/chaos/runner.go`)

Maps scenario failures to `session.FailureInjector`:

| Scenario Mode | Session Mode | Behavior |
|--------------|--------------|----------|
| `timeout` | `FailureNetworkTimeout` | Connection timeout |
| `http_error` | `FailureHTTPError` | HTTP 5xx/429 |
| `hash_mismatch` | `FailureHashMismatch` | Corrupt source (non-retryable) |
| `partial_download` | `FailurePartialDownload` | Connection drop mid-download |
| `steam_api_unavailable` | `FailureSteamAPIUnavailable` | Complete API failure |

### 3. Runner (`libs/chaos/runner.go`)

Executes scenarios and validates expectations:

```go
runner := chaos.NewRunner(cacheDir, executor)
result, err := runner.RunScenario(ctx, scenario)

// Validation includes:
// - Did expected successes occur?
// - Did expected failures occur?
// - Was duration within limits?
// - Is partial success acceptable?
```

### 4. Replay Engine (`libs/chaos/replay.go`)

Validates determinism:

```go
replay := chaos.NewReplayEngine(baselineDir)

// Record baseline
replay.RecordBaseline(result)

// Later: replay and compare
comparison, err := replay.ReplayResult(newResult)
// comparison.Deterministic = true/false
// comparison.DurationDelta = acceptable variance
```

## Preset Scenarios

### 1. Flaky Network (`flaky_network`)
**Purpose**: Validates retry and recovery
- 30% network timeout
- 20% HTTP errors
- **Expectation**: 100% success (via retry)

### 2. Steam API Down (`steam_api_down`)
**Purpose**: Validates fallback to SteamCMD
- 100% Steam API unavailable
- **Expectation**: 100% success (via SteamCMD)

### 3. Hash Mismatch (`hash_mismatch`)
**Purpose**: Validates fail-fast for corrupt source
- 100% hash mismatch
- **Expectation**: 0% success (correctly fails)

### 4. Partial Download (`partial_download`)
**Purpose**: Validates resume/retry behavior
- 50% connection drop mid-download
- **Expectation**: 100% success (via retry)

### 5. Retry Exhaustion (`retry_exhaustion`)
**Purpose**: Validates graceful degradation
- 100% failure with limited budget
- **Expectation**: 0% success (budget exhausted)

## CLI Usage

### List Scenarios
```bash
go run apps/chaos-cli/main.go -list
```

### Run Single Scenario
```bash
go run apps/chaos-cli/main.go -scenario=flaky_network -v
```

### Run Full Suite
```bash
go run apps/chaos-cli/main.go
```

### Record Baselines
```bash
go run apps/chaos-cli/main.go -record
```

### Replay & Validate Determinism
```bash
go run apps/chaos-cli/main.go -replay
```

## Interpretation

### Pass Criteria
- **Suite Pass**: ≥80% scenarios pass
- **Determinism**: ≥80% deterministic replays

### What Success Means
1. System correctly handles network failures (retry works)
2. Fallback chain executes (API → SteamCMD)
3. Corrupt sources are detected (hash mismatch fails fast)
4. Partial downloads are recovered
5. Budget exhaustion is graceful

### What Failure Means
- **Scenario fails**: System doesn't handle chaos correctly
- **Non-deterministic**: Behavior varies unpredictably (usually timing, but could be race conditions)

## Integration with Development

### CI/CD Pipeline
```yaml
- name: Chaos Tests
  run: |
    go run apps/chaos-cli/main.go
    go run apps/chaos-cli/main.go -record
    go run apps/chaos-cli/main.go -replay
```

### Local Development
```bash
# Before committing, verify system stability
make chaos-test

# After changes, ensure no regressions
make chaos-test-replay
```

## Extending Scenarios

```go
// Add custom scenario
func PresetCustomChaos() *chaos.Scenario {
    return chaos.NewScenario("custom", "Description").
        AddPackage(chaos.TestPackage{
            PackageID:     "my-mod",
            WorkshopID:    "999999999",
            SHA256:        "ffffffff...",
            ShouldSucceed: true,
        }).
        AddFailure(chaos.FailureConfig{
            FailureMode:  "timeout",
            Probability:  0.5,
        }).
        SetExpectation(chaos.ExpectationConfig{
            TotalSuccesses: 1,
            MaxDuration:    120000,
        })
}

// Register in PresetScenarios()
```

## Relationship to Production

| Chaos Scenario | Production Equivalent |
|----------------|----------------------|
| Flaky Network | WiFi drops, mobile networks |
| Steam API Down | Steam maintenance, rate limits |
| Hash Mismatch | Corrupted CDN cache |
| Partial Download | Large file interrupted |
| Retry Exhaustion | Persistent network issues |

## Determinism Notes

**What is deterministic:**
- Success/failure outcomes
- Number of attempts (±2 variance allowed)
- State transitions

**What is non-deterministic (acceptable variance):**
- Timing (±50% of baseline)
- Network delay
- Retry intervals

**Why this matters:**
- Deterministic core = predictable behavior
- Acceptable variance = real-world flexibility
- Non-deterministic failures = bugs to fix

## Success Metrics

After running chaos validation:

```
CHAOS TEST SUITE RESULTS
==================================================
Duration: 4500ms
Scenarios: 5 total
Passed: 5 (100.0%)
Failed: 0

✓ Suite PASSED (>=80% scenarios passed)
```

And determinism report:

```
DETERMINISM REPORT
==================================================
Total: 5
Deterministic: 5
Non-deterministic: 0
Pass Rate: 100.0%
```

This proves the system is **production-grade**.
