# RFC-0050: v2.0.0 Architecture Rebaseline

**Status**: Canonical — v2.0.0  
**Effective**: v2.0.0  
**Supersedes**: Architectural assumptions in RFC-0002, RFC-0005, RFC-0006, RFC-0008, RFC-0010, RFC-0013, RFC-0014, RFC-0030–0036 where they conflict with this document  
**Preserves**: v1.x history — all prior RFCs remain as historical record; this document is the forward-authoritative architecture

---

## Summary

This RFC establishes the **canonical architecture for v2.0.0**.

The central rule is:

> **The Launcher communicates exclusively with the Backend. The Backend is the single control plane.**

All responsibilities not listed under "Launcher" in this document belong to the Backend or its managed infrastructure (Agents, SteamCMD, storage).

---

## Problem (v1.x limitations)

v1.x documentation allowed the Launcher to:

- Talk directly to Agents for content or metadata
- Know Agent addresses and infrastructure topology
- Execute SteamCMD locally for Workshop content
- Embed Workshop IDs as first-class launcher concerns
- Know server filesystem layout through manifest fields
- Act as a provider of last resort for Steam content

This created tight coupling between client and infrastructure, leaked infrastructure details into the client, and made the system difficult to evolve independently.

---

## Goals

- Launcher has a single, stable API surface: the Backend
- Agents are invisible to the Launcher
- SteamCMD is invisible to the Launcher
- Backend owns all content acquisition decisions
- Launcher role is limited to: discover → download → install → launch
- All URLs the Launcher downloads from are issued by the Backend

---

## Component Map

```text
┌──────────────────────────────────────────────────────────────┐
│                         BACKEND                              │
│                                                              │
│  • Registry          • Manifest versioning                   │
│  • Authentication    • Join APIs                             │
│  • Download APIs     • Agent enrollment                      │
│  • Installation tokens  • Content acquisition                │
│                                                              │
│         ┌─────────────────────────────────┐                  │
│         │            AGENTS               │                  │
│         │  (Backend-managed infrastructure)│                  │
│         │  • Discover mods                │                  │
│         │  • Build manifests              │                  │
│         │  • Expose content               │                  │
│         │  • Send heartbeats              │                  │
│         │  • Provide metadata             │                  │
│         └─────────────────────────────────┘                  │
│                                                              │
│         ┌─────────────────────────────────┐                  │
│         │          SteamCMD               │                  │
│         │  (Backend last-resort mechanism) │                  │
│         │  Used only when:                │                  │
│         │    1. Client cache miss         │                  │
│         │    2. Backend storage miss      │                  │
│         │    3. Agent content unavailable │                  │
│         └─────────────────────────────────┘                  │
└──────────────────────────────────────────────────────────────┘
                              │
                   Backend API (single surface)
                              │
                              ▼
┌──────────────────────────────────────────────────────────────┐
│                        LAUNCHER                              │
│                                                              │
│  • Server discovery (via Backend registry)                   │
│  • Manifest fetch (from Backend)                             │
│  • Mod plan resolution (local, from manifest data)           │
│  • Download (URLs from Backend — never self-resolved)        │
│  • Verify (SHA256 of downloaded blobs)                       │
│  • Install (profile isolation)                               │
│  • Launch (Project Zomboid with prepared profile)            │
└──────────────────────────────────────────────────────────────┘
```

---

## Strict Boundaries

### Launcher MUST

- Communicate exclusively with Backend APIs
- Accept download URLs from Backend (not construct them)
- Verify all downloaded content by SHA256
- Maintain isolated per-server profiles locally
- Emit standard events for UI progress

### Launcher MUST NOT

- Contact Agents directly
- Know any Agent address or endpoint
- Execute SteamCMD or any Steam tooling
- Know Workshop IDs as actionable download targets
- Know server filesystem layout
- Construct content URLs independently
- Determine content origin (Agent vs Backend storage vs SteamCMD)

### Backend MUST

