# RFC 0021: Launch State Machine

## Problem

Launcher Core needs a defined runtime state machine to coordinate the full join and launch lifecycle.

## Goals

- define core launcher states explicitly
- support failure and recovery transitions
- make the system observable to future UI and diagnostics

## States

```text
Idle

ResolvingManifest

ResolvingPackages

CheckingCache

Downloading

BuildingProfile

Launching

Running

Stopped

Failed
```

## Transitions

- `Idle` → `ResolvingManifest`
- `ResolvingManifest` → `ResolvingPackages`
- `ResolvingPackages` → `CheckingCache`
- `CheckingCache` → `Downloading` or `BuildingProfile`
- `Downloading` → `BuildingProfile`
- `BuildingProfile` → `Launching`
- `Launching` → `Running`
- `Running` → `Stopped`
- any non-retriable failure → `Failed`

## Invariants

- state transitions must be linear and observable
- `Running` is only reached after a successful profile build and launch
- failures should clearly map to `Failed` or `Stopped`
