# RFC 0018: Launcher State Machine

## Problem

Launcher Core needs a clear runtime state machine to coordinate manifest resolution, package download, profile construction, and game launch.

## Goals

- Define the core launcher lifecycle states
- Keep transitions explicit and traceable
- Make failure and retry paths visible
- Ensure the state machine can drive UI and logs

## States

```text
Idle
↓
ResolvingManifest
↓
ResolvingPackages
↓
CheckingCache
↓
Downloading
↓
BuildingProfile
↓
Launching
↓
Running
↓
Stopped
```

## Description

- `Idle` — waiting for join or launch request
- `ResolvingManifest` — fetching and validating the manifest
- `ResolvingPackages` — mapping manifest mods to resolved package records
- `CheckingCache` — verifying existing local cache content
- `Downloading` — fetching missing package blobs
- `BuildingProfile` — assembling the profile filesystem
- `Launching` — starting Project Zomboid with the profile
- `Running` — game process is active
- `Stopped` — the session has ended or been cancelled

## Invariants

- transitions must follow the defined order unless a recovery path is triggered
- failures from any state may transition to `Stopped` or retry within the state machine
- `Running` must only occur after successful profile build and launch
- `Stopped` is the terminal state after a session ends
