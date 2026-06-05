# RFC 0010: Steam Provider

> **v1.x historical record.** For v2.0.0 see [RFC-0050](0050-v2-architecture-rebaseline.md). The key v2 delta: SteamCMD and all Steam content acquisition belong exclusively to the Backend. The Launcher has no SteamProvider, no SteamCMD binary, and no Steam Workshop awareness beyond informational `workshopId` fields in manifests. The Backend invokes SteamCMD as a last resort when client cache misses, Backend storage misses, and Agent content is unavailable.

## Problem

Some package content may come from Steam Workshop or other Steam-hosted sources, requiring a dedicated provider.

## Goals

- Define how Steam-hosted mods are discovered and downloaded
- Integrate Steam provider into the generic provider system
- Keep Steam-specific logic isolated from core launcher flows

## Steam provider responsibilities (v1.x)

- validate Steam workshop package IDs
- resolve Steam download metadata and URLs
- support Steam authentication or token requirements
- expose `Exists` and `Download` semantics like other providers

## v2.0.0 placement

All Steam-related responsibilities move to the **Backend**:

- Backend validates Workshop IDs when ingesting manifests from Agents
- Backend acquires Steam content via SteamCMD as a last-resort fallback
- Backend stores acquired content and issues signed download URLs to the Launcher
- `workshopId` in `ModEntry` is **informational provenance only**; Launcher never acts on it

## Open Questions

- Will Steam content be downloaded directly, or via an intermediate cache proxy?
- How are Steam package integrity and provider metadata represented?
- Should Steam provider support authenticated versus anonymous downloads?
