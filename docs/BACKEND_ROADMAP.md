# Backend Roadmap — v2.0.0

**Status**: Phase A in progress  
**Decision**: [PRODUCT_DECISION.md](../PRODUCT_DECISION.md)  
**RFCs**: [0052](rfc/0052-backend-core-api.md) · [0053](rfc/0053-agent-enrollment.md) · [0054](rfc/0054-backend-content-store.md) · [0055](rfc/0055-join-response-contract.md)

---

## Phase A — Backend Core (current)

Goal: Backend serves Launcher with servers list, join response, and content download.

### RFC-0052: Backend Core API

| Task | Status | File |
|------|--------|------|
| `GET /api/v1/health` | ✅ Done | `apps/backend/internal/api/router.go:38` |
| `GET /api/v1/servers` | ✅ Done | `apps/backend/internal/api/router.go:42` |
| `GET /api/v1/servers/{serverId}` | ✅ Done | `apps/backend/internal/api/router.go:49` |
| `GET /api/v1/agents` | ✅ Done | `apps/backend/internal/api/router.go:60` |
| `POST /api/v1/join/{serverId}` | ✅ Done | `apps/backend/internal/join/join.go` |
| `GET /api/v1/download/{sha256}` | 🟡 In progress | `apps/backend/internal/storage/` |
| `GET /metrics` (Prometheus) | ✅ Done | `apps/backend/internal/metrics/` |

### RFC-0054: Backend Content Store

| Task | Status | Notes |
|------|--------|-------|
| SHA256 content identity design | ✅ Done | Spec + interface defined |
| Local blob storage interface | 🟡 In progress | `apps/backend/internal/storage/` |
| Content record schema | ✅ Done | See RFC-0054 |
| `GET /api/v1/download/{sha256}` handler | ⬜ Pending | Depends on blob storage |
| Content source resolution (agent → steamcmd → upload) | ⬜ Pending | Phase A close-out |
| Integration tests | ⬜ Pending | — |

### RFC-0055: JoinResponse Contract

| Task | Status | Notes |
|------|--------|-------|
| JoinResponse struct definition | ✅ Done | `apps/backend/internal/join/join.go` |
| Manifest resolution in join handler | ✅ Done | — |
| Download plan generation | ✅ Done | — |
| Signed URL issuance | ⬜ Pending | After blob storage |

**Phase A exit criteria**: Launcher can `POST /join`, receive a valid JoinResponse with download URLs, and fetch mod blobs from the backend.

---

## Phase B — Agent Control Plane (next)

Goal: Agent enrolls with Backend and keeps manifest + content up to date.

See [AGENT_ROADMAP.md](AGENT_ROADMAP.md) for task breakdown.

### RFC-0053: Agent Enrollment (backend side)

| Task | Status | Notes |
|------|--------|-------|
| `POST /api/v1/agents/register` | ✅ Done | Open registration, auto-creates server record; issues token |
| `POST /api/v1/agents/heartbeat` | ✅ Done | Token-authed; updates agent gauges |
| `PUT /api/v1/manifests/{serverId}` | ✅ Done | Versioned store + history/diff endpoints (B4) |
| `PUT /api/v1/blobs/{sha256}` (+ HEAD) | ✅ Done | Agent **pushes** content; replaces backend-pull design |
| Heartbeat staleness → online/degraded/offline | ✅ Done | Computed in `internal/registry` (90s / 5min) |
| Enrollment token generation (admin/operator) | ⬜ Pending | Single-use, 24h TTL, serverId-scoped |
| Revocation / Revoked terminal state | ⬜ Pending | — |

**Phase B exit criteria — met 2026-07-03**: Agent registers, heartbeats, pushes
blobs and manifest; Launcher `POST /join` builds a byte-identical profile from
agent-published content (validated E2E via `join-cli -backend`).

---

## Not in scope (post-MVP)

- S3 / MinIO / R2 storage backends (local only for MVP)
- Admin API for operator token management
- Multi-agent load balancing
- Agent auto-update
