# Domain RFC Index — Phase 1 Product / v2.0.0

**Active execution**. Infrastructure RFCs (0022–0025) are **complete** — do not extend without a product bug.

**Decision**: [PRODUCT_DECISION.md](../PRODUCT_DECISION.md)  
**Roadmap**: [PRODUCT_ROADMAP.md](PRODUCT_ROADMAP.md)

---

## v2.0.0 Canonical Architecture

| RFC | Title | Status |
|-----|-------|--------|
| [0050](rfc/0050-v2-architecture-rebaseline.md) | v2.0.0 Architecture Rebaseline | **Canonical — read first** |
| [0051](rfc/0051-v2-phase-plan.md) | v2.0.0 Phase Plan (A/B/C) | **Active** |

**Rule**: Launcher communicates exclusively with the Backend. Agents and SteamCMD are Backend-internal infrastructure invisible to the Launcher.

---

## Phase A — Backend Core

| RFC | Title | Status |
|-----|-------|--------|
| [0052](rfc/0052-backend-core-api.md) | Backend Core API | **Active** |
| [0053](rfc/0053-agent-enrollment.md) | Agent Enrollment | **Active** |
| [0054](rfc/0054-backend-content-store.md) | Backend Content Store | **Active** |
| [0055](rfc/0055-join-response-contract.md) | JoinResponse Contract | **Active — primary contract** |

## Phase B — Agent

| RFC | Title | Status |
|-----|-------|--------|
| [0056](rfc/0056-agent-minimal.md) | Agent Minimal | **Active** |

## Launcher Surface Freeze

The Launcher API surface is frozen at:

```text
GetServers() · Join() · Download() · Launch() · Settings() · Diagnostics()
```

No new Launcher features during Phase A and B. The product from here is **Backend + Agent**. The Launcher is the first client of the Platform.

---

## Build order (strict)

| Order | RFC | Title | Week |
|-------|-----|-------|------|
| 1 | [0030](rfc/0030-server-manifest-v1.md) | Server Manifest v1 | ✅ `libs/manifestv1` |
| 2 | [0031](rfc/0031-mod-dependency-resolver.md) | Mod Dependency Resolver | ✅ `libs/modplan` |
| 3 | [0032](rfc/0032-download-manager.md) | Download Manager | ✅ `libs/download` |
| 4 | [0033](rfc/0033-installation-pipeline.md) | Installation Pipeline | ✅ `libs/pipeline` |
| 5 | [0034](rfc/0034-profile-system.md) | Profile System | ✅ profile + snapshot |
| 6 | [0035](rfc/0035-game-launch-flow.md) | Game Launch Flow | ✅ `pipeline.Launch` |
| 7 | [0036](rfc/0036-settings.md) | Launcher Settings | ✅ `libs/settings` + schema |

**Shared types**: [shared/contracts/README.md](../shared/contracts/README.md) → `make contracts`

---

## Superseded / reference only (do not rewrite)

These informed domain RFCs; implementation follows **0030–0035**:

| Legacy | Use instead |
|--------|-------------|
| RFC-0001, manifest-schema.md | RFC-0030 |
| RFC-0013, RFC-0019 | RFC-0031 |
| RFC-0011, RFC-0004 | RFC-0032 |
| RFC-0015, RFC-0020, profile-system.md | RFC-0033, RFC-0034 |
| RFC-0016, RFC-0021 | RFC-0035 |

---

## Infrastructure (frozen — no new RFCs)

| RFC | Topic |
|-----|--------|
| 0022 | UI events |
| 0023 | State management |
| 0024 | Event log + replay |
| 0025 | Snapshot + compaction |

---

## Cancelled until post-MVP

- RFC-0026+ (platform plugins, multi-game, distributed replay)
- Hybrid enforcement docs (`PHASE_ENFORCEMENT_*`) — Path A does not require them

---

## Foundation RFCs (0001–0021)

Still valid for **background** and Go platform design. Product work traces to domain RFCs above.

Full list: [docs/README.md](README.md)
