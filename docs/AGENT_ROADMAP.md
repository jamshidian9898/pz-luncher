# Agent Roadmap — v2.0.0 Phase B

**Status**: Phase B (not started)  
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
| Build `ModEntry[]` from discovered mods | ⬜ Pending | Depends on RFC-0053 backend |
| `PUT /api/v1/agents/manifest` call | ⬜ Pending | — |
| Retry on transient errors | 🟡 In progress | `apps/pz-agent/internal/ingest/` |
| Re-submit on fs change | ⬜ Pending | — |

### Content serving

| Task | Status | Notes |
|------|--------|-------|
| HTTP server for blob requests | ⬜ Pending | Backend calls `GET /content/{sha256}` on agent |
| Stream mod archive on request | ⬜ Pending | — |

### Heartbeat

| Task | Status | Notes |
|------|--------|-------|
| `POST /api/v1/agents/heartbeat` every 30s | ⬜ Pending | Depends on RFC-0053 backend |
| Reconnect on backend unavailable | ⬜ Pending | Exponential backoff |

---

## RFC-0053: Agent Enrollment (agent side)

| Task | Status | Notes |
|------|--------|-------|
| Read enrollment token from `agent.json` / env | ⬜ Pending | Single-use, 24h TTL |
| `POST /api/v1/agents/register` on startup | ⬜ Pending | Returns agent access token |
| Persist agent token securely | ⬜ Pending | `agent-token` file, mode 0600 |
| Re-enroll if token revoked (401 response) | ⬜ Pending | — |

---

## Phase B exit criteria

- [ ] Agent enrolls with a one-time token and receives agent access token
- [ ] Agent sends heartbeat every 30s; Backend marks it Healthy
- [ ] Agent scans mods and submits manifest to Backend
- [ ] Backend can pull a mod blob from Agent via content serving endpoint
- [ ] A Launcher `POST /join` results in valid download URLs served by Agent content

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
