# RFC 0005: Agent Lifecycle

## Problem

A server-side agent is required to generate manifests, publish status, and interact with the launcher ecosystem without modifying Project Zomboid.

## Goals

- Provide a lightweight agent for dedicated server hosts
- Report server health, player counts, and manifest updates
- Detect mod changes and generate manifests automatically
- Keep the agent loosely coupled to the launcher services

## Design

### Agent responsibilities

- Monitor mod folders and server configuration
- Generate and publish the current server manifest
- Emit periodic heartbeats to the directory or status service
- Optionally upload package metadata to the registry

### Agent API

- `POST /heartbeat` — reports server health and availability
- `PUT /manifest` — submits the current manifest
- `GET /config` (optional) — returns agent configuration or server settings

### Operation

- The agent runs beside the dedicated server binary
- It can be packaged as a small Go binary or cross-platform executable
- It should support local manifest file output for debugging

## Non-Goals

- Running game servers as a hosted service
- Performing heavy package transformation or delta generation
- Managing client-side downloads directly

## Open Questions

- Should the agent support automatic package uploads to the registry?
- How should the agent authenticate with backend services?
- Should the agent expose a local UI or CLI for server operators?
