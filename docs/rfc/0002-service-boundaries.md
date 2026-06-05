# RFC 0002: Service Boundaries

> **v1.x historical record.** Superseded by [RFC-0050](0050-v2-architecture-rebaseline.md) for v2.0.0. The key v2 delta: Launcher→Agent coupling is removed; the Backend is the single control plane.

## Problem

The Project Zomboid launcher ecosystem requires multiple cooperating services, but the boundaries between those services are not yet clearly defined.

## Goals

- Define each service's responsibility
- Minimize overlap and coupling between services
- Keep APIs narrow and stable
- Make it easy to evolve services independently

## Services

### Directory Service

- Publishes server metadata, search, filtering, and heartbeat status
- Stores only metadata, not mod content
- Exposes public server discovery APIs

### Manifest Service

- Stores and versions server manifests
- Tracks manifest history and supports rollback
- Provides manifest retrieval for launcher and directory services

### Registry Service

- Stores content-addressable packages by hash
- Manages metadata and download endpoints for packages
- Supports deduplication and shared cache semantics

### Download Service

- Manages chunked downloads, resume, integrity, and parallelism
- Can act as a dedicated download edge service or internal shared component
- Translates registry package downloads into client-friendly flows

### Launcher Core

- Orchestrates server join flows, dependency resolution, and profile preparation
- Interfaces with manifest, registry, and download services
- Launches Project Zomboid in isolated profile environments

### Agent

> **v2.0.0**: Agent is Backend-managed infrastructure. Launcher never contacts the Agent directly.

- Runs beside a dedicated server
- Detects mod changes and generates manifests
- Reports server state, player counts, and status to **Backend only**
- Uploads manifest and package metadata to **Backend only**

## Non-Goals

- Combining all functionality into a single monolithic service
- Defining implementation details for UI or packaging formats

## Open Questions

- Should the download service be a separate app or a shared library used by launcher and registry?
- Should the agent be able to publish packages directly, or only manifests and status?
- How should cross-service authentication and trust be modeled?
