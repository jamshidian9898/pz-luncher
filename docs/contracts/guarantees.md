# Platform Guarantees (Contract Lock v1.0)

**Status**: LOCKED  
**Version**: 1.0  
**Date**: 2026-06-04  

This document defines the formal guarantees provided by the execution platform. Once locked, these contracts are immutable — all future development must operate within these boundaries.

## Core Philosophy

> "The platform provides execution. Everything else is a plugin."

## Locked Interfaces

### 1. Session Manager (`libs/session/manager.go`)

```go
// FROZEN — Do not modify signature
type Manager interface {
    CreateSession(id, profile string, decisions []ProviderDecision) (*Session, error)
    LoadSession(id string) (*Session, error)
    SaveSession(session *Session) error
    Execute(ctx context.Context, session *Session, executor Executor) error
    GetTrace(session *Session) *SessionTrace
}
```

**Guarantees:**
- **Idempotency**: Same inputs → same session ID
- **Persistence**: Session state survives process restarts
- **Atomicity**: Session updates are atomic (all-or-nothing)
- **Resume**: Interrupted sessions can resume from last persisted state

**Invariants:**
- Session ID is deterministic hash of (profile + package list + version)
- State transitions: Pending → Downloading → Verifying → Complete/Failed
- No state transition backward except resume (Failed → Downloading)

---

### 2. Executor (`libs/session/executor.go`)

```go
// FROZEN — Do not modify signature
type Executor interface {
    Execute(ctx context.Context, exec *PackageExecution) (*PackageExecution, error)
}
```

**Guarantees:**
- **Determinism**: Same PackageExecution → same outcome (accounting for network variance)
- **Bounded Execution**: Execution completes within bounded time (configurable, default 5min)
- **Cancellation**: Context cancellation is honored within 1 second
- **Error Classification**: Errors are either retryable or fatal

**Plugin Boundary:**
- New providers = new Executor implementations
- No changes to Executor interface

---

### 3. Provider Decision (`libs/contracts/provider_decision.go`)

```go
// FROZEN — Do not modify struct
type ProviderDecision struct {
    PackageID       string
    PackageVersion  string
    PackageSHA256   string
    ChosenProvider  string
    DecisionAt      time.Time
    Attempts        []ProviderAttempt
    FinalReason     string
}
```

**Guarantees:**
- **Immutability**: Once created, decision cannot change
- **Traceability**: Every provider attempt is recorded
- **Verifiability**: SHA256 is always present for integrity checks

---

### 4. Validation Layer (`libs/validation/`)

```go
// FROZEN — Do not modify
type ShadowExecutor struct {
    // Dual-mode executor for live/chaos/shadow execution
}

// Guarantees:
// - Drift detection between live and chaos execution
// - Drift rate < 10% acceptable
// - Telemetry collection for real-world metrics
```

---

## Formal Guarantees

### Execution Guarantees

| Guarantee | Definition | Bound |
|-----------|------------|-------|
| **Determinism** | Same input → same state transition | 100% (outcome) |
| **Bounded Time** | Execution completes or fails | 5 minutes max |
| **Idempotency** | Re-execution produces same result | 100% |
| **Atomicity** | Partial completion impossible | 100% |
| **Resumability** | Can resume from any persisted state | 100% |

### Fault Guarantees

| Guarantee | Definition | Bound |
|-----------|------------|-------|
| **Retry Budget** | Maximum attempts per session | 10 (configurable) |
| **Rate Limiting** | API calls per second | 1 req/sec (Steam) |
| **Chaos Isolation** | Failure injection never leaks | 100% |
| **Drift Bound** | Acceptable live/chaos divergence | < 10% |

### Observability Guarantees

| Guarantee | Definition | Bound |
|-----------|------------|-------|
| **Trace Completeness** | Every decision recorded | 100% |
| **Progress Streaming** | Real-time download progress | < 1s latency |
| **State Visibility** | Current state always queryable | 100% |
| **Telemetry Accuracy** | Metrics reflect reality | ±5% |

---

## Plugin Contract

### What is a Plugin?

```go
// A plugin is any type that implements Executor
type MyCustomExecutor struct { ... }

func (e *MyCustomExecutor) Execute(ctx context.Context, 
    exec *PackageExecution) (*PackageExecution, error) {
    // Plugin logic here
    // - Can use any external APIs
    // - Must respect context cancellation
    // - Must update exec.State appropriately
    // - Must return non-nil error on failure
}
```

### Plugin Rules

1. **No Core Changes**: Plugins cannot modify Session Manager, Executor interface, or ProviderDecision
2. **State Compliance**: Must use `PackageExecutionState` enum values
3. **Error Handling**: Must classify errors as retryable or fatal
4. **Respect Context**: Must check `ctx.Done()` and exit promptly
5. **Progress Updates**: Should call progress callback if provided

### Plugin Lifecycle

```
1. Register: CompositeExecutor adds plugin to routing table
2. Decision: ProviderLogic selects plugin based on availability
3. Execute: SessionManager calls plugin.Execute()
4. Observe: SessionManager records outcome in trace
5. Persist: SessionManager saves state
```

---

## Breaking Change Policy

### What Constitutes a Breaking Change?

- Modifying any FROZEN interface signature
- Removing or renaming exported types/functions
- Changing guarantee bounds without major version bump
- Altering state machine transitions

### Change Process

```
1. Propose change via RFC
2. Evaluate impact on all plugins
3. If breaking → require MAJOR version bump
4. Migration path documented
5. Deprecation period (min 6 months)
```

---

## Stability Levels

### Level 1: Core (Frozen)
- Session Manager
- Executor Interface
- ProviderDecision
- PackageExecutionState

**Changes**: MAJOR version bump only  
**Testing**: Chaos + Shadow validation required  
**Review**: All maintainers must approve

### Level 2: Built-in Plugins (Stable)
- SteamExecutor
- LocalCacheExecutor
- CompositeExecutor

**Changes**: MINOR version bump acceptable  
**Testing**: Unit + integration tests required  
**Review**: 1 maintainer approval

### Level 3: Extensions (Flexible)
- HTTP provider (future)
- Registry provider (future)
- Custom executors (external)

**Changes**: No version bump required  
**Testing**: Self-tested  
**Review**: External maintainers

---

## Migration Guide (Future Versions)

### v1.x → v2.0 (Hypothetical)

If breaking changes are ever needed:

```
1. Freeze v1.x branch
2. Maintain v1.x for 12 months
3. Provide migration tool
4. Document all breaking changes
5. Dual-support period for plugins
```

---

## Current Plugin Registry

| Plugin | Type | Status | Maintainer |
|--------|------|--------|------------|
| SteamExecutor | Executor | Stable | Core |
| LocalCacheExecutor | Executor | Stable | Core |
| CompositeExecutor | Router | Stable | Core |

---

## Summary

> "The platform is now a contract, not just code."

These guarantees enable:
- External plugin development with confidence
- Long-term stability for dependent systems
- Clear boundaries for feature development
- Verifiable correctness through testing

**Any code that violates these guarantees is a bug.**
