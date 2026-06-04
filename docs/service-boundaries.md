# Service Boundaries

This project uses a microservice-style monorepo layout where each service owns a focused responsibility.

## Directory Service

- Publishes server metadata only
- Supports search, filtering, region, tags, favorites
- Exposes heartbeat and server status
- Does not store mod content

## Manifest Service

- Stores server manifests and manifest history
- Tracks required mods, checksums, dependencies, and download locations
- Exposes retrieval and rollback APIs

## Registry Service

- Stores packages in content-addressable storage by hash
- Serves package metadata and download endpoints
- Supports deduplication and shared cache semantics

## Launcher Core

- Orchestrates join flow and dependency resolution
- Checks cache and downloads missing content
- Prepares isolated profile directories
- Launches Project Zomboid with the correct mod environment

## Agent

- Runs next to a dedicated server
- Detects mod changes and generates manifests
- Reports server status, player counts, and optionally uploads content
- Uses server-local access to file and mod configuration state
