# Vision

**v2.0.0** — see [RFC-0050](rfc/0050-v2-architecture-rebaseline.md) for canonical architecture.

Build a modern ecosystem for Project Zomboid servers that provides:

- Server discovery
- One-click join
- Automatic mod management
- Versioned server packs
- Profile isolation
- Rollback support
- Smart caching
- Backend-managed content distribution

This system must operate without modifying the game itself. The launcher is a layer above Project Zomboid that allows players to switch between servers without manually managing mods, saves, or configuration.

## Core goals

- Keep servers discoverable and easy to join
- Backend is the single control plane: registry, auth, manifests, download URLs, agent enrollment
- Agents are Backend-managed infrastructure: mod discovery, manifest building, content exposure
- Keep player profiles isolated per server
- Enable asset deduplication and content integrity (SHA256)
- Allow rollback to previous manifest versions
- Launcher communicates exclusively with the Backend — no direct Agent or SteamCMD access
- SteamCMD is a Backend last-resort mechanism, invisible to the Launcher
