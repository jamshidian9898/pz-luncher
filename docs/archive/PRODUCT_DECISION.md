# Product Decision — Phase 1 Product Execution

**Date**: 2026-06-04  
**Status**: ✅ **LOCKED** — Launcher for players (Path A)  
**Supersedes**: Open strategic choice in `ARCHITECTURAL_DECISION.md` (reference only)

---

## Decision

> **We are building a player-ready Project Zomboid launcher.**  
> **No new infrastructure/architecture RFCs. Domain RFCs and features only.**

| Choice | Value |
|--------|--------|
| Path | **A — Product** |
| MVP target | **8 weeks** from kickoff |
| MVP definition | Join server → download mods → launch game with isolated profile |

---

## Infrastructure — DONE (do not extend)

The following are **complete**. Treat as frozen unless a domain RFC requires a bug fix:

- Event Bus, Reducer, Validation Firewall
- Replay Engine, Snapshot Engine, Compaction
- Go Session Manager, Executor, Steam integration (see `STATUS.md`)

**Do not start**: RFC-0026+, plugin system, multi-game, distributed replay, analytics platform, SDK, marketplace.

---

## What we build now

| Area | Status |
|------|--------|
| Domain RFCs 0030–0035 | ✅ Implemented (`libs/pipeline`, …) |
| RFC-0036 Settings | ✅ `libs/settings` + schema |
| Shared contracts | ✅ `shared/contracts` → `make contracts` |
| Registry UI | ✅ `RegistryLauncherApi` + `public/registry` |
| Dev without Wails | ✅ `go run ./apps/dev-api` |

Full timeline: [docs/PRODUCT_ROADMAP.md](docs/PRODUCT_ROADMAP.md)

---

## Deferred (after MVP ships)

- Plugin system, multi-game support
- Remote event log, distributed replay
- Directory / Registry / Manifest **microservices** (use local manifest + launcher-core until then)
- Hybrid phase enforcement (`PHASE_ENFORCEMENT_*`) — not required for Path A

---

## Document map (read this, not the old decision stack)

| Need | Read |
|------|------|
| What to build this week | [docs/PRODUCT_ROADMAP.md](docs/PRODUCT_ROADMAP.md) |
| Domain specs | [docs/DOMAIN_RFC_INDEX.md](docs/DOMAIN_RFC_INDEX.md) |
| Platform / Go status | [STATUS.md](STATUS.md) |
| Architecture vision (background) | [docs/vision.md](docs/vision.md) |
| Old Path A/B/C debate | `ARCHITECTURAL_DECISION.md` (historical) |

---

## Sign-off

| Role | Name | Date |
|------|------|------|
| Product owner | | |
| Engineering lead | | |
