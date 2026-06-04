# Platform Status

**Platform**: PZ Launcher Execution Kernel  
**Status**: 🟢 **FROZEN v1.0**  
**Last Updated**: 2026-06-04  

## Quick Status

| Component | Status | Notes |
|-----------|--------|-------|
| Core Contracts | 🟢 Frozen | See `docs/contracts/guarantees.md` |
| Execution Graph | 🟢 Locked | See `docs/architecture/execution-graph.md` |
| Steam Integration | 🟢 Stable | Chaos + Shadow validated |
| Validation Layer | 🟢 Ready | Live/Chaos/Shadow modes |
| Extended Campaign | 🟢 Ready | Long-run validation (100-1000+ sessions) |
| Reliability Metrics | 🟢 Ready | SLO/SLI tracking |
| HTTP Provider | ⚪ Planned | Plugin-only development |
| Registry Provider | ⚪ Planned | Plugin-only development |

## What "Frozen" Means

> Core interfaces are **immutable**.

- No signature changes to Session Manager
- No changes to Executor interface
- No changes to State Machine
- No changes to ProviderDecision
- Plugin boundary is the **only** extension point

## Adding New Providers

New download sources (HTTP, Registry, etc.) = **Plugins**

```go
// Example: HTTP Provider (future plugin)
type HTTPExecutor struct { ... }

func (e *HTTPExecutor) Execute(ctx context.Context, 
    exec *PackageExecution) (*PackageExecution, error) {
    // Implement HTTP download logic
    // Must respect all platform guarantees
}

// Register:
composite := session.NewCompositeExecutor()
composite.Register("HTTP", &HTTPExecutor{})
```

## Validated Guarantees

| Guarantee | Status | Evidence |
|-----------|--------|----------|
| Determinism | ✅ Validated | Chaos + Shadow tests pass |
| Bounded Time | ✅ Validated | 5min timeout enforced |
| Idempotency | ✅ Validated | Session replay tests |
| Atomicity | ✅ Validated | No partial states in traces |
| Drift < 10% | ⏳ Pending | Extended campaign execution |

## Run Validation

### Chaos Tests
```bash
go run apps/chaos-cli/main.go -list
go run apps/chaos-cli/main.go
```

### Shadow Validation
```bash
# Live (real Steam)
go run apps/validation-cli/main.go -mode=live

# Chaos (failure injection)
go run apps/validation-cli/main.go -mode=chaos

# Shadow (compare both)
go run apps/validation-cli/main.go -mode=shadow

# Detailed comparison
go run apps/validation-cli/main.go -compare
```

### Campaign CLI
```bash
# Check version
go run apps/campaign-cli/main.go -version

# Run validation campaign
go run apps/campaign-cli/main.go \
  -runs=100 \
  -mode=shadow \
  -name=v1.0-validation
```

### Extended Validation Campaign
```bash
# 100-session validation
go run apps/campaign-cli/main.go -runs=100 -mode=shadow

# Continuous until stopped
go run apps/campaign-cli/main.go -infinite -mode=shadow

# With custom parameters
go run apps/campaign-cli/main.go -runs=500 -interval=10s -concurrent=5
```

## Architecture at a Glance

```
┌─────────────────┐
│   Profile       │
│   + Packages    │
└────────┬────────┘
         ↓
┌─────────────────┐
│ Provider Router │ ← Selects best provider
└────────┬────────┘
         ↓
┌─────────────────┐
│   Executor      │ ← Plugin boundary (FROZEN interface)
│   (Steam/HTTP/  │
│   Registry/etc) │
└────────┬────────┘
         ↓
┌─────────────────┐
│   Session       │ ← State machine (FROZEN)
│   Manager       │
└─────────────────┘
```

## Breaking Change Policy

**v1.x → v2.0 requires:**
1. RFC proposal
2. All maintainer approval
3. Migration guide
4. 6-month deprecation period

**No breaking changes planned for v1.x.**

## Repository Structure

```
pz-launcher/
├── apps/
│   ├── launcher-core/    # Main launcher
│   ├── chaos-cli/        # Chaos testing
│   ├── validation-cli/   # Shadow validation
│   └── pz-agent/         # Agent service (future)
├── libs/
│   ├── contracts/        # FROZEN interfaces
│   ├── session/          # FROZEN execution engine
│   ├── chaos/            # Testing framework
│   ├── validation/       # Live validation
│   └── providers/        # Provider interface
├── docs/
│   ├── contracts/        # Guarantees
│   ├── architecture/     # Execution graph
│   └── domain/           # Domain docs
└── STATUS.md             # This file
```

## SLOs (Service Level Objectives)

| SLO | Target | Measurement |
|-----|--------|-------------|
| **Availability** | ≥ 99% | (Total - Fatal Failures) / Total |
| **Success Rate** | ≥ 95% | Successful / Total Executions |
| **Drift Rate** | < 10% | Drift Detections / Total Comparisons |
| **P99 Latency** | < 60s | 99th percentile execution time |

**Reliability Score**: 0-100 (25 points per SLO met)  
**Production Threshold**: ≥ 80/100

## Next Steps

### Immediate (Validation)
1. ⏳ **Execute extended campaign** — 100+ sessions
2. ⏳ **Verify all SLOs met** — availability, success, drift, latency
3. ⏳ **Collect reliability metrics** — failure distribution
4. ⏳ **Document results** — production readiness checklist

### After Validation (Production)
5. ✅ **Platform frozen** — contracts locked
6. ✅ **Plugin guide** — HTTP/Registry as plugins
7. ⏳ **Production deployment** — with monitoring

### Documents
- `docs/contracts/production-readiness.md` — Full checklist

## Contact

For plugin development questions, see:
- `docs/contracts/guarantees.md` — Platform contracts
- `docs/architecture/execution-graph.md` — Execution flow
- `docs/contracts/production-readiness.md` — Production checklist
- `libs/session/executor.go` — Plugin interface
