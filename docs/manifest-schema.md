# Manifest Schema

A server manifest describes the game version, required mods, and package metadata needed to join a server.

## Example manifest

```json
{
  "gameVersion": "42.8",
  "manifestVersion": 91,
  "mods": [
    {
      "id": "example-mod",
      "version": "1.2.3",
      "sha256": "...",
      "downloadUrl": "https://...",
      "dependencies": ["other-mod"]
    }
  ]
}
```

## Fields

- `gameVersion`: Project Zomboid build version
- `manifestVersion`: numeric manifest revision
- `mods`: list of required mods and dependencies
- `sha256`: content hash for integrity verification
- `downloadUrl`: location to retrieve the package
- `dependencies`: optional dependency graph for mod ordering

## Goals

- Enable deterministic mod resolution
- Support manifest history and rollback
- Support multiple package providers and download locations
- Allow local and remote package validation
