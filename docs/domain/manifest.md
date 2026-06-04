# Domain: Manifest

## Entity: Manifest

Fields
- `id` (string): unique manifest identifier
- `serverId` (string): owning server id
- `version` (int): manifest version number
- `gameVersion` (string): Project Zomboid build version
- `createdAt` (timestamp): creation time
- `mods` ([]ManifestMod)
- `checksum` (string): manifest content hash

`ManifestMod` fields
- `id` (string)
- `version` (string)
- `sha256` (string)
- `downloadUrl` (string)
- `dependencies` ([]string)

Rules
- `serverId` is immutable after manifest creation
- `version` increases monotonically for a server
- `checksum` is deterministic and derived from manifest contents
- `mods` dependency graph must be acyclic
- `gameVersion` must match the server's published supported game version

State transitions
- new manifest is created for a server when mod requirements change
- old manifests are retained for rollback
- a manifest can be marked deprecated if superseded

Relations
- Each manifest belongs to one server
- A server holds a latest manifest pointer and history of older manifests
