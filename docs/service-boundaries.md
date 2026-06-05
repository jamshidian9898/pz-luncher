# Service Boundaries

**v2.0.0 canonical** — see [RFC-0050](rfc/0050-v2-architecture-rebaseline.md)

The Backend is the **single control plane**. The Launcher communicates exclusively with the Backend API. Agents and SteamCMD are Backend-internal infrastructure invisible to the Launcher.

## Backend

- Single control plane for all server-side operations
- Owns: registry, authentication, manifest versioning, join APIs, download APIs, agent enrollment, one-time installation tokens, content acquisition
- Issues all download URLs to the Launcher
- Manages Agent trust and enrollment
- Invokes SteamCMD as a last resort (client cache miss → Backend storage miss → Agent unavailable)

## Agent (Backend-managed infrastructure)

- Runs beside a dedicated Project Zomboid server
- Reports exclusively to the Backend (heartbeats, manifests, metadata)
- Discovers mods from server filesystem and builds manifests
- Exposes content to Backend on request
- Is never contacted by the Launcher
- Agent addresses are Backend-internal

## SteamCMD (Backend-exclusive tool)

- Used only by the Backend, never by the Launcher
- Invoked as last resort when: client cache misses, Backend storage misses, Agent content unavailable
- Workshop IDs in manifest data are informational provenance only

## Launcher

- Communicates exclusively with the Backend API
- Responsibilities: server discovery, manifest fetch, mod plan resolution (local), download (URLs from Backend), SHA256 verify, profile installation, game launch
- Never contacts Agents directly
- Never executes SteamCMD
- Never constructs content URLs independently
