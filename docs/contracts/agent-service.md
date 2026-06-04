# Contract: Agent Service

## Responsibility

The agent runs beside a dedicated Project Zomboid server and reports status, manifests, and availability to the launcher ecosystem.

## API surface

- `POST /heartbeat`
  - agent sends server status, player counts, and metadata
- `PUT /manifest`
  - agent publishes the current server manifest
- `GET /config` (optional)
  - agent returns runtime configuration and local status

## Data contracts

- Heartbeat payload includes `serverId`, `timestamp`, `status`, `playerCount`, `maxPlayers`, and optional metadata
- Manifest payload includes `serverId`, `version`, `gameVersion`, `mods`, `checksum`, and `createdAt`

## Invariants

- agent must only publish manifests for the local server
- heartbeats must be emitted at a regular interval to maintain availability
- manifests are immutable once published and versioned sequentially

## Integration points

- directory service consumes heartbeat events for public status
- manifest service stores published manifests and history
- provider system may use agent-published package metadata to seed registry entries
