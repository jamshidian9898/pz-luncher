# Roadmap

This roadmap describes the first-phase scaffold and the next key milestones.

## Phase 1: Architecture and docs

- Create monorepo structure and service boundaries
- Define manifest format, API contracts, and profile isolation
- Build initial docs from `ProjectBaseDocs`
- Create starter app and library placeholders

## Phase 2: Core services

- Implement `directory-service` API and server discovery
- Implement `manifest-service` with manifest history and rollback
- Implement `registry-service` with content addressable storage
- Implement `launcher-core` join and resolve flow
- Implement `pz-agent` manifest generation and heartbeat

## Phase 3: Client and experience

- Build `launcher-ui` for server browsing and join flow
- Add download manager and resume support in `download-service`
- Add live updates via `websocket-gateway`
- Integrate `steam` or external provider adapters

## Phase 4: polish and distribution

- Add installers for Windows, macOS, and Linux
- Add smart caching and delta update support
- Add rollback UI and profile management
- Harden security and integrity checks
