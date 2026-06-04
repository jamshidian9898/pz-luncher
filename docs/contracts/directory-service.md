# Contract: Directory Service

## Responsibility

The directory service provides server discovery, search, filtering, tags, and heartbeat-driven availability.

## API surface

- `GET /servers`
  - returns public server listings and metadata
- `GET /servers/{id}`
  - returns details for a single server
- `POST /heartbeat`
  - receives heartbeat updates from servers or agents

## Data contracts

- Server metadata contains `id`, `name`, `region`, `tags`, `playerCount`, `maxPlayers`, `status`, `manifestId`, and `lastHeartbeat`
- Heartbeat payload includes server status and optional metadata

## Invariants

- directory server metadata is authoritative for discovery only
- the directory does not store package blobs or mod content
- heartbeat freshness determines online status
