# RFC 0006: Launcher Core

## Problem

The launcher requires a central orchestration layer to convert a manifest into a runnable isolated profile and start Project Zomboid.

## Goals

- Resolve required mods and package locations from manifests
- Reuse shared cache content when available
- Prepare isolated profile directories for each server
- Launch Project Zomboid with correct mod installation

## Design

### Join flow

1. User selects server in launcher UI
2. Launcher fetches latest manifest from `manifest-service`
3. Resolve mod dependencies and package hashes
4. Check shared and profile caches for required content
5. Download missing packages via `download-service`
6. Build or sync the server-specific profile layout
7. Launch Project Zomboid with the profile's mods and config

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
