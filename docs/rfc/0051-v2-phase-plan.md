# RFC-0051: v2.0.0 Phase Plan

**Status**: Active — v2.0.0  
**Depends on**: [RFC-0050](0050-v2-architecture-rebaseline.md)  
**Supersedes**: v1.x roadmap assumptions (servers.json, SteamProvider, Manifest-as-transport)

---

## The canonical sentence

> **Backend = Single Control Plane**

This is the defining statement of v2.0.0.

---

## v1.x → v2.0.0 transition line

| Axis | v1.x | v2.0.0 |
|------|------|--------|
| Architecture center | Launcher-centric | Backend-centric |
| Content routing | SteamProvider in Launcher | Agent infrastructure (Backend-managed) |
| Server discovery | `servers.json` (local/bundled) | Registry API (`GET /servers`) |
| Download URL source | `ModEntry.downloadUrl` in Manifest | `JoinResponse.downloadPlan` from Backend |
| Manifest role | Desired State + Transport Layer | Desired State only |
| Steam fallback | Launcher executes SteamCMD | Backend last-resort, invisible to Launcher |

**RFC-0050 is the formal end of v1.x.**

---

## Launcher surface (frozen for v2.0.0)

The Launcher API surface is exactly:

```text
GetServers()       — server discovery via Backend registry
Join()             — POST /join/{serverId} → JoinResponse
Download()         — fetch blobs from Backend-issued URLs
Launch()           — start Project Zomboid with prepared profile
Settings()         — read/write local settings (backendUrl, gamePath, cache, profiles)
Diagnostics()      — copy diagnostics JSON
```

The Launcher has no knowledge of:
- Agent addresses or existence
- SteamCMD or Workshop IDs (beyond informational provenance in manifest data)
- Manifest generation
- Server topology
- Content origin (Agent / Backend storage / SteamCMD)

---

## Phase A — Backend Core  ✅ COMPLETE (2026-06-05)

All six milestones shipped. The system is now a trusted distributed content platform.

### A1. Backend Skeleton  ✅

`apps/backend/cmd/backend` — HTTP server, flags `-addr`, `-registry`.  
Endpoints: `GET /api/v1/health`, `GET /api/v1/servers`, `GET /api/v1/servers/{id}`.

### A2. Server Registry  ✅

Launcher migrated from `servers.json` to `GET /api/v1/servers`.  
`backendUrl` setting replaces `steamcmdPath`.  
`RegistryLauncherApi` + `UIService.GetServerList` call Backend.

### A3. Join API  ✅

`POST /api/v1/join/{serverId}` returns canonical `JoinResponse`:

```json
{
  "sessionId": "...",
  "manifest": { "serverId": "...", "version": "...", "mods": [...], "launchArgs": [], "profile": {} },
  "downloadPlan": [{ "modId": "...", "sha256": "...", "sizeBytes": 0, "url": "..." }],
  "issuedAt": "..."
}
```

`pipeline.RunJoinFromBackend` — Launcher no longer owns manifest resolution or provider selection.

### A4. Content Store (CAS)  ✅

`GET /api/v1/download/{sha256}` streams blobs.  
`internal/storage.DiskStore` — content-addressable layout `<root>/<sha256[:2]>/<sha256>`.  
SHA256 verified on write (`Put`), served with `Cache-Control: immutable`.

### A5. Agent Content Publisher  ✅

`apps/pz-agent` — scan mods dir → hash → PUT blob → PUT manifest → heartbeat.  
Backend endpoints: `PUT /api/v1/blobs/{sha256}`, `PUT /api/v1/manifests/{serverId}`.  
Live manifests override disk fixtures; join resolution uses Agent-pushed data immediately.

### A6. Agent Trust Boundary  ✅

`internal/auth.Store` — `Register(serverID) → token`, `Validate(token) → serverID`.  
`POST /api/v1/agents/register` — bootstrap (no token required).  
All Agent ingestion endpoints protected by `requireAgentToken` middleware.  
Agent: `-token` flag, `PZ_AGENT_TOKEN` env, auto-register fallback.  
`-no-auth` backend flag for dev/test mode.

### Phase A outcome

```text
Backend  = control plane + CAS storage + manifest authority + auth authority
Agent    = trusted, identity-bound content publisher node
Launcher = deterministic executor — no manifest knowledge, no provider logic
```

Trust model:
```text
Agent  → authenticated (token)    — write access to blobs/manifests
Launcher → unauthenticated        — read-only consumer of JoinResponse
Backend  → single source of truth — decides everything
```

---

## Phase B — Scale + Resilience + Operations

Phase A delivered the architecture. Phase B makes it production-grade.

### B1. Observability

Structured logging (JSON), trace IDs propagated through join flow.  
`GET /api/v1/metrics` — Prometheus-compatible counters (join count, blob hit/miss, agent count).  
Agent health visible in registry (`lastSeen`, `status` derived from heartbeat staleness).

### B2. Storage Evolution

Extract `storage.Store` to support pluggable backends without Launcher changes:

```text
DiskStore    — current (Phase A)
S3Store      — AWS S3 / Cloudflare R2
MinIOStore   — self-hosted object storage
```

Launcher unchanged — it only knows `GET /api/v1/download/{sha256}`.

### B3. Agent Reliability

Retry model with exponential backoff.  
Offline blob queue — Agent buffers push attempts when Backend unreachable.  
Partial sync — skip blobs already verified in store (idempotent by design via SHA256).

### B4. Manifest Versioning

Manifest diff — only changed mods trigger re-download on client.  
Version history stored by Backend; rollback to previous manifest version.  
Incremental update plan in JoinResponse (`action: "add" | "remove" | "update"` per mod).

---

## Phase C — v1.x Pipeline Removal

Safe to execute after Phase B is stable:

```text
Remove from Launcher:
  - libs/manifestv1     (manifest resolution owned by Backend)
  - libs/modplan        (mod planning owned by Backend)
  - libs/providers      (SteamProvider, ServerProvider)
  - libs/resolver       (content resolution owned by Backend)
  - pipeline.RunJoin    (replaced by RunJoinFromBackend)
  - fixtures/manifests  (Backend generates manifests from Agent data)
```

Each removal is safe once the Backend is the sole source of `JoinResponse.downloadPlan`.

---

## Out of scope (v2.0.0)

These are explicitly deferred. Do not design for them now.

```text
❌ Multi-game support
❌ Plugin system
❌ More event architecture RFCs
❌ SDK / public API
❌ Marketplace
❌ P2P content distribution
```

---

## Why the existing infrastructure fits v2.0.0 better than v1.x

The Event Runtime, Replay (RFC-0024), Snapshot (RFC-0025), and Domain RFCs (0030–0036) were designed around clean state machines and event flows. In v1.x, the SteamProvider and Manifest-as-transport model created out-of-band state that leaked into those layers.

In v2.0.0:
- The Launcher has a single outbound call (`POST /join`) that returns a complete download plan
- All subsequent steps are deterministic local operations (resolve → cache check → download → verify → install → launch)
- This maps cleanly onto the existing pipeline state machine and event model
- No provider routing decisions, no SteamCMD subprocess, no URL construction

The existing infrastructure is more compatible with v2.0.0 than with v1.x.
