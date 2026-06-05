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

## Phase A — Backend Core

The first real things to build. Everything else depends on this.

### A1. Registry API

```http
GET /servers
GET /servers/{id}
```

Returns server list with metadata. Backed by heartbeat data from Agents.

### A2. Join API

```http
POST /join/{serverId}
```

Response:

```json
{
  "sessionId": "...",
  "manifest": {
    "serverId": "...",
    "version": "...",
    "gameVersion": "...",
    "mods": [
      { "id": "...", "name": "...", "version": "...", "sha256": "...", "dependencies": [] }
    ],
    "launchArgs": [],
    "profile": {}
  },
  "downloadPlan": [
    { "modId": "...", "sha256": "...", "sizeBytes": 0, "url": "..." }
  ]
}
```

This single response replaces the v1.x manifest-fetch + provider-selection chain.

### A3. Download API

```http
GET /mods/{sha256}
GET /download/{token}
```

Signed or direct URLs. Launcher downloads blob, verifies SHA256, installs.

### A4. Agent Enrollment

```http
POST /agents/register   → enrollment token
POST /agents/heartbeat  → server status, player count
PUT  /agents/manifest   → submit manifest to Backend
```

Backend-internal. Launcher never calls these.

### A5. One-Time Installation Token

```http
POST /tokens
```

Issued to new Launcher clients for first-run authentication.

---

## Phase B — Agent (minimal)

The Agent is not a server manager. It is a content and metadata publisher.

Required capabilities only:

```text
discover mods       — scan server mod folders
build manifest      — generate ModEntry[] from discovered mods
serve content       — expose mod blobs to Backend on request
heartbeat           — report server status at regular interval
```

Not in scope for Phase B:
- Server start/stop management
- Player management
- Configuration editing
- Any UI

---

## Phase C — v1.x Pipeline Removal

Gradual cleanup after Phase A and B are stable:

```text
Remove from Launcher:
  - SteamProvider
  - ServerProvider
  - servers.json bundled fixture
  - ModEntry.downloadUrl handling
  - steamcmdPath from settings
  - libs/steam
  - libs/providers (Steam/Server implementations)
```

Each removal is safe once the Backend issues all download URLs via JoinResponse.

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
