# RFC 0013: Manifest Resolution

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

## Steps

1. `Server` is selected by the user or loaded from a local descriptor
2. `Manifest` is fetched from a manifest URL or local file
3. `Package Resolution` converts manifest mods into package records
4. `Dependency Resolution` expands transitive package requirements and validates acyclicity
5. `Provider Selection` chooses the best provider for each package
6. `Download` retrieves missing package content for the profile

## Invariants

- manifest contents must be validated before package resolution
- dependency graphs must be acyclic
- provider selection must follow configured priority and fallback rules
- downloads are only attempted for unresolved or uncached packages
