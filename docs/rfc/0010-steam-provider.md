# RFC 0010: Steam Provider

## Problem

Some package content may come from Steam Workshop or other Steam-hosted sources, requiring a dedicated provider.

## Goals

- Define how Steam-hosted mods are discovered and downloaded
- Integrate Steam provider into the generic provider system
- Keep Steam-specific logic isolated from core launcher flows

## Steam provider responsibilities

- validate Steam workshop package IDs
- resolve Steam download metadata and URLs
- support Steam authentication or token requirements
- expose `Exists` and `Download` semantics like other providers

## Implementation outline

- `SteamProvider` implements the provider interface
- uses Steam API or local Steam cache to check package availability
- downloads package payloads to the launcher cache
- reports provider priority lower than local cache and registry

## Open Questions

- Will Steam content be downloaded directly, or via an intermediate cache proxy?
- How are Steam package integrity and provider metadata represented?
- Should Steam provider support authenticated versus anonymous downloads?
