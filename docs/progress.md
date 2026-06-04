# Progress Report

This file tracks the current state of repository scaffolding, documentation, and agent progress.

## Summary

- Created a workspace-level scaffolding agent: `.agent.md`
- Generated a project monorepo scaffold with:
  - `apps/`
  - `libs/`
  - `docs/`
  - `docs/rfc/`
- Created core documentation files for vision, architecture, service boundaries, APIs, manifest schema, profile isolation, and roadmap
- Added domain model docs in `docs/domain/`
- Added service contract docs in `docs/contracts/`
- Added OpenAPI definitions in `schemas/openapi/`
- Added shared Go contract type stubs in `libs/contracts/` and provider interface stubs in `libs/providers/`
- Added launcher-core and agent contract docs in `docs/contracts/`
- Added foundation-layer app skeletons in `apps/launcher-core` and `apps/pz-agent`
- Added resolver, game, and launchstate packages in `libs/`
- Added twenty-one RFCs in `docs/rfc/` describing architecture, provider model, cache system, download session design, manifest resolution, profile build, game launch, save isolation, and state machine

## RFC files completed

- `docs/rfc/0001-manifest-format.md`
- `docs/rfc/0002-service-boundaries.md`
- `docs/rfc/0003-content-registry.md`
- `docs/rfc/0004-download-manager.md`
- `docs/rfc/0005-agent-lifecycle.md`
- `docs/rfc/0006-launcher-core.md`
- `docs/rfc/0007-profile-isolation.md`
- `docs/rfc/0008-provider-system.md`
- `docs/rfc/0009-cache-system.md`
- `docs/rfc/0010-steam-provider.md`
- `docs/rfc/0011-download-session.md`
- `docs/rfc/0012-content-addressable-storage.md`
- `docs/rfc/0013-manifest-resolution.md`
- `docs/rfc/0014-provider-priority.md`
- `docs/rfc/0015-profile-build.md`
- `docs/rfc/0016-game-launcher.md`
- `docs/rfc/0017-save-isolation.md`
- `docs/rfc/0018-state-machine.md`
- `docs/rfc/0019-package-resolution.md`
- `docs/rfc/0020-game-installation.md`
- `docs/rfc/0021-launch-state-machine.md`

## Agent state

- Agent name: `project-zomboid-launcher-scaffolder`
- Purpose: scaffold the monorepo and documentation from `ProjectBaseDocs`
- Current status: created and ready for use
- Recommended use: ask the agent to generate additional service skeletons, code stubs, or new docs based on the proposal

## Current state

- Documentation: 95%
- Architecture: 90%
- Domain model: 90%
- Contracts: 90%
- Shared libraries: 75%
- Services: 0%
- Launcher Core: 55% (platform frozen — reliability metrics + campaign runner ready)
- UI: 0%

### Recent engineering progress (Phase 3 — Platform Grade)