- Expose a registry API for server discovery
- Expose an authentication mechanism
- Version and serve manifests
- Issue download URLs for all required content
- Manage Agent enrollment and trust
- Manage one-time installation tokens
- Orchestrate content acquisition (from Agents, storage, or SteamCMD as needed)

### Agents MUST

- Report exclusively to Backend (heartbeats, manifests, metadata)
- Discover mods from server filesystem
- Build and submit manifests to Backend
- Expose content to Backend on request
- Never be contacted directly by the Launcher

### SteamCMD

- Is a Backend-exclusive tool
- Is invoked only as a last resort (client cache miss → Backend storage miss → Agent unavailable)
- Is never exposed to or invoked by the Launcher
- Workshop IDs in manifest data are informational provenance only; the Launcher never uses them to download

---

## Backend APIs (Launcher-facing surface, v2)

The Backend exposes the following API categories to the Launcher. Exact schema is specified in `schemas/openapi/`.

| Category | Example endpoints |
|----------|-------------------|
| Discovery | `GET /servers`, `GET /servers/{id}` |
| Manifest | `GET /servers/{id}/manifest`, `GET /manifests/{id}/{version}` |
| Join | `POST /join/{serverId}` → returns download plan |
| Download | `GET /download/{token}` → signed URL or blob |
| Auth | `POST /auth/token`, token refresh |
| Installation | `POST /install/token` → one-time token for new client |
| Health | `GET /health` |

The Launcher never calls any endpoint that is not part of this Backend API surface.

---

## Download Model (v2)

```text
Launcher                     Backend
   │                            │
   │  POST /join/{serverId}     │
   │ ─────────────────────────► │  ← Backend resolves manifest,
   │                            │    checks content availability,
   │                            │    orchestrates acquisition if needed
   │  JoinResponse              │
   │ ◄───────────────────────── │
   │  { downloadPlan: [         │
   │      { modId, sha256,      │
   │        url, sizeBytes }    │
   │    ] }                     │
   │                            │
   │  GET url (per mod)         │
   │ ─────────────────────────► │  ← Launcher downloads blobs
   │  blob                      │
   │ ◄───────────────────────── │
   │                            │
   │  SHA256 verify locally     │
   │  Install to profile        │
```

- The Launcher **never** decides where content comes from
- The Launcher **never** knows if a URL came from Agent, Backend storage, or SteamCMD
- The Backend issues all URLs with appropriate auth/expiry
- The Launcher only verifies SHA256 after download

---

## Agent Model (v2)

Agents are **Backend-managed infrastructure**. They are invisible to the Launcher.

```text
Agent                        Backend
  │                             │
  │  heartbeat (status,         │
  │  playerCount, serverId)     │
  │ ──────────────────────────► │
  │                             │
  │  PUT /manifest              │
  │  (built from server fs)     │
  │ ──────────────────────────► │
  │                             │
  │  content request            │
  │ ◄────────────────────────── │  ← Backend pulls content from Agent
  │  content blob               │     when needed for download plan
  │ ──────────────────────────► │
```

- Agent enrollment is managed by the Backend (tokens, trust)
- Agents never receive requests from the Launcher
- Agent addresses are Backend-internal

---

## SteamCMD Model (v2)

SteamCMD is a **Backend-internal tool** used as a last resort:

```text
Resolution order for content:
  1. Backend storage (content-addressable cache)
  2. Agent-provided content
  3. SteamCMD (last resort — triggered by Backend only)
```

The Launcher sees none of this. The Launcher receives a download URL and downloads the blob.

---

## Manifest Schema (v2 delta from RFC-0030)

The manifest format is preserved with the following change:

`ModEntry.workshopId` becomes **informational provenance only**:

- It is no longer an actionable download target for the Launcher
- It is stored for human reference and Backend content tracing
- The Launcher MUST NOT use `workshopId` to construct download URLs
- Download URLs are always provided by the Backend in the `JoinResponse.downloadPlan`

```ts
export interface ModEntry {
  id: string;
  name: string;
  version: string;
  sha256: string;
  sizeBytes?: number;

  workshopId?: string;       // INFORMATIONAL ONLY — not used by Launcher for download
  // downloadUrl removed — URLs come from Backend JoinResponse, not manifest

  dependencies: string[];
  optional?: boolean;
}
```

