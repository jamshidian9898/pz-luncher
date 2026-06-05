# RFC 0006: Launcher Core

> **v1.x historical record.** For v2.0.0 see [RFC-0050](0050-v2-architecture-rebaseline.md). The key v2 delta: Launcher communicates exclusively with the Backend API. It does not contact manifest-service, registry-service, or agents directly.

## Problem

The launcher requires a central orchestration layer to convert a manifest into a runnable isolated profile and start Project Zomboid.

## Goals

- Resolve required mods and package locations from manifests
- Reuse shared cache content when available
- Prepare isolated profile directories for each server
- Launch Project Zomboid with correct mod installation

## Design

### Join flow (v2.0.0)

1. User selects server in launcher UI
2. Launcher calls `POST /join/{serverId}` on the **Backend**
3. Backend returns a download plan (mod list with Backend-issued URLs and SHA256 hashes)
4. Launcher resolves mod dependencies from the plan (local, deterministic)
5. Check shared and profile caches for required content (by SHA256)
6. Download missing packages from Backend-issued URLs
7. Verify SHA256 of each downloaded blob
8. Build or sync the server-specific profile layout
9. Launch Project Zomboid with the profile's mods and config

### Profile preparation

- Each server gets a dedicated profile folder
- Mods, saves, and cache files are isolated per server
- Shared downloaded blobs may be hard-linked or copied into profile locations

### Failure handling

- Validate manifest compatibility before starting downloads
- Report missing or broken packages clearly to the user
- Roll back to a previous manifest if the current one fails

## Non-Goals

- Implementing UI details for server browsing
- Replacing the native game launch command entirely

## Open Questions

- How should launcher handle server packs with optional mods?
- Should profiles support multiple manifests per server, e.g., variants or presets?
- How should the launcher expose progress and retry semantics to the user?
