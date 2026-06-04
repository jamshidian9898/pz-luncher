# RFC 0014: Provider Priority

## Problem

Multiple providers may offer the same package, so the launcher needs a clear selection and fallback policy.

## Goals

- Define provider ordering for package resolution
- Ensure cached content is used first
- Support fallback from one provider to the next
- Keep provider selection deterministic and configurable

## Priority order

1. `LocalCacheProvider`
2. `RegistryProvider`
3. `ServerProvider`
4. `SteamProvider`

## Behavior

- `LocalCacheProvider` is queried first for already downloaded content
- `RegistryProvider` is queried before remote server or Steam sources
- `ServerProvider` is used for direct server-provided package URLs
- `SteamProvider` is a last resort for Steam-hosted package content

## Fallback rules

- if a provider reports `Exists` and download succeeds, no lower-priority provider is consulted
- if a provider fails to download, the next provider is tried
- providers may be disabled dynamically and excluded from resolution
- package metadata should be normalized across providers where possible