---

## Provider System (v2 delta from RFC-0008, RFC-0014)

The Launcher provider stack is simplified:

| Priority | Provider | Description |
|----------|----------|-------------|
| 1 | `LocalCacheProvider` | Already-downloaded content, SHA256-keyed |
| 2 | `BackendProvider` | URLs issued by Backend JoinResponse |

`ServerProvider` and `SteamProvider` are **removed from the Launcher**.  
They exist in the Backend's internal resolution stack, not in the Launcher.

---

## Settings (v2 delta from RFC-0036)

`steamcmdPath` is **removed** from Launcher settings. SteamCMD is a Backend concern.

```json
{
  "gamePath": "/path/to/ProjectZomboid",
  "cachePath": "/path/to/cache",
  "profilesPath": "/path/to/profiles",
  "backendUrl": "https://api.pzlauncher.example.com",
  "concurrentDownloads": 3,
  "bandwidthLimitMbps": 0,
  "verifyChecksum": true
}
```

New field: `backendUrl` — the single Backend API base URL.

---

## Migration from v1.x

| v1.x behavior | v2.0.0 replacement |
|---------------|-------------------|
| Launcher calls Agent directly | Launcher calls Backend only |
| Launcher knows Agent addresses | Agent addresses are Backend-internal |
| Launcher executes SteamCMD | Backend executes SteamCMD as last resort |
| `ModEntry.downloadUrl` in manifest | Backend JoinResponse issues download URLs |
| `ModEntry.workshopId` used for download | `workshopId` is informational provenance only |
| `SteamProvider` in Launcher provider stack | Removed; Backend resolves Steam content |
| `ServerProvider` in Launcher provider stack | Removed; Backend issues all URLs |
| `steamcmdPath` in Launcher settings | Removed |

---

## Non-Goals (v2.0.0)

- Defining Backend internal architecture (microservices vs monolith)
- Defining Agent-to-Agent communication
- Defining Backend SteamCMD orchestration implementation
- UI changes beyond adapter contract updates
- P2P content distribution

---

## Invariants (canonical)

1. **Launcher → Backend only.** No other outbound API call from Launcher.
2. **Agents are opaque.** Launcher has no Agent concept in its codebase.
3. **SteamCMD is absent from Launcher.** No SteamCMD binary, path, or invocation.
4. **All download URLs are Backend-issued.** Launcher never constructs content URLs.
5. **SHA256 verification is Launcher responsibility.** Backend supplies hashes; Launcher verifies.
6. **Profile isolation is Launcher responsibility.** Per-server isolated profiles on client disk.

---

## Affected RFCs (v1.x — historical, not modified)

These RFCs are preserved as v1.x history. This RFC supersedes their architectural assumptions where they conflict:

| RFC | Topic | v2 delta |
|-----|-------|----------|
| RFC-0002 | Service Boundaries | Launcher→Agent coupling removed |
| RFC-0005 | Agent Lifecycle | Agent reports to Backend only |
| RFC-0006 | Launcher Core | Launcher talks Backend APIs only |
| RFC-0008 | Provider System | Launcher providers: LocalCache + Backend only |
| RFC-0010 | Steam Provider | Moved entirely to Backend |
| RFC-0013 | Manifest Resolution | Manifest + URLs from Backend |
| RFC-0014 | Provider Priority | SteamProvider + ServerProvider removed from Launcher |
| RFC-0030 | Server Manifest v1 | `downloadUrl` removed; `workshopId` informational only |
| RFC-0031 | Mod Dependency Resolver | Unchanged (local resolution logic unaffected) |
| RFC-0032 | Download Manager | URLs from Backend JoinResponse only |
| RFC-0033 | Installation Pipeline | Backend issues JoinResponse before download |
| RFC-0034 | Profile System | `steamcmdPath` removed from settings |
| RFC-0036 | Launcher Settings | `steamcmdPath` removed; `backendUrl` added |
