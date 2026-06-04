# Domain: Cache

## Entity: Cache

Fields
- `hash` (string): package content hash, typically SHA256
- `size` (int64): byte size of cached blob
- `path` (string): file system path to cache blob
- `referenceCount` (int): number of profiles or packages referencing it
- `createdAt` (timestamp)
- `lastAccessedAt` (timestamp)
- `integrityState` (string): `valid`, `invalid`, `repairing`

Rules
- cache blobs are content-addressed and immutable once stored
- `hash` is the primary identity for cached content
- `referenceCount` is incremented when a profile uses the content and decremented when released
- `size` must match the content blob

State transitions
- cache entry created when a package is downloaded or published
- reference count changes when profiles start/end using a package
- cache entry may be garbage-collected when referenceCount reaches 0 after retention
- integrity state moves to `invalid` if verification fails

Relations
- Cache entries are referenced by package records and profile provisioning
- The launcher and download manager query cache entries before downloading
- Cache entries may be shared globally across profiles and servers
