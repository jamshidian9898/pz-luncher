# RFC 0008: Provider System

## Problem

Package resolution must be flexible enough to use multiple content sources and provider types.

## Goals

- Define a generic provider interface for package existence and download
- Support local cache, registry, server, and Steam providers
- Control provider priority and fallback behavior
- Keep provider logic pluggable and testable

## Provider interface

A provider should expose:

```go
package provider

import "context"

type Package struct {
    ID       string
    Version  string
    SHA256   string
    Size     int64
    Provider string
    OriginURL string
}

type Provider interface {
    Name() string
    Priority() int
    Exists(ctx context.Context, pkg Package) (bool, error)
    Download(ctx context.Context, pkg Package, destination string) error
}
```

## Provider implementations

- `LocalCacheProvider`
- `RegistryProvider`
- `ServerProvider`
- `SteamProvider`

## Provider resolution

- providers are queried in priority order
- the first provider that reports `Exists` may be selected
- download failures can be retried with the next provider

## Open Questions

- Should providers announce package metadata or just existence?
- How should provider-specific errors be normalized?
