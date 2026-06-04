# Download Execution Architecture

This document describes the execution layer that transforms provider decisions into actual artifacts.

## Overview

The execution layer bridges the gap between **"which provider should deliver this"** and **"the artifact is now in cache"**.

```
Provider Decision
       ↓
Session Manager (orchestration)
       ↓
Composite Executor (routing)
       ↓
Provider-Specific Executor (Steam/HTTP/Registry)
       ↓
Artifact in cache/sha256/
```

## Key Design Principles

### 1. Complexity Lives in Providers

The session manager remains simple:
- Calls `executor.Execute(ctx, exec)`
- Tracks state transitions
- Persists progress

All complexity lives in provider executors:
- Steam: Workshop API, authentication, retry logic
- HTTP: Range requests, redirects, TLS
- Registry: Metadata resolution, CDN selection

### 2. Idempotency at Every Level

- **Session level**: Same inputs → same session ID
- **File level**: Skip download if hash already matches
- **API level**: Safe to retry without side effects

### 3. Observable by Default

Every executor must produce:
- Execution state (pending → downloading → complete/failed)
- Timing information (duration per attempt)
- Error classification (retryable vs fatal)
- Progress metrics (bytes downloaded)

## Components

### Session Manager (`libs/session/manager.go`)

Orchestrates execution without knowing provider details:

```go
for _, pkg := range session.Executions {
    executor.Execute(ctx, pkg)  // Delegates to provider executor
}
```

Responsibilities:
- Create/resume sessions
- State persistence
- Progress tracking
- Error aggregation

### Composite Executor (`libs/session/composite_executor.go`)

Routes packages to provider-specific executors:

```go
if pkg.ProviderDecision.ChosenProvider == "Steam" {
    return steamExecutor.Execute(ctx, pkg)
}
if pkg.ProviderDecision.ChosenProvider == "LocalCache" {
    return localExecutor.Execute(ctx, pkg)
}
```

This allows multi-provider sessions where each package may use a different provider.

### Provider Executors

Each provider has its own executor implementing the same interface:

```go
type Executor interface {
    Execute(ctx context.Context, exec *PackageExecution) (*PackageExecution, error)
}
```

#### Steam Executor (`libs/session/steam_executor.go`)

Real-world integration with Steam Workshop using multi-strategy fallback:

**Download Chain:**
```
Resolve Workshop ID
       ↓
Try Steam Web API (direct URL)
       ↓ (if no URL or API unavailable)
Fallback to SteamCMD
       ↓
Verify SHA256
       ↓
Store in cache
```

**Key Features:**

1. **Steam Web API** (`libs/session/steam_api.go`)
   - Resolves Workshop ID to metadata
   - Gets direct download URL for public items
   - Validates item availability (not banned, public visibility)

2. **SteamCMD Fallback** (`libs/session/steamcmd.go`)
   - Used when API doesn't provide direct URL
   - Handles authenticated downloads for private items
   - Downloads to temp, verifies, then moves

3. **Progress Streaming** (`libs/session/progress.go`)
   ```go
   ProgressEvent {
       BytesDownloaded int64
       BytesTotal      int64
       SpeedBps        float64
       Percent         float64
   }
   ```

4. **Hash Verification at Multiple Stages**
   ```go
   // Before download (idempotency)
   if verifyHash(targetPath, expectedSHA) == nil {
       return alreadyComplete
   }
   
   // During download (streaming hash)
   hasher := sha256.New()
   io.MultiWriter(file, hasher)
   
   // After download (final verification)
   if actualSHA != expectedSHA {
       return hashMismatch // Non-retryable
   }
   ```

5. **Retry Strategy**
   - Network errors: retry with backoff
   - Hash mismatches: fail fast (source corrupt)
   - Context cancellation: abort immediately

#### Local Cache Executor (`libs/session/executor.go`)

Simple executor for already-cached content:

- Just verifies hash
- No network calls
- Instant completion

## Execution Flow

```
1. Create Session from Provider Decisions
   ├─ Generate deterministic session ID
   ├─ Check for existing session (resume)
   └─ Mark cached packages as "skipped"

2. Execute Session
   ├─ For each non-skipped package:
   │   ├─ Route to provider executor
   │   ├─ Execute with retry logic
   │   ├─ Update execution state
   │   └─ Persist progress
   └─ Mark session complete

3. Materialize Profile
   └─ Link/copy artifacts from cache to profile
```

## State Machine Integration

The execution layer extends the launch state machine:

```
ResolvingPackages
       ↓
CreatingSession        ← NEW
       ↓
Downloading            ← Execution happens here
       ↓
Verifying              ← NEW: Hash checks
       ↓
Materializing          ← NEW: Profile assembly
       ↓
Launching
```

## Trace Output

After execution, two trace files exist:

1. **provider-trace.json**: Why each provider was chosen
2. **session-trace.json**: How execution proceeded

Combined, they answer:
- "Why did we try to download from Steam?" (provider trace)
- "Did the download succeed?" (session trace)
- "How long did it take?" (both)
- "Can we resume if interrupted?" (session persistence)

## Future Work

### Testing & Hardening (Current Phase)
- Test Steam Web API with real workshop items
- Test SteamCMD fallback path with actual steamcmd
- Implement Workshop ID mapping service (mod name → Steam ID)
- Add rate limiting for Steam API calls
- API key management for private workshop items
- Handle restricted/banned workshop items gracefully

### Additional Providers (After Steam)
HTTP and Registry providers follow the same pattern:
1. Implement `Executor` interface
2. Register in `CompositeExecutor`
3. Add provider-specific configuration

HTTP Provider will be simpler (just URL fetch with progress).
Registry Provider adds metadata resolution before download.

### Multi-Source Downloads
Future enhancement: try multiple providers in sequence if one fails:
```
Steam API failed → try SteamCMD → try HTTP mirror → try Registry
```

This is where the `CompositeExecutor` pattern pays off — just add more fallbacks.

## Architecture Validation

This design is validated by:
- **Steam Executor**: Most complex real-world provider
- If Steam works, HTTP/Registry are just simplifications
- If the abstraction leaks, we'd see session manager changes
- Currently: zero changes to session manager for Steam integration ✓
