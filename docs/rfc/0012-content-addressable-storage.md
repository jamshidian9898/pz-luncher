# RFC 0012: Content Addressable Storage

## Problem

The system must store mod packages and blobs efficiently and verify content by hash.

## Goals

- Use hashes as the primary identity for stored content
- Avoid duplicate blob storage
- Enable integrity verification on ingestion and retrieval
- Support shared caching across multiple profiles

## Storage layout

```text
content/
  sha256/
    {hash}
metadata/
  {hash}.json
```

## Principles

- content blobs are immutable once written
- hash content is the canonical address for retrieval
- metadata is separate from blob storage
- storage may be local, S3-compatible, or derived from a registry

## Behavior

- a package can be referenced by hash in manifests, cache entries, and provider records
- retrieval validates the requested hash against actual blob content
- storage may expose deduplication by hard links or object storage semantics

## Open Questions

- Should the storage system support variant-specific content addresses?
- How will partial or delta objects be represented?
- Should garbage collection be based on reference counts or lease agreements?
