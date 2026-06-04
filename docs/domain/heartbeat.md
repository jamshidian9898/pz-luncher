# Domain: Heartbeat

## Entity: Heartbeat

Fields
- `id` (string): unique heartbeat event identifier
- `serverId` (string): server associated with the heartbeat
- `timestamp` (timestamp): event time
- `status` (string): `online`, `offline`, `maintenance`
- `playerCount` (int)
- `maxPlayers` (int)
- `metadata` (map[string]string]): optional status details

Rules
- heartbeat events are append-only records
- `serverId` must reference an existing server
- status values are constrained to allowed states
- `playerCount` must be non-negative and no greater than `maxPlayers`

State transitions
- each heartbeat reflects the current status snapshot
- status may change between online, offline, and maintenance
- missing heartbeats beyond a threshold imply offline state

Relations
- Heartbeats belong to a server
- The directory service and agent consume heartbeat data for availability
- Heartbeats may be summarized to derive current server status
