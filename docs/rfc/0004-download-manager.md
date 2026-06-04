# RFC 0004: Download Manager

## Problem

Launcher downloads can be large and unreliable, and the system needs to support resume, integrity checks, and efficient delivery.

## Goals

- Provide robust resume support for interrupted downloads
- Support parallel downloads and chunked transfer
- Verify integrity for each package before use
- Enable delta updates where possible

## Design

### Download flow

1. Resolve package metadata and download URL
2. Check local shared cache and profile cache
3. If missing, request chunked download from `download-service`
4. Validate package hash after download
5. Store package in shared cache and profile-specific locations

### Chunking and resume

- Download endpoints should support range requests or explicit chunk ranges
- The download manager tracks progress per package and resumes incomplete chunks
- Partial downloads are stored in temporary files until complete

### Delta updates

- When available, the manager can download only changed blocks
- The manifest can reference delta metadata for supported packages
- Delta updates are optional and fall back to full package download

### Integrity checks

- Validate package hash against manifest SHA256
- Optionally validate signatures or provider checksums

## Non-Goals

- Building a full CDN or distribution network in the first phase
- Replacing native game download facilities for Steam/Workshop

## Open Questions

- What delta format should the system use for partial package updates?
- Should the download manager expose a generic `resume` API to all clients, or only internally to launcher?
- How should the system prioritize downloads for faster join times?
