# RFC-0056: Agent Minimal (v2.0.0)

**Status**: Active — v2.0.0 Phase B  
**Depends on**: [RFC-0050](0050-v2-architecture-rebaseline.md), [RFC-0053](0053-agent-enrollment.md)  
**Implements**: Agent responsibilities from RFC-0050

---

## Purpose

Define the minimal Agent that satisfies v2.0.0 Phase B requirements. This RFC is intentionally narrow. The Agent is not a server manager. It is a content and metadata publisher that reports to the Backend.

---

## Capabilities (required, v2.0.0)

```text
1. Config discovery     — read server config to determine serverId, gameVersion
2. Mod discovery        — scan server mod folders, extract id/version/sha256
3. Manifest generation  — build ModEntry[] and submit to Backend
4. Content serving      — expose mod blobs to Backend on request
5. Heartbeat            — report server status at regular interval
```

These five capabilities are the complete scope of Phase B.

---

## Out of scope (explicitly deferred)

```text
❌ Server start/stop management
❌ RCON integration
❌ Player management or kick/ban
❌ Metrics collection or Prometheus export
❌ Multi-game support
❌ Auto-update of Agent binary
❌ Agent UI or web dashboard
❌ Log streaming
❌ Backup management
```

---

## Config discovery

On startup, the Agent reads:

```text
<serverRoot>/Server/<serverName>.ini    — server configuration
<serverRoot>/mods/                      — mod folder
```

From configuration it extracts:
- `serverId` (or derives from server name)
- `gameVersion` (from game binary version or config)
- `maxPlayers`

The Agent is configured with:
- `serverRoot` — path to the dedicated server installation
- `backendUrl` — Backend API base URL
- `serverId` — explicit override (if not auto-detected)

Configuration file: `agent.json` or environment variables.

---

## Mod discovery

The Agent scans `<serverRoot>/mods/` for installed mods.

For each mod directory, it:
1. Reads mod metadata (id, name, version from mod info file)
2. Computes SHA256 of the mod archive or directory content
3. Records `sizeBytes`
4. Extracts `workshopId` if present in mod metadata

Mod discovery runs:
- On Agent startup
- On filesystem change event (inotify/FSEvents/ReadDirectoryChanges)
- On explicit trigger from Backend (future)

---

## Manifest generation

After mod discovery, the Agent builds a `ServerManifest` (RFC-0030 schema) and submits it to the Backend via `PUT /api/v1/agents/manifest`.

The Agent only submits a new manifest if the mod set has changed (SHA256 of manifest content changed).

---

## Content serving

The Backend may request mod blobs from the Agent when:
- A Launcher calls `POST /join` and the Backend does not have the blob cached
- The Backend needs to populate its content store

The Agent exposes a Backend-internal endpoint (not in the public API, not reachable by Launchers):

```http
GET /agent/content/{sha256}
```

Returns the raw blob for the given SHA256. The Backend authenticates this request with the agent's access token.

The Agent only serves blobs it has discovered locally. It never fetches from external sources.

---

## Heartbeat

The Agent sends a heartbeat to `POST /api/v1/agents/heartbeat` every 30 seconds (configurable).

Heartbeat payload includes:
- `serverId`
- `status`: `online` | `offline` | `starting` | `stopping`
- `playerCount` (read from server log or RCON if available; 0 if unknown)
- `maxPlayers`
- `gameVersion`
- `timestamp`

The Agent reads `playerCount` from server log file tail if RCON is not configured. If neither is available, it reports `playerCount: 0` — this is acceptable in Phase B.

---

## Binary

- Written in Go (consistent with the rest of the monorepo)
- Single static binary, no runtime dependencies
- Configurable via `agent.json` and environment variable overrides
- Runs as a systemd service or background process alongside the PZ dedicated server

---

## Startup sequence

```text
1. Load config (agent.json + env)
2. POST /api/v1/agents/register (if not already registered)
3. Start heartbeat ticker (30s)
4. Run mod discovery (initial scan)
5. Submit manifest if mods found
6. Start filesystem watcher for mod changes
7. Start content server (Backend-internal)
8. Block on ticker + watcher events
```

---

## Error handling

- If Backend is unreachable at startup: retry with exponential backoff, continue local operations
- If manifest submission fails: retry 3× with backoff; log error; continue heartbeat
- If content request fails to serve: log error, return 404 — Backend falls back to SteamCMD

---

## Invariants

1. Agent never contacts the Launcher
2. Agent never constructs download URLs for Launchers
3. Agent never executes SteamCMD
4. Agent only serves content it has on local disk
5. Agent token is scoped to one `serverId`
