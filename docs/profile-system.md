# Profile Isolation

The launcher must isolate each server session so mod sets, saves, and cache files do not conflict.

## Directory layout

```text
profiles/
  serverA/
    mods/
    saves/
    cache/
  serverB/
    mods/
    saves/
    cache/
```

## Principles

- Every server gets its own profile directory
- No shared save or mod state across profiles unless explicitly cached
- Use content-addressable cache for shared downloads
- Support profile-specific configuration and load order

## Benefits

- Prevents mod conflicts between servers
- Enables easy rollback per server
- Keeps saves isolated for each gameplay experience
- Makes it possible to switch servers without manual cleanup
