# Architecture

The project is organized as a monorepo with separate apps and shared libraries.

## Architecture layers

```text
Directory Service
        ↓
Manifest System
        ↓
Content Registry
        ↓
Launcher Client
        ↓
Profile Isolation
        ↓
Project Zomboid
```

## Component overview

- `apps/directory-service`: public server list, search, filtering, favorties, tags, and heartbeats
- `apps/registry-service`: package storage and content addressable registry
- `apps/manifest-service`: manifest history, versioning, and rollback
- `apps/launcher-core`: launch orchestration, dependency resolution, and profile preparation
- `apps/launcher-ui`: user-facing interface for server discovery and join flow
- `apps/pz-agent`: server-side agent that detects mod changes, generates manifests, and reports status
- `apps/download-service`: download manager with resume, parallelism, and integrity checks
- `apps/websocket-gateway`: live updates and status streaming

## Shared libraries

- `libs/manifest`: manifest format, validation, and history helpers
- `libs/package`: package metadata and content manifest utilities
- `libs/profile`: profile isolation rules and layout helpers
- `libs/hashing`: hashing utilities for content-addressable storage
- `libs/storage`: S3/MinIO and local cache storage helpers
- `libs/providers`: external provider interfaces and adapters
- `libs/steam`: Steam Workshop or external provider integration helpers
- `libs/cache`: shared cache policies and deduplication logic
- `libs/downloader`: chunked downloads, resume, and delta update helpers
- `libs/telemetry`: metrics, logging, and monitoring support
- `libs/logger`: structured logger wrappers
- `libs/contracts`: API contracts and DTO definitions
