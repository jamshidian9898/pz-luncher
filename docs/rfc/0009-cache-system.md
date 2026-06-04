# RFC 0009: Cache System

## Problem

The cache is the central mechanism for deduplicating downloads and sharing package content across profiles.

## Goals

- Use content-addressable storage for cached blobs
- Maintain metadata and reference counting
- Support garbage collection for unused objects
- Verify integrity of cached content

## Cache layout

```text
cache/
  sha256/
    {hash}
  metadata/
    {hash}.json
```

## Cache entries

Fields
- `hash` (string)
- `size` (int64)
- `path` (string)
- `referenceCount` (int)
- `createdAt` (timestamp)
- `lastAccessedAt` (timestamp)
- `state` (`valid`, `invalid`, `repairing`)

## Deduplication

- cache entries are identified by hash
- multiple profiles may reference the same blob
- shared cache blobs are never duplicated on disk unless required by filesystem semantics

## Garbage collection

- remove entries when `referenceCount` reaches 0 and retention period expires
- verify `lastAccessedAt` before deleting
- support manual and automated cleanup

## Integrity

- validate blob hash on ingest and periodically
- mark blobs invalid if corruption is detected
- attempt repair from provider or redeploy if possible
