# RFC 0003: Content Registry

## Problem

A universal launcher must distribute mod content reliably without duplicating downloads and while preserving integrity.

## Goals

- Use content-addressable storage for packaged mods and assets
- Avoid duplicate downloads by sharing cached content across profiles and servers
- Provide a registry API for upload, metadata lookup, and download
- Support multiple providers and external sources

## Design

### Package identity

Packages are identified by a strong content hash such as SHA256.
Each package includes metadata:
- `id`
- `version`
- `sha256`
- `size`
- `compression`
- `provider`
- `originUrl`

### Registry storage

- Store package blobs in a hashed object store
- Store metadata in a database keyed by hash
- Allow packages to be referenced by manifest entries and profile caches

### Download endpoints

- `GET /packages/{hash}` returns metadata
- `GET /packages/{hash}/download` returns the content blob
- `POST /packages` accepts uploads with metadata and optional source URLs

### Cache semantics

- A shared cache directory can reference package hashes
- Launcher and server profiles reuse cached package blobs
- Content is never modified after upload

## Non-Goals

- Serving arbitrary game configuration or save files
- Replacing provider-specific stores like Steam Workshop or local mod folders

## Open Questions

- Should the registry support package deduplication across provider-specific namespaces?
- How should package expiration and garbage collection be handled?
- Should the registry validate provider-supplied package checksums on upload?
