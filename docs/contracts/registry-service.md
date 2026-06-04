# Contract: Registry Service

## Responsibility

The registry service stores content-addressable packages and provides metadata and download endpoints.

## API surface

- `POST /packages`
  - upload package metadata and register the package blob
- `GET /packages/{hash}`
  - retrieve package metadata by content hash
- `GET /packages/{hash}/download`
  - download the package blob

## Data contracts

- Package metadata includes `id`, `version`, `sha256`, `size`, `provider`, `originUrl`, and dependency metadata
- Downloads are validated against the reported `sha256`

## Invariants

- registry entries are immutable by hash
- integrity verification is required for all downloads
- registry stores only package metadata and blobs, not server manifests
