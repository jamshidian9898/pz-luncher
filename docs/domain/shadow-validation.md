# Shadow Validation Layer

Real-world validation system that compares live execution against simulated chaos to verify system correctness.

## Purpose

> "Does the system behave in reality as it behaves in simulation?"

This is the final validation step before declaring the execution kernel "production-grade."

## The Problem

Chaos testing validates robustness, but with **controlled, known failures**:
```
Chaos: 30% timeout (deterministic probability)
```

Real-world has **uncontrolled, unknown failures**:
```
Reality: Steam API slow at 2AM, CDN cache miss, ISP throttling
```

## The Solution

**Shadow Validation** — Run both simultaneously and compare:

```
Live Execution        Chaos Simulation
       ↓                     ↓
   Real Steam API      Failure Injection
   Real Network        Controlled Chaos
   Real Timing         Simulated Timing
       ↓                     ↓
   Live Result         Chaos Result
       ↓                     ↓
         Drift Detection
              ↓
      Match? → System Valid
      Drift? → Investigate
```

## Components

### 1. Shadow Executor (`shadow_executor.go`)

Three execution modes:

```go
ModeLive    // Real APIs, no injection
ModeChaos   // Failure injection only
ModeShadow  // Both, with comparison
```

**Usage:**
```go
real := session.NewSteamExecutor(cacheDir)
chaos := session.NewSteamExecutor(cacheDir)
chaos.WithFailureInjector(injector)

shadow := validation.NewShadowExecutor(real, chaos).
    WithMode(validation.ModeShadow)

result, err := shadow.Execute(ctx, exec)
// Returns live result, but also compares with chaos
driftReport := shadow.GetDriftReport()
```

### 2. Drift Detection (`drift.go`)

Compares live vs chaos results and detects divergence:

**Drift Types:**

| Type | Description | Severity |
|------|-------------|----------|
| `outcome` | Live succeeded but chaos failed (or vice versa) | **Critical** |
| `timing` | Duration varies >3x | Warning |
| `attempts` | Retry count differs >2 | Warning |

**Acceptable Variance:**
- Outcome: Must match 100% (no variance allowed)
- Timing: ±3x (chaos delays are expected)
- Attempts: ±2 (some variance in retry count)

### 3. Telemetry Collection (`telemetry.go`)

Gathers real-world metrics:

```go
telemetry.RecordLiveRun(packageID, duration, err, state)
telemetry.RecordChaosRun(packageID, duration, err, state)
```

**Collected Data:**
- Execution duration (live vs chaos)
- Success rates
- Latency profiles (buckets: <100ms, 100-500ms, etc.)
- Drift detections

## Validation Workflow

### Phase 1: Controlled Chaos
```bash
# Validate system handles known failures
go run apps/chaos-cli/main.go
# Expect: 100% pass rate
```

### Phase 2: Live Validation
```bash
# Test against real Steam
go run apps/validation-cli/main.go -mode=live
# Expect: Success (proves real API works)
```

### Phase 3: Shadow Comparison
```bash
# Run both and compare
go run apps/validation-cli/main.go -mode=shadow
# Expect: Drift rate < 10%
```

### Phase 4: Detailed Comparison
```bash
# Side-by-side analysis
go run apps/validation-cli/main.go -compare
# Shows: live result, chaos result, differences
```

## Drift Interpretation

### Drift Rate < 10%
```
✓ System behaves correctly in reality
✓ Chaos simulation accurately models reality
✓ Safe to rely on chaos testing for regression
```

### Drift Rate 10-30%
```
⚠ Some divergence detected
→ Investigate specific drift types
→ May need to adjust chaos parameters
→ Real-world has factors chaos doesn't model
```

### Drift Rate > 30%
```
✗ Significant divergence
→ Chaos model is wrong
→ System may behave unpredictably in production
→ DO NOT deploy without investigation
```

## Example Output

### Shadow Mode
```
[Validation] Mode: shadow
[Validation] Package: test-mod (Workshop: 123456789)

✓ Execution completed
  State: complete
  Duration: 2500ms
  Attempts: 1

==================================================
DRIFT DETECTION REPORT
==================================================
Total comparisons: 1
Drifts detected: 0 (0.0%)

✓ No drift detected between live and chaos execution
```

### Compare Mode
```
==================================================
LIVE VS CHAOS COMPARISON
==================================================

Live Run:
  State: complete
  Duration: 2500ms
  Attempts: 1

Chaos Run:
  State: complete
  Duration: 12000ms  (injected delays)
  Attempts: 2       (1 retry due to timeout)

Comparison:
  ✓ Outcome matches: complete
  Timing ratio (live/chaos): 0.21
  ⚠ Significant timing drift detected (expected - chaos has delays)
  ⚠ Attempt count differs significantly: live=1, chaos=2
```

## Drift Analysis

### Good Drift (Expected)
- **Timing**: Chaos is slower (injected delays)
- **Attempts**: Chaos has more retries (injected failures)
- **Outcome**: Both succeed (resilience working)

### Bad Drift (Investigate)
- **Outcome mismatch**: Live succeeds, chaos fails
  - → Chaos model too pessimistic
- **Outcome mismatch**: Live fails, chaos succeeds
  - → Real-world problem not in chaos model
- **Timing**: Live much slower than chaos
  - → Real-world performance issue

## Production Readiness Criteria

| Criteria | Threshold | Status |
|----------|-----------|--------|
| Chaos pass rate | ≥80% | ✅ System handles simulated failures |
| Live success rate | ≥90% | ✅ Real API works |
| Drift rate | <10% | ✅ Model matches reality |
| Outcome match | 100% | ✅ Deterministic behavior |

**When all criteria met:**
> System is **production-grade**

## After Shadow Validation

Once shadow validation passes:

1. **HTTP Provider** — Trivial to add (same pattern)
2. **Registry Provider** — Metadata adapter only
3. **Multi-Node** — Safe to distribute
4. **UI** — Can be built on stable foundation

## Key Insight

> Shadow validation is the bridge between **"works in simulation"** and **"works in reality."**

Without it:
- Chaos testing proves nothing about production
- Real-world behavior is unknown
- Confidence is unjustified

With it:
- Chaos results are predictive of reality
- Real-world issues can be modeled
- System correctness is **verified**

## Summary

```
Simulation-Only System          Production-Grade System
        ↓                              ↓
   Chaos Testing              Chaos + Shadow Validation
   (controlled failures)      (controlled + real failures)
        ↓                              ↓
   "Should work"              "Proven to work"
        ↓                              ↓
   Hope-based                  Evidence-based
```

Shadow validation transforms the system from **"probably correct"** to **"verified correct."**
