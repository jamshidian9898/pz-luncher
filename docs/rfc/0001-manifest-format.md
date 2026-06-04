# RFC 0001: Manifest Format

## Problem

Servers need a standard, versioned manifest format that defines required mods, checksums, and download locations.

## Goals

- Provide deterministic mod resolution
- Enable manifest history and rollback
- Support multiple providers and download URLs
- Validate content integrity before launch

## Schema

```json
{
  "gameVersion": "string",
  "manifestVersion": "integer",
  "mods": [
    {
      "id": "string",
      "version": "string",
      "sha256": "string",
      "downloadUrl": "string",
      "dependencies": ["string"]
    }
  ]
}
```

## Non-Goals

- Describing full server configuration for the game
- Replacing Project Zomboid's native mod loader

## Open Questions

- Should manifests include explicit provider metadata?
- How should delta updates be expressed in the manifest?
- Should the manifest support optional mod groups or packs?
