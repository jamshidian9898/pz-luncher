# Architecture

**v2.0.0 canonical** — see [RFC-0050](rfc/0050-v2-architecture-rebaseline.md)

The project is organized as a monorepo with separate apps and shared libraries.

## Architecture layers (v2.0.0)

```text
┌─────────────────────────────────────────┐
│               BACKEND                   │
│  (single control plane)                 │
│  Registry · Auth · Manifests            │
│  Join APIs · Download APIs              │
│  Agent enrollment · Installation tokens │
│  Content acquisition                    │
│       ┌──────────┐  ┌──────────┐        │
│       │  Agents  │  │ SteamCMD │        │
│       │(managed) │  │(last res)│        │
│       └──────────┘  └──────────┘        │
└───────────────────┬─────────────────────┘
                    │ Backend API (only)
                    ↓
┌─────────────────────────────────────────┐
│               LAUNCHER                  │
│  Discover · Fetch manifest              │
│  Download (URLs from Backend)           │
│  Verify · Install · Profile isolation   │
│  Launch Project Zomboid                 │
└─────────────────────────────────────────┘
```

**Rule**: Launcher communicates exclusively with Backend APIs. Agents and SteamCMD are Backend-internal infrastructure and are invisible to the Launcher.

## Component overview

### Backend (server-side)
- `apps/backend`: single control plane — registry, auth, manifest versioning, join APIs, download APIs, agent enrollment, installation tokens, content acquisition
- `apps/pz-agent`: server-side agent (Backend-managed) — discovers mods, builds manifests, exposes content, sends heartbeats to Backend

### Launcher (client-side)
- `apps/launcher-core`: join orchestration, dependency resolution, profile preparation, game launch
- `apps/launcher-ui`: user-facing interface for server discovery and join flow

## Shared libraries

- `libs/manifest`: manifest format, validation, and history helpers
- `libs/package`: package metadata and content manifest utilities
- `libs/profile`: profile isolation rules and layout helpers
- `libs/hashing`: hashing utilities for content-addressable storage
- `libs/downloader`: chunked downloads, resume, and integrity helpers
- `libs/telemetry`: metrics, logging, and monitoring support
- `libs/logger`: structured logger wrappers
- `libs/contracts`: API contracts and DTO definitions

## v1.x history

v1.x architecture (Launcher → Agent direct communication, Launcher-side SteamCMD, per-provider priority stack) is preserved in Foundation RFCs 0001–0036 as historical record.