- **Download Session Manager** — Execution engine for provider decisions.
  - `libs/session/manager.go`: Full session lifecycle with Create/Load/Execute/Save/GetTrace.
  - `libs/session/executor.go`: Pluggable executor interface.
  - **Steam Hardening Layer** — Production-grade Steam integration:
    - `libs/session/steam_executor.go`: Multi-strategy download chain with hardened reliability
    - `libs/session/steam_api.go`: Steam Web API client with **rate limiting** (prevents API bans)
    - `libs/session/steamcmd.go`: SteamCMD fallback for authenticated/private items
    - `libs/session/workshop_mapping.go`: **Mod name → Workshop ID resolver** (local cache + registry)
    - `libs/session/ratelimit.go`: Token bucket rate limiter (configurable for Steam API limits)
    - `libs/session/failure_inject.go`: **Chaos testing framework** — simulates real-world failures
    - `libs/session/progress.go`: Internal progress event streaming (bytes, speed, percent)
  - **Chaos Validation Suite** — Controlled failure testing (quick validation):
    - `libs/chaos/scenario.go`: Chaos test scenarios with controlled failure injection
    - `libs/chaos/runner.go`: Scenario execution engine with expectation validation
    - `libs/chaos/replay.go`: Deterministic replay engine for behavior validation
    - `apps/chaos-cli/main.go`: CLI tool for running chaos tests
    - **5 Preset Scenarios**: flaky network, Steam API down, hash mismatch, partial download, retry exhaustion
  - **Shadow Validation Layer** — Live vs simulation comparison:
    - `libs/validation/shadow_executor.go`: Dual-mode executor (live/chaos/shadow)
    - `libs/validation/drift.go`: Drift detection between live and chaos results
    - `libs/validation/telemetry.go`: Real-world telemetry collection
    - `apps/validation-cli/main.go`: CLI for live validation and drift detection
    - **Features**: ModeLive (real APIs), ModeChaos (failure injection), ModeShadow (compare both)
  - **Extended Validation Campaign** — Long-run reliability testing:
    - `libs/validation/metrics.go`: SLO/SLI metrics tracking (availability, success rate, drift, latency)
    - `libs/validation/campaign.go`: Long-run campaign scheduler (100-1000+ sessions)
    - `apps/campaign-cli/main.go`: CLI for continuous validation campaigns
    - **SLOs**: Availability ≥99%, Success Rate ≥95%, Drift <10%, P99 Latency <60s
  - **Reliability Features**:
    - Workshop ID mapping with local cache and registry fallback
    - Rate limiting (10 tokens, 1 req/sec refill for Steam API)
    - Failure injection for chaos testing (network timeout, HTTP errors, hash mismatch, partial download)
    - SHA256 hash verification at multiple stages
    - Retry budget system (global budget per session)
    - Atomic file operations (temp → final)
  - `libs/session/composite_executor.go`: Router that delegates to provider-specific executors.
  - **Idempotency**: Same inputs → same session ID → resumes from persisted state.
  - **Resume support**: Can restart mid-session, skips already-completed packages.
  - Session trace combines provider decisions + execution results.
  - State machine extended: `CreatingSession` → `Downloading` → `Verifying` → `Materializing`.

- **Structured Provider Trace** — Added comprehensive decision tracing before Download Session.
  - Enhanced `ProviderDecision` with timing, detailed attempts, fallback chain, and reasoning.
  - Human-readable trace output + structured JSON in `profiles/<server>/provider-trace.json`.
  - Enables visibility into "why this decision was made".

- Deterministic demo fixture seeder and `--demo` flag.

### Recent engineering progress (Phase 3)

- Launcher Core now compiles and runs an offline flow (resolve → prepare profile → launch).
- Implemented `ProfileBuilder` materialization: cache blobs are linked into `profiles/<server>/mods/<pkgID>/` using symlinks (or copies on Windows) with SHA256 and size integrity checks; idempotent behavior.
- LocalCacheProvider supports copying blobs from `cache/sha256/<sha>` into destinations.

## What you can do next

1. Review the RFCs in `docs/rfc/`
2. Add implementation placeholders in `apps/` and `libs/`
3. Use the `.agent.md` agent for follow-up scaffolding tasks
4. Track future progress by updating this file with new work items and states

## Future work items

### Next: Execute Extended Validation Campaign (Current Phase)
- ⏳ Run: `go run apps/campaign-cli/main.go -runs=100` — 100 session validation
- ⏳ Run: `go run apps/campaign-cli/main.go -infinite` — Continuous until stopped
- ⏳ Run: `go run apps/campaign-cli/main.go -mode=shadow` — Shadow comparison
- ⏳ Verify: SLOs met (availability ≥99%, success ≥95%, drift <10%)
- ⏳ Collect: Reliability metrics and failure distribution
- ⏳ Document: Campaign results and production readiness
- ✅ Plugin guide: HTTP/Registry = **plugins only** (no core changes)

### After Stabilization: HTTP/Registry (Plugins)
- HTTP Provider = new plugin implementing Executor
- Registry Provider = new plugin implementing Executor
- Multi-source = CompositeExecutor enhancement (within plugin boundary)

### Later phases:
- Add service skeletons and API definitions inside `apps/`
- Add shared library stubs inside `libs/`
- Expand the launcher and agent design into implementation tasks
- Add more RFCs for authentication, provider integration, and distribution
