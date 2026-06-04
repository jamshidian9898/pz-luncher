# Contract: Agent

## Responsibility

The agent monitors a dedicated Project Zomboid server, generates manifests, and syncs status with the launcher ecosystem.

## Interface

- `Heartbeat()` — send periodic status, player count, and health information
- `GenerateManifest()` — inspect the local server mod state and construct a manifest
- `DetectMods()` — identify local mods, versions, and dependencies
- `Sync()` — synchronize local state with the manifest and heartbeat endpoints

## Invariants

- the agent must only publish manifests for the local game server
- heartbeat and manifest operations should be idempotent and retryable
- `DetectMods()` must preserve mod dependency invariants and versions

## Integration

- `Heartbeat()` is consumed by directory or status services
- `GenerateManifest()` writes or publishes manifest data for launcher consumption
- `Sync()` may update local cache and package metadata based on service responses
