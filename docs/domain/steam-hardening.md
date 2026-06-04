# Steam Integration Hardening Layer

This document describes the production-grade hardening layer for Steam Workshop integration.

## Overview

The hardening layer transforms the basic Steam executor into a **resilient, observable, and testable** production system. It addresses real-world concerns:

- **Rate limiting** — Prevents API bans
- **Failure injection** — Validates robustness before production
- **Workshop ID mapping** — Bridges mod names to Steam artifacts
- **Retry budgets** — Prevents infinite retry loops
- **Multi-strategy fallback** — API → SteamCMD → Cache

## Architecture

```
Mod Name (e.g., "Brita")
       ↓
Workshop ID Mapping
       ↓
Rate Limiter (token bucket)
       ↓
Steam Web API → Direct Download URL
       ↓ (if API unavailable/no URL)
SteamCMD Fallback
       ↓
Download with Progress
       ↓
SHA256 Verification
       ↓
Atomic Store (temp → final)
       ↓
Cache/sha256/<hash>
```

## Components

### 1. Workshop ID Mapping (`workshop_mapping.go`)

Resolves mod names (e.g., "Brita") to Steam Workshop IDs (e.g., "123456789").

**Priority Chain:**
1. **Local cache** — Previously resolved mappings
2. **Numeric check** — If already a number, assume Workshop ID
3. **Registry API** — Query mod registry service
4. **Steam API** — Direct search (fallback)

**Usage:**
```go
mapper := session.NewMappingService(cacheDir)

// Load manual mappings
mapper.AddManualMapping("Brita", "123456789", "Brita's Weapon Pack")

// Or load from file
mapper.LoadFromFile("workshop-mappings.json")

// Resolve
workshopID, err := mapper.Resolve("Brita")
```

### 2. Rate Limiting (`ratelimit.go`)

Token bucket rate limiter prevents Steam API bans.

**Steam Limits:**
- ~100,000 requests per day
- ~1.15 requests per second average
- Conservative: 1 req/sec with burst of 10

**Usage:**
```go
limiter := session.NewRateLimiter().WithSteamLimits()

// Blocks until token available
err := limiter.Acquire(ctx)
```

**Features:**
- Token refill (1 per second)
- Burst capacity (10 tokens)
- Minimum interval (1 second between requests)
- Context cancellation support

### 3. Failure Injection (`failure_inject.go`)

Chaos testing framework for validating robustness.

**Failure Modes:**

| Mode | Probability | Retryable | Description |
|------|-------------|-----------|-------------|
| NetworkTimeout | 10% | Yes | Connection timeout |
| HTTPError | 5% | Yes (429, 5xx) | API errors |
| PartialDownload | 5% | Yes | Connection closed mid-download |
| CorruptData | 5% | Yes | Stream corruption |
| HashMismatch | 2% | **No** | Source corruption (fatal) |
| SteamAPIUnavailable | Variable | Yes | API completely down |

**Usage:**
```go
injector := session.NewFailureInjector()

// Enable chaos mode
injector.PresetChaosMode()

// Or manual configuration
injector.SetFailureProbability(session.FailureNetworkTimeout, 0.1)
injector.SetNetworkDelay(100*time.Millisecond, 2*time.Second)

// Attach to executor
steamExecutor.WithFailureInjector(injector)
```

**Presets:**
- `PresetChaosMode()` — Flaky network, occasional failures
- `PresetCorruptMode()` — Rare but serious data corruption
- `PresetSteamDownMode()` — Complete API failure, tests fallback

### 4. Retry Budget System

Global retry budget prevents infinite loops and resource exhaustion.

**Configuration:**
```go
type SteamExecutor struct {
    MaxRetries  int // Per-attempt retries (default: 3)
    RetryBudget int // Global session budget (default: 10)
}
```

**Behavior:**
- Each retry consumes 1 from budget
- Budget exhaustion → immediate failure
- Distinguishes retryable vs non-retryable errors

**Error Classification:**

**Retryable:**
- Network timeout
- HTTP 5xx / 429
- Partial download
- Stream corruption

**Non-Retryable:**
- Hash mismatch (source corrupt)
- Context cancellation
- Configuration error

### 5. Integration in SteamExecutor

All hardening layers integrate seamlessly:

```go
executor := session.NewSteamExecutor(cacheDir).
    WithSteamCMD("/usr/bin/steamcmd").
    WithFailureInjector(chaosInjector)

// Execution flow:
// 1. Check rate limiter
// 2. Maybe inject failure
// 3. Resolve workshop ID via MappingService
// 4. Try Steam API (with rate limit)
// 5. Fallback to SteamCMD
// 6. Download with progress
// 7. Verify hash
// 8. Atomic store
```

## Testing Scenarios

### Scenario 1: Happy Path
```go
// No failures injected
// Expected: Download succeeds, ~1-5 seconds
```

### Scenario 2: Flaky Network (Chaos Mode)
```go
injector.PresetChaosMode()
// 10% timeout, 5% HTTP errors, network delays
// Expected: Retry succeeds, ~5-15 seconds
```

### Scenario 3: Steam API Down
```go
injector.PresetSteamDownMode()
// 100% API failure
// Expected: Falls back to SteamCMD
```

### Scenario 4: Data Corruption
```go
injector.PresetCorruptMode()
// 2% hash mismatch
// Expected: Non-retryable failure, clear error
```

### Scenario 5: Rate Limit Exhaustion
```go
// Make 100 rapid requests
// Expected: Rate limiter blocks, orderly queuing
```

## Observability

### Progress Events
```go
type ProgressEvent struct {
    PackageID       string
    Provider        string // "steam", "steamcmd"
    BytesDownloaded int64
    BytesTotal      int64
    SpeedBps        float64
    Percent         float64
}
```

### Execution Trace
```json
{
  "packageId": "mod-b",
  "state": "complete",
  "attempts": 2,
  "durationMs": 8500,
  "error": "",
  "workshopId": "123456789",
  "downloadMethod": "steamcmd"
}
```

## Production Checklist

- [ ] Rate limiter configured for Steam API limits
- [ ] Workshop ID mappings loaded for common mods
- [ ] SteamCMD available as fallback
- [ ] Retry budget appropriate for mod count
- [ ] Chaos testing passed (all failure modes)
- [ ] Rate limit testing passed (high load)
- [ ] Fallback chain tested (API → SteamCMD)
- [ ] Corruption recovery tested (hash mismatch handling)

## Future Enhancements

1. **Adaptive Rate Limiting** — Adjust based on API response headers
2. **Circuit Breaker** — Pause requests when Steam API consistently fails
3. **Download Resume** — HTTP range requests for partial downloads
4. **CDN Detection** — Choose nearest download server
5. **Parallel Downloads** — Multiple mods concurrently (within rate limits)
