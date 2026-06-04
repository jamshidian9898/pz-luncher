# Contract: Manifest Service

## Responsibility

The manifest service stores server manifests, manages manifest history, and supports rollback.

## API surface

- `GET /manifest/{serverId}`
  - returns the latest manifest for the server
- `GET /manifest/{serverId}/{version}`
  - returns the manifest for a specific version
- `POST /manifest/{serverId}`
  - creates or updates a manifest
- `POST /manifest/{serverId}/rollback`
  - requests rollback to a prior manifest version

## Data contracts

- Manifest includes `serverId`, `version`, `gameVersion`, `mods`, `checksum`, and `createdAt`
- Manifest mods include `id`, `version`, `sha256`, `downloadUrl`, and `dependencies`

## Invariants

- persisted manifests are immutable once created
- manifest versions are sequential and versioned per server
- rollback is implemented by selecting an older manifest version, not by mutating existing manifests
