# Vision

Build a modern ecosystem for Project Zomboid servers that provides:

- Server discovery
- One-click join
- Automatic mod management
- Versioned server packs
- Profile isolation
- Rollback support
- Smart caching
- Decentralized content distribution

This system must operate without modifying the game itself. The launcher is a layer above Project Zomboid that allows players to switch between servers without manually managing mods, saves, or configuration.

## Core goals

- Keep servers discoverable and easy to join
- Ensure every server can publish a manifest
- Keep player profiles isolated per server
- Enable asset deduplication and content integrity
- Allow rollback to previous manifest versions
- Support efficient updates with delta and cached downloads
