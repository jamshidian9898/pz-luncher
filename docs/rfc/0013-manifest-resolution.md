# RFC 0013: Manifest Resolution

> **v1.x historical record.** For v2.0.0 see [RFC-0050](0050-v2-architecture-rebaseline.md). The key v2 delta: the manifest and download plan are both returned by the Backend `POST /join/{serverId}` response. The Launcher no longer constructs download URLs or selects providers beyond LocalCache.

## Problem

Launcher Core must resolve a server manifest into a concrete set of packages and providers before download.

## Goals

- Define the end-to-end manifest resolution flow
- Resolve package references and dependencies deterministically
- Select providers in priority order
- Keep the manifest resolution process stable and explicit

## Flow

```text
Server
↓
Manifest
↓
Package Resolution
↓
Dependency Resolution
↓
Provider Selection
↓
Download
```

## Steps (v1.x)

1. `Server` is selected by the user or loaded from a local descriptor
2. `Manifest` is fetched from a manifest URL or local file
3. `Package Resolution` converts manifest mods into package records
4. `Dependency Resolution` expands transitive package requirements and validates acyclicity
5. `Provider Selection` chooses the best provider for each package
6. `Download` retrieves missing package content for the profile

## Steps (v2.0.0)

1. `Server` is selected by the user
2. Launcher calls `POST /join/{serverId}` on the Backend
3. Backend returns `JoinResponse` with manifest metadata + download plan (mod list, SHA256s, Backend-issued URLs)
4. `Dependency Resolution` runs locally on the manifest mod list (acyclicity, load order)
5. `Cache Check` skips already-verified local blobs (by SHA256)
6. `Download` fetches missing blobs from Backend-issued URLs
7. `Verify` SHA256 of each downloaded blob

## Invariants

- manifest contents must be validated before package resolution
- dependency graphs must be acyclic
- provider selection must follow configured priority and fallback rules
- downloads are only attempted for unresolved or uncached packages
