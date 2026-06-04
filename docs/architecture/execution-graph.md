# Execution Graph (Simplified)

**Version**: 1.0 (Locked)  
**Scope**: Core execution flow only

```
┌─────────────────────────────────────────────────────────────────┐
│                        INPUT                                    │
│  Profile + PackageList + Versions                                 │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│  1. RESOLVE → ProviderDecision[]                                  │
│     - LocalCache check                                            │
│     - Provider selection (Steam, HTTP, etc.)                    │
│     - Decision trace recorded                                   │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│  2. CREATE → Session                                              │
│     - Deterministic ID generation                               │
│     - State = Pending                                             │
│     - Persist to disk                                             │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│  3. EXECUTE (for each package)                                    │
│     ┌─────────────┐    ┌─────────────┐    ┌─────────────┐     │
│     │  Provider   │ →  │   Executor  │ →  │    Result   │     │
│     │  Router     │    │  (Plugin)   │    │  (State)    │     │
│     └─────────────┘    └─────────────┘    └─────────────┘     │
│          ↓                    ↓                  ↓             │
│     SteamExecutor      Retry Logic          Complete/Failed    │
│     HTTPEXecutor       Rate Limiter        Error (if any)      │
│     etc.               Telemetry                              │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│  4. VERIFY                                                        │
│     - SHA256 check (if applicable)                              │
│     - State = Verifying → Complete/Failed                       │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│  5. OUTPUT                                                        │
│  SessionTrace + Cache Files + State                             │
└─────────────────────────────────────────────────────────────────┘
```

## State Machine (Locked)

```
Pending ──► Downloading ──► Verifying ──► Complete
   │            │              │
   │            │              │
   └────────────┴──────────────┘ Failed
                ▲
                │ (Resume)
            Interrupted
```

**Transitions:**
- Pending → Downloading: On first execution attempt
- Downloading → Verifying: Download completed, hash check
- Verifying → Complete: Verification passed
- Downloading/Verifying → Failed: Error or verification failed
- Failed → Downloading: On resume (if retry budget allows)

## Plugin Boundary

```
┌─────────────────────────────────────────┐
│           PLATFORM CORE                 │
│  (Frozen - No changes allowed)          │
│  • Session Manager                      │
│  • Executor Interface                   │
│  • State Machine                        │
│  • Provider Router                      │
└─────────────────────────────────────────┘
                   │
                   │ Implements
                   ↓
┌─────────────────────────────────────────┐
│           PLUGINS                       │
│  (Extensible - New providers here)      │
│  • SteamExecutor ✓                    │
│  • LocalCacheExecutor ✓                 │
│  • HTTPExecutor (future)                │
│  • RegistryExecutor (future)            │
└─────────────────────────────────────────┘
```

## Data Flow (Per Package)

```
Input: PackageID + SHA256 + Provider
           ↓
┌────────────────────┐
│ ProviderDecision   │ ← Immutable
└────────────────────┘
           ↓
┌────────────────────┐
│ PackageExecution   │ ← Mutable state
└────────────────────┘
           ↓
┌────────────────────┐
│   Executor.Execute │ ← Plugin hook
└────────────────────┘
           ↓
┌────────────────────┐
│ ExecutionResult    │ ← State + Error
└────────────────────┘
```

## Key Invariants (Locked)

1. **One Session per Profile+Packages combination**
2. **One PackageExecution per Package per Session**
3. **Immutable ProviderDecision**
4. **State transitions only forward (except resume)**
5. **SHA256 verification happens exactly once per package**
6. **Retry budget is global per session**

## Error Handling

```
┌─────────────────┐
│ Error Occurs    │
└─────────────────┘
        ↓
   ┌─────────┐
   │ Fatal?  │──Yes──► State = Failed (terminal)
   └─────────┘
        │ No
        ↓
   ┌─────────┐
   │ Budget? │──No───► State = Failed (exhausted)
   └─────────┘
        │ Yes
        ↓
   ┌─────────┐
   │ Backoff │──► Retry after delay
   └─────────┘
```

## Observability Points

```
┌─────────────────────────────────────────┐
│  Provider Decision                      │
│  → provider-trace.json                  │
└─────────────────────────────────────────┘
                   ↓
┌─────────────────────────────────────────┐
│  Session State Changes                  │
│  → session.json                         │
└─────────────────────────────────────────┘
                   ↓
┌─────────────────────────────────────────┐
│  Execution Progress                     │
│  → ProgressCallback                     │
└─────────────────────────────────────────┘
                   ↓
┌─────────────────────────────────────────┐
│  Telemetry (Validation)               │
│  → telemetry.json                       │
└─────────────────────────────────────────┘
```

## Simplification Rules

**Any code that:**
- Adds state transitions outside this graph → **Refactor**
- Bypasses ProviderDecision → **Bug**
- Modifies Session outside Manager → **Bug**
- Creates circular dependencies → **Refactor**

**Any feature that:**
- Requires core change → **Defer to v2.0**
- Fits plugin boundary → **Proceed**
