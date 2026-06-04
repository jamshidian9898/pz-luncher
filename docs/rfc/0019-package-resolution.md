# RFC 0019: Package Resolution

## Problem

Launcher Core must transform a manifest into a deterministic set of resolved packages and provider assignments.

## Goals

- define a manifest-to-package resolution flow
- build and validate a dependency graph
- perform a topological sort of package dependencies
- produce a list of resolved packages with provider metadata

## Flow

```text
Manifest
↓
Dependency Graph
↓
Topological Sort
↓
Resolved Packages
↓
Providers
```

## Behavior

- `Manifest` is parsed and normalized
- each mod entry becomes a package node
- the dependency graph is validated for acyclicity
- packages are ordered so dependencies appear before consumers
- provider selection is attached to each resolved package

## Invariants

- dependency graphs must be acyclic
- resolved packages must preserve manifest identity and integrity
- provider assignment must be deterministic and follow configured priority
