# RFC 0007: Profile Isolation

## Problem

Project Zomboid servers often require different mod sets and save states, so a shared game directory can cause conflicts.

## Goals

- Isolate saves, mods, and cache files per server profile
- Keep shared downloads deduplicated, while preserving profile isolation
- Allow users to switch servers without manual cleanup
- Preserve rollback and replayability for each server profile

## Design

### Profile directory layout

```text
profiles/
  {serverId}/
    mods/
    saves/
    cache/
    config/
```

### Isolation rules

- Each server profile has its own `mods` and `saves`
- Shared content is stored by hash in a global cache
- Launcher core populates profile-specific folders from the cache
- Profiles are versioned by manifest and can be rolled back independently

### Profile lifecycle

- Create profile on first join
- Sync required mods and files before launching
- Preserve previous manifest versions for rollback
- Clean up unused profile data based on retention policies

## Non-Goals

- Storing user-wide saves or mods across unrelated servers
- Synchronizing profile state across different machines

## Open Questions

- Should profiles support multiple variants for the same server?
- How should the launcher expose profile cleanup and retention options?
- Should cache entries be garbage-collected automatically when no profiles reference them?
