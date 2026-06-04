# Contract: Launcher Core

## Responsibility

Launcher Core orchestrates the join flow, package resolution, profile population, and game launch.

## API surface

This contract refers to internal core services rather than external HTTP endpoints. It defines the orchestration boundaries and expected inputs.

### Core interface

- `JoinServer(serverID string) error`
- `ResolveManifest(serverID string) error`
- `ResolvePackages() error`
- `DownloadMissing() error`
- `PrepareProfile() error`
- `Launch() error`

### Inputs and outputs

- `serverID` — server identifier selected by the user
- `manifest` — latest server manifest with mods and package metadata
- `packages` — resolved package set needed for the profile
- `profile` — isolated profile environment built for the server

## Invariants

- launcher core must never launch the game with missing or invalid package content
- package resolution must prefer cached content before downloading
- a profile must be fully synced before `LaunchGame` succeeds
- failed downloads or invalid manifests must surface a recoverable error

## Contracts with other services

- uses Manifest Service to obtain server manifests
- uses Registry Service and Provider system to resolve package content
- uses Download Service to fetch missing blobs
- updates Profile metadata and cache references during build
