# Domain: Server

## Entity: Server

Fields
- `id` (string): unique server identifier
- `name` (string): display name
- `description` (string): public description
- `region` (string): region or locality
- `tags` ([]string): server tags and metadata
- `manifestId` (string): current manifest reference
- `status` (string): `online`, `offline`, `maintenance`
- `playerCount` (int)
- `maxPlayers` (int)
- `lastHeartbeat` (timestamp)

Rules
- `id` is immutable once created
- `name` is required and should be stable
- `region` should be one of predefined regions or empty for global
- `status` may only transition between valid states

State transitions
- offline → online when heartbeat and manifest are valid
- online → maintenance or offline when heartbeat fails or the server is unavailable
- maintenance → online after manual re-enable

Relations
- One server has one active manifest reference
- A server may have many heartbeat events over time
- A server may appear in directory search results and can be favorited by users
