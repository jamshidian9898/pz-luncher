# RFC-0024: Event Log + Replay System for State Reconstruction

**Date**: June 2026  
**Status**: Implemented  
**Authors**: Launcher Architecture Team

## Executive Summary

This RFC documents the **Event Log + Replay System**, a foundational layer that transforms the launcher-ui frontend from a reactive application into a **domain-driven state transition engine with full audit trail capabilities**.

The system enables:
- ✅ Complete event history persistence and replay
- ✅ State reconstruction at any point in time
- ✅ Automatic anomaly detection and debugging
- ✅ Observable patch validation failures
- ✅ Deterministic state verification

## Problem Statement

### Before: Silent Failures
```
event → reducer → patch → validation ✗ → SKIP (log only)
                                          ↓
                                         UI confused
                                         cause unknown
```

Without observability:
- Patch rejection → silent fail
- No audit trail → no debugging path
- State divergence → undetectable

### Why This Matters
The launcher frontend transitions game state. If events are **silently dropped**:
- Downloads appear "stuck"
- Mod installation fails mysteriously
- Launch state becomes inconsistent
- Users cannot troubleshoot

## Solution Architecture

### Layer 1: PatchSchemaRegistry (Centralized Validation)
```typescript
PatchSchemaRegistry
  ├── tracePatchSchema: Defines allowed trace operations
  ├── downloadsPatchSchema: Defines allowed download operations
  ├── sessionPatchSchema: Defines allowed session operations
  └── serversPatchSchema: Defines allowed server operations
```

**Purpose**: Single source of truth for validation rules per domain.

**Benefits**:
- No validation logic drift
- Easy to audit allowed operations
- Type-safe schema enforcement
- Extensible for future domains

### Layer 2: EventLog Store (Persistence)
```typescript
EventLog
  ├── entries: EventLogEntry[] (max 1000)
  └── Methods:
      ├── addEntry(entry)
      ├── getEntriesBySession(sessionId)
      ├── getEntriesAfter(timestamp)
      └── getStats()
```

**What gets logged**:
```typescript
EventLogEntry {
  id: string                          // unique ID
  event: LauncherEvent               // original event
  patch: LauncherEventPatch          // reducer output
  validationErrors: string[]         // if any
  appliedAt: number                  // timestamp
  status: 'applied' | 'rejected'     // outcome
  sessionId: string                  // audit trail
}
```

### Layer 3: PatchFailureLog Store (Observability)
```typescript
PatchFailureLog
  ├── failures: PatchFailure[] (max 500)
  └── Methods:
      ├── addFailure(failure)
      ├── getFailuresBySession(sessionId)
      ├── getFailuresByDomain(domain)
      └── getRecentFailures(count)
```

**Purpose**: Make silent failures LOUD.

### Layer 4: StateReconstructor (Replay Logic)
```typescript
StateReconstructor
  ├── reconstructAtTimestamp(sessionId, targetTime): StateSnapshot
  ├── reconstructSessionTimeline(sessionId): StateSnapshot[]
  ├── diffStates(sessionId, start, end): Changes
  └── findAnomalyEvent(sessionId, anomalyCheck): EventLogEntry
```

**Capabilities**:
- Reconstruct state at any moment
- Compare state changes
- Find events that caused anomalies
- Verify state consistency

### Layer 5: EventReplay Engine (Deterministic Testing)
```typescript
EventReplay
  ├── replaySession(sessionId): ReplayResult
  ├── replayAll(): ReplayResult
  ├── replaySessionStepwise(sessionId): Generator
  └── verifyStateConsistency(sessionId): Verification
```

**Use cases**:
- Replay failed sessions to debug
- Test new reducer logic against historical events
- Verify state invariants
- Audit state transitions

### Layer 6: EventLogDebugPanel (UI)
Visual component for developers:
- View all events in a session
- Inspect patch failures
- Statistics dashboard
- Replay controls
- Export event logs

## Event Flow with System

```
LauncherEvent
     ↓
reduceLauncherEvent (pure)
     ↓
LauncherEventPatch
     ↓
validateLauncherEventPatch (schema-based)
     ↓
      ├─→ ✓ Valid ──→ dispatchLauncherEvent
      │                    ↓
      │            Log as "applied"
      │                    ↓
      │            Apply patches to stores
      │                    ↓
      │                UI updates
      │
      └─→ ✗ Invalid ──→ Log as "rejected"
                              ↓
                      Record to failureLog
                              ↓
                      EventLogDebugPanel alerts dev
                              ↓
                      StateReconstructor can investigate
```

## Key Design Decisions

### 1. Schema Centralization (PatchSchemaRegistry)
**Why**: Validation logic scattered = future maintenance nightmare.

**Design**: Single registry with domain-specific schemas.

```typescript
const schema = PatchSchemaRegistry.getSchema('trace');
const errors = PatchSchemaRegistry.validateAgainstSchema('trace', patch);
```

**Benefits**:
- ✅ Changes in one place
- ✅ Easy to audit allowed keys
- ✅ Extensible to new domains
- ✅ Type-safe

### 2. Event Log Limiting (max 1000 entries)
**Why**: Infinite logs = memory bloat in long-running sessions.

**Design**: Keep last 1000 entries. Sufficient for debugging most issues.

**Alternative considered**: Persistent storage (IndexedDB). Decision: Not yet; logging layer is ready.

### 3. Failure Log Separation
**Why**: Failures are rare but critical to track.

