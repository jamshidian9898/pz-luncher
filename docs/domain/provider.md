# Domain: Provider

## Entity: Provider

Fields
- `name` (string): provider name
- `type` (string): provider type, e.g. `local`, `registry`, `server`, `steam`
- `priority` (int): resolution priority order
- `enabled` (bool): whether this provider is active
- `config` (map[string]any): provider-specific configuration

Rules
- provider names are unique within a launcher installation
- `priority` determines search order for package resolution
- providers may be enabled or disabled dynamically

State transitions
- provider added → ready to resolve packages
- provider disabled → skipped during package resolution
- provider removed → no longer queried

Relations
- Providers resolve `Package` entities
- Providers are queried by launcher core and download manager
- Providers are referenced in manifests only by package download metadata rather than by provider reference directly
