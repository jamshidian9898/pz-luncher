# RFC 0014: Provider Priority

> **v1.x historical record.** For v2.0.0 see [RFC-0050](0050-v2-architecture-rebaseline.md). The key v2 delta: `SteamProvider` and `ServerProvider` are removed from the Launcher. The v2 Launcher priority order is: `LocalCacheProvider` → `BackendProvider`. Content origin (Agent, Backend storage, SteamCMD) is resolved by the Backend before URLs are issued.

## Problem

Multiple providers may offer the same package, so the launcher needs a clear selection and fallback policy.

## Goals

- Define provider ordering for package resolution
- Ensure cached content is used first
- Support fallback from one provider to the next
- Keep provider selection deterministic and configurable

## Priority order (v1.x)

1. `LocalCacheProvider`
2. `RegistryProvider`
3. `ServerProvider`
4. `SteamProvider`

## Priority order (v2.0.0)

1. `LocalCacheProvider` — already-downloaded content, SHA256-keyed
2. `BackendProvider` — URLs issued by Backend `JoinResponse`

## Behavior (v1.x)

- `LocalCacheProvider` is queried first for already downloaded content
- `RegistryProvider` is queried before remote server or Steam sources
- `ServerProvider` is used for direct server-provided package URLs
- `SteamProvider` is a last resort for Steam-hosted package content

## Behavior (v2.0.0)

- `LocalCacheProvider` is queried first by SHA256 key
- `BackendProvider` is used for all cache misses; Backend has already resolved origin internally

## Fallback rules

- if a provider reports `Exists` and download succeeds, no lower-priority provider is consulted
- if a provider fails to download, the next provider is tried
- providers may be disabled dynamically and excluded from resolution
- package metadata should be normalized across providers where possible