**Design**: Separate store with max 500 failures. Easy to query all failures in a session.

### 4. StateReconstructor as Utility (not Store)
**Why**: Reconstruction is expensive. Should be on-demand.

**Design**: Pure functions that traverse EventLog and replay patches.

### 5. EventReplay Generator Pattern
**Why**: Need step-by-step debugging without storing intermediate states.

**Design**: Generator yields each replayed event for inspection.

## Invariants and Guarantees

### Invariant 1: Event Order
- Events are always replayed in `appliedAt` order
- Ensures deterministic state reconstruction
- Verified in `EventReplay.replaySession()`

### Invariant 2: Schema Enforcement
- No patch can violate PatchSchemaRegistry rules
- Violations are logged, not silently dropped
- Replay engine re-validates on reconstruction

### Invariant 3: Session Isolation
- Events and failures are always scoped to sessionId
- No cross-session state pollution
- Queries by session are guaranteed isolated

### Invariant 4: Patch Idempotency (Domain-dependent)
- Some patches are idempotent (e.g., `setLaunchState('running')` twice = once)
- Some are not (e.g., `addTraceEvent` is additive)
- Documented per patch type

## Integration Points

### With LauncherStateReducer
- Patches are validated before application
- Validation uses PatchSchemaRegistry
- All patches go through dispatcher logging

### With Zustand Stores
- Event application only through validated dispatcher
- No direct store mutations from events
- StateReconstructor can replay patches atomically

### With UI
- EventLogDebugPanel shows real-time log
- No performance impact (logging is deferred)
- Can be hidden in production with `#ifdef DEBUG`

## Testing and Verification

### Test Scenario 1: Replay After Crash
```typescript
// Simulate crash by clearing stores
stores.clear();

// Replay events
const result = EventReplay.replaySession(sessionId);

// Verify state matches
const consistency = EventReplay.verifyStateConsistency(sessionId);
assert(consistency.isValid);
```

### Test Scenario 2: Find Anomaly
```typescript
// Find event that caused invalid state
const anomalyEvent = StateReconstructor.findAnomalyEvent(
  sessionId,
  (snapshot) => snapshot.session.launchState === 'invalid'
);
```

### Test Scenario 3: Patch Validation Coverage
- Every domain (downloads, trace, session, servers) has schema
- Every patch key has validation rule
- Every validation rule has test case

## Future Extensions

### 1. Event Log Persistence (Phase 2)
- Persist EventLog to IndexedDB
- Survive browser refresh
- Export full session history

### 2. Event Replay in Tests
- Use EventReplay to generate test fixtures
- Deterministic test scenarios
- Regression test infrastructure

### 3. State Timeline Visualization (Phase 3)
- Visual timeline of state changes
- Click to jump to any moment
- Diff viewer for state changes

### 4. Anomaly Detection (Phase 3)
- ML-based pattern detection
- Alert on unexpected state transitions
- Automatic root cause analysis

### 5. Distributed Replay (Phase 3)
- Replay same event sequence on multiple machines
- Verify determinism across environments
- Cross-platform state consistency

## Performance Considerations

### Memory Usage
- EventLog: ~1KB per entry × 1000 = ~1MB
- PatchFailureLog: ~2KB per entry × 500 = ~1MB
- Total: ~2MB overhead for typical session

### CPU Usage
- Validation: O(patch keys) = 5-10 keys per patch
- Replay: O(event count) = linear, ~100μs per event
- StateReconstructor: O(event count) = linear replay

### Optimization Strategies
- Lazy evaluation in StateReconstructor
- Memoization for repeated reconstructions
- Pagination for large event lists in UI

## Rollout Strategy

### Phase 1: Foundation (Current)
✅ PatchSchemaRegistry  
✅ EventLog store  
✅ PatchFailureLog store  
✅ StateReconstructor  
✅ EventReplay engine  
✅ EventLogDebugPanel  

### Phase 2: Production Hardening
- Add error boundary to debug panel
- Performance monitoring
- Event log export functionality
- Failure rate alerting

### Phase 3: Observability Integration
- Send failures to backend for analysis
- Aggregate replay metrics
- Dashboard of state transition patterns

## Risks and Mitigations

| Risk | Mitigation |
|------|-----------|
| Memory bloat in long sessions | Limited EventLog (1000) + PatchFailureLog (500) |
| Performance regression | Deferred logging, lazy reconstruction |
| False positives in anomaly detection | Threshold-based verification, manual review |
| Cross-browser inconsistency | Deterministic replay validates invariants |

## References

- LauncherStateReducer (reducer logic)
- PatchSchemaRegistry (validation schemas)
- EventLogDebugPanel (UI component)
- docs/contracts/launcher-core.md (state contracts)
- RFC-0006-launcher-core.md (reducer design)

## Questions for Review

1. Should EventLog be persisted to IndexedDB by default?
2. Should failures auto-alert to a backend service?
3. Should EventReplay support partial replay (skip certain events)?
4. Should we add event metadata (source, userId, etc.)?

## Sign-Off

- **Architecture**: ✓ Event-driven state machine with validation firewall
- **Observability**: ✓ Full audit trail and failure tracking
- **Debuggability**: ✓ State reconstruction and replay capabilities
- **Testability**: ✓ Deterministic replay for regression testing
- **Production-ready**: ⚠ Needs monitoring integration

---

**Next Steps**:
1. Integrate with launcher backend for event sourcing
2. Build state transition dashboard
3. Add performance monitoring
4. Write integration tests for replay scenarios
