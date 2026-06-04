# RFC 0017: Save Isolation

## Problem

Project Zomboid profiles must keep mods, saves, cache, logs, and config isolated per server to prevent conflicts.

## Goals

- Separate profile directories for runtime artifacts
- Keep shared game state out of server-specific profiles
- Make cleanup and rollback easy and safe

## Isolation boundaries

Each profile should contain:

- `mods`
- `saves`
- `cache`
- `logs`
- `config`

## Profile structure

```text
profiles/
  {serverId}/
    mods/
    saves/
    cache/
    logs/
    config/
```

## Invariants

- profile runtime data must not leak between servers
- shared cache content may be referenced, but profile folders remain distinct
- logs and config are profile-scoped and can be inspected independently
