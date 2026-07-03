# Agent Roadmap — v2.0.0 Phase B

**Status**: Phase B core implemented (push-based); E2E validated 2026-07-03  
**Depends on**: Phase A backend complete — see [BACKEND_ROADMAP.md](BACKEND_ROADMAP.md)  
**RFCs**: [0053](rfc/0053-agent-enrollment.md) · [0056](rfc/0056-agent-minimal.md)  
**Code**: `apps/pz-agent/`

---

## Phase B scope

The Agent's sole job in v2.0.0:

```
1. Config discovery     — read serverId, gameVersion from server config
2. Mod discovery        — scan mods/, compute SHA256 per mod
3. Manifest generation  — build ModEntry[] and POST to Backend
4. Content serving      — serve mod blobs to Backend on request
5. Heartbeat            — report liveness every 30 seconds
```

Everything else is explicitly out of scope for this phase.

---

## RFC-0056: Agent Minimal

### Config discovery

| Task | Status | File |
|------|--------|-------|
| Read `agent.json` / env vars | ✅ Done | `apps/pz-agent/agent.go` |
| Parse `<serverRoot>/Server/<name>.ini` | 🟡 In progress | `apps/pz-agent/internal/discover/` |
| Extract `serverId`, `gameVersion`, `maxPlayers` | 🟡 In progress | — |

### Mod discovery

| Task | Status | File |
|------|--------|-------|
| Scan `<serverRoot>/mods/` | ✅ Done | `apps/pz-agent/internal/discover/discover.go` |
| Compute SHA256 per mod | ✅ Done | — |
| Extract `workshopId` from mod metadata | 🟡 In progress | — |
| Filesystem watch (inotify/FSEvents/ReadDirectoryChanges) | ⬜ Pending | — |

### Manifest generation + submission

| Task | Status | Notes |
|------|--------|-------|
| Build `ModEntry[]` from discovered mods | ✅ Done | `apps/pz-agent/internal/ingest/ingest.go` |
| `PUT /api/v1/manifests/{serverId}` call | ✅ Done | Versioned store on backend (B4) |
| Retry on transient errors | ✅ Done | `apps/pz-agent/internal/retry/` |
| Publish only on content change | ✅ Done | Content-hash diff in `cmd/agent/main.go` |
| Re-submit on fs change (watch) | ⬜ Pending | Poll-based via `-interval` for now |

### Content publishing

Design change vs original RFC-0053: the Agent **pushes** blobs to the Backend
(`PUT /api/v1/blobs/{sha256}`) instead of serving them for Backend pull.
Directory mods are identified by the SHA256 of their **deterministic zip
archive** (`libs/ziputil`), so the uploaded bytes always match the declared hash
and the Launcher extracts them back into a real mod folder.

| Task | Status | Notes |
|------|--------|-------|
| Push blobs (HEAD-then-PUT, idempotent) | ✅ Done | `ingest.PushBlob` |
| Deterministic zip identity for directory mods | ✅ Done | `libs/ziputil` |

### Heartbeat

| Task | Status | Notes |
|------|--------|-------|
| `POST /api/v1/agents/heartbeat` | ✅ Done | Interval via `-interval` flag |
| Reconnect on backend unavailable | ✅ Done | Exponential backoff (`internal/retry`) |

---

## RFC-0053: Agent Enrollment (agent side)

| Task | Status | Notes |
|------|--------|-------|
| `POST /api/v1/agents/register` on startup | ✅ Done | Open registration; returns access token |
| Token via `-token` flag / `PZ_AGENT_TOKEN` env | ✅ Done | `cmd/agent/main.go` |
| One-time enrollment tokens (single-use, 24h TTL) | ⬜ Pending | Registration is open for MVP |
| Persist agent token securely | ⬜ Pending | Token held in memory per run |
| Re-enroll if token revoked (401 response) | ⬜ Pending | — |

---

## Phase B exit criteria

- [x] Agent registers and receives agent access token *(open registration for MVP)*
- [x] Agent sends heartbeats; Backend marks it online/degraded/offline
- [x] Agent scans mods and submits manifest to Backend
- [x] Agent pushes mod blobs to Backend content store *(push replaces pull design)*
- [x] A Launcher `POST /join` downloads agent-published content and builds a
      byte-identical profile — validated E2E 2026-07-03 (`join-cli -backend`)

---

## Installation

The Agent is installed on the game server host via:

```bash
curl https://<backend>/install.sh | bash
```

Install script: `apps/backend/cmd/backend/install.sh` (served when `-deploy` flag set).

---

## Not in scope (post-MVP)

- Server start/stop or RCON
- Player management
- Metrics export (Prometheus)
- Auto-update of Agent binary
- Multi-game support
- Log streaming or backup management
