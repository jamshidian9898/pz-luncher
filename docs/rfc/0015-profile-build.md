# RFC 0015: Profile Build

## Problem

The launcher must materialize server manifests into isolated profile directories without corrupting shared cache content.

## Goals

- Build an isolated profile from resolved packages
- Reuse shared cache content safely
- Prepare mods, saves, cache, logs, and config for the profile
- Keep profile construction deterministic and auditable

## Flow

```text
Manifest
↓
Packages
↓
Cache
↓
Profile
↓
Symlink/Junction
↓
Launch
```

## Steps

1. `Manifest` defines required mods and package hashes
2. `Packages` are resolved and provider-selected
3. `Cache` is checked for local package blobs
4. `Profile` directory is prepared for the chosen server
5. `Symlink/Junction` or copy strategy links package content into the profile
6. `Launch` starts the game against the prepared profile

## Considerations

- use symlinks or hard links when supported to avoid duplicate disk usage
- preserve per-profile `mods`, `saves`, `cache`, `logs`, and `config`
- ensure profile directories can be cleaned or rolled back safely
