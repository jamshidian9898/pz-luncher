# Domain: Profile

## Entity: Profile

Fields
- `id` (string): profile identifier, typically based on server id or server pack
- `serverId` (string): associated server
- `manifestVersion` (int): manifest version currently provisioned
- `path` (string): filesystem location of the profile
- `createdAt` (timestamp)
- `updatedAt` (timestamp)
- `status` (string): `ready`, `syncing`, `error`

Rules
- each profile belongs to exactly one server
- a profile may be updated when the server manifest changes
- `path` must remain stable for the profile lifetime
- a profile may be created only when the launcher has a valid manifest

State transitions
- new profile created on first join
- profile sync begins when downloads are required
- profile becomes ready after package and mod resolution completes
- profile becomes error if required packages fail

Relations
- Profiles reference a specific manifest version
- Profiles use shared cache packages to populate local mod folders
- Profiles are the runtime environment for launching Project Zomboid
