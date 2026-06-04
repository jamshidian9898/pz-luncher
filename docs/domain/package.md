# Domain: Package

## Entity: Package

Fields
- `id` (string): package identifier (provider-agnostic or internal)
- `version` (string): package version string
- `sha256` (string): content hash of the package blob
- `size` (int64): byte size of package content
- `provider` (string): source provider name
- `originUrl` (string): original source URL
- `dependencies` ([]string): package dependency ids
- `metadata` (map[string]string]): provider metadata

Rules
- `sha256` is immutable once the package blob is published
- `version` is immutable for a given package release
- dependencies must form an acyclic graph
- `size` must match the package blob contents
- package identity is defined by content hash and provider semantics

State transitions
- a package record is created when metadata is published
- package metadata may be enriched, but core identity fields remain stable
- packages are marked deprecated or retired when no longer valid

Relations
- Packages are referenced by manifests and provider lookups
- Packages may be stored in the registry and referenced by hash
- Packages may be shared between profiles via the global cache
