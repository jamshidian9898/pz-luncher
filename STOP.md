# STOP — Platform v1.0 (Calibration Phase)

> **Product team**: Primary work is [PRODUCT_DECISION.md](PRODUCT_DECISION.md) / RFC 0030–0035.  
> This file applies to **Go platform validation** (optional parallel track).

**Date**: 2026-06-04  
**Status**: � **CALIBRATING — Core Frozen, Validation Layer Tuning**  

---

## What This Means

> **Core Platform: FROZEN**  
> **Validation Layer: CALIBRATING**  
> **Steam Integration: BETA (needs offline fixture mode)**

From the first campaign run (June 4, 2026), reality showed:
- ❌ Campaign used fake data (campaign-pkg-0) → Fixed: Now uses real workshop IDs
- ❌ Drift detection bug (100% drift rate) → Fixed: Proper comparison counting
- ❌ Steam integration incomplete → Fixed: Added ModeOfflineFixtures

**Now uses: Real metadata + Local fixtures (no Steam API needed for validation)**

---

## What You CAN Do

### ✅ Setup Fixture Mode (PRIORITY)
1. Create fixtures directory
2. Add sample .zip files (any files work for testing)
3. Update fixture registry in campaign
4. Run offline campaign

### ✅ Execute Validation Campaign (After Fixes)
```bash
# Required before production
go run apps/campaign-cli/main.go -runs=1000 -mode=shadow
```

### ✅ Write Reports
- `reports/campaign-2026-06-04/metrics.json`
- `reports/campaign-2026-06-04/drift-report.md`
- `reports/campaign-2026-06-04/summary.html` (optional dashboard)

### ✅ Fix Bugs
- Memory leaks (if found in campaign)
- Goroutine leaks (if found in campaign)
- Race conditions (if found in campaign)

### ✅ Plugins (HTTP/Registry)
```go
// OK: New plugin implementing existing interface
type HTTPExecutor struct { ... }
func (e *HTTPExecutor) Execute(...) { ... }
```

### ✅ UI (Wails)
```
Wails UI → launcher-core API → Session Manager
```

---

## Current State (Calibration Phase)

| Component | Status | Action |
|-----------|--------|--------|
| Session Manager | 🟢 Frozen | No changes |
| Executor | 🟢 Frozen | No changes |
| Steam Integration | � Beta | Needs test mode + wiring fixes |
| Chaos Tests | 🟢 Complete | Run only |
| Shadow Validation | � Bug Fixed | Drift counting fixed |
| Campaign Runner | � Data Fixed | Now uses real workshop IDs |
| SLO Metrics | 🟢 Complete | Monitor only |
| HTTP Provider | ⚪ Planned | Plugin only |
| Registry Provider | ⚪ Planned | Plugin only |
| Wails UI | ⚪ Planned | Consumer only |

---

## Validation Required

Before any production use:

### Phase 1: Quick (5 min)
```bash
go run apps/chaos-cli/main.go
# Expect: 100% pass
```

### Phase 2: Medium (30 min)
```bash
go run apps/validation-cli/main.go -mode=shadow
# Expect: Drift < 10%
```

### Phase 3: Extended (Hours/Days)
```bash
go run apps/campaign-cli/main.go -infinite -mode=shadow
# Run for hours
# Monitor: htop, memory, goroutines
```

### Phase 4: Load Test (Optional)
```bash
go run apps/campaign-cli/main.go -runs=1000 -concurrent=20
# Watch for:
# - Memory leaks
# - Goroutine leaks
# - Retry storms
```

---

## SLOs to Prove

| SLO | Target | Evidence |
|-----|--------|----------|
| Availability | ≥ 99% | Campaign metrics |
| Success Rate | ≥ 95% | Campaign metrics |
| Drift Rate | < 10% | Drift report |
| P99 Latency | < 60s | Latency histogram |
| **Reliability Score** | **≥ 80/100** | **Final report** |

---

## After Validation

### If SLOs Met (Score ≥ 80)
```
1. Tag: v1.0.0
2. Write: Release notes
3. Deploy: To production (with monitoring)
4. Monitor: Continuous campaign
```

### If SLOs Not Met
```
1. Analyze: Failure distribution
2. Fix: Bugs (not architecture)
3. Re-run: Campaign
4. Repeat: Until SLOs met
```

---

## Architecture Summary

```
┌─────────────────────────────────────────┐
│         CONSUMERS (External)            │
│  • Wails UI (future)                    │
│  • HTTP Provider (future plugin)        │
│  • Registry Provider (future plugin)     │
└─────────────────────────────────────────┘
                   │
                   │ Uses
                   ↓
┌─────────────────────────────────────────┐
│         PLATFORM CORE (Frozen)          │
│                                         │
│  Session Manager                        │
│    ├── Create/Load/Save                 │
│    ├── Execute                          │
│    └── GetTrace                         │
│                                         │
│  Executor Interface                     │
│    ├── Execute(context, exec)           │
│    └── Returns (exec, error)            │
│                                         │
│  State Machine                          │
│    ├── Pending → Downloading            │
│    ├── Downloading → Verifying          │
│    ├── Verifying → Complete/Failed      │
│    └── Failed → Downloading (resume)    │
│                                         │
│  Validation (External to Core)          │
│    ├── Chaos Tests                      │
│    ├── Shadow Validation                │
│    └── Campaign Runner                  │
│                                         │
└─────────────────────────────────────────┘
                   │
                   │ Implements
                   ↓
┌─────────────────────────────────────────┐
│         PLUGINS (Extensible)            │
│                                         │
│  ✅ SteamExecutor (Built-in)            │
│  ✅ LocalCacheExecutor (Built-in)       │
│  ⚪ HTTPExecutor (Future)                │
│  ⚪ RegistryExecutor (Future)            │
│                                         │
└─────────────────────────────────────────┘
```

---

## Common Pitfalls to Avoid

### ❌ "Let's add just one more feature..."
> Result: Complexity, bugs, delay

### ❌ "The architecture could be cleaner..."
> Result: Breaking changes, instability

### ❌ "Let's optimize this..."
> Result: Premature optimization, bugs

### ❌ "What about distributed execution?"
> Result: Massive scope creep

### ✅ "Let's run the campaign..."
> Result: Evidence, confidence, production-ready

---

## The Only Metric That Matters Now

```
Reliability Score ≥ 80/100
```

Everything else is distraction.

---

## Contact

For questions about this STOP:
- See: `docs/contracts/guarantees.md`
- See: `docs/contracts/production-readiness.md`
- See: `STATUS.md`

---

## TL;DR

> **Architecture: Complete ✅**  
> **Validation: Pending ⏳**  
> **Production: Blocked 🚫**

**Execute campaign. Prove SLOs. Then release.**

Nothing else.
