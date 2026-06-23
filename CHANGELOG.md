# Changelog

## v2.0.0 — Architecture Rebaseline (Active)

**RFC-0050**: v2.0.0 Architecture Rebaseline — Launcher + Backend + Agent model  
**Decision**: [PRODUCT_DECISION.md](PRODUCT_DECISION.md) — Path A (Product Only)

### Breaking changes

#### Authentication
- Agent authentication now uses **Bearer tokens** issued at enrollment time (RFC-0053)
- Enrollment uses a single-use, 24h-TTL operator token scoped to a `serverId`
- Previous session-based agent auth is removed

#### JoinResponse contract (RFC-0055)
- `POST /api/v1/join/{serverId}` now returns a structured `JoinResponse`
- Response includes: `sessionId`, `server`, `manifest` (RFC-0030 format), `downloadPlan[]`
- Each download entry carries `sha256`, `sizeBytes`, `url` — Launcher verifies SHA256 after download
- Old flat response format is removed

#### Manifest format (RFC-0030)
- Manifests now follow `ServerManifest` v1 schema (`libs/manifestv1`)
- Legacy RFC-0001 manifest format is still parsed via `convert.go` but deprecated
- All new manifests must use v1 JSON schema (`shared/contracts/manifest.schema.json`)

#### Settings (RFC-0036)
- New field `backendUrl` added to launcher settings
- Settings file: `config/launcher-settings.json`
- Schema: `shared/contracts/settings.schema.json`

### New in v2.0.0

- **Backend**: Go HTTP server (`apps/backend/`) — replaces direct launcher→agent calls
- **Agent** (`apps/pz-agent/`): headless mod discovery and content serving
- **Content-addressable storage**: all mod blobs keyed by SHA256
- **Installation pipeline** (RFC-0033): 8-stage join flow (manifest → resolve → download → profile → ready)
- **Per-server profile isolation** (RFC-0034): separate `mods/`, `saves/`, `config/` per server
- **Prometheus metrics**: `GET /metrics` on backend

### Libraries introduced in v2.0.0

| Library | Replaces | RFC |
|---------|----------|-----|
| `libs/manifestv1` | `libs/manifest` | 0030 |
| `libs/modplan` | `libs/resolver` | 0031 |
| `libs/download` | `libs/downloader` | 0032 |
| `libs/pipeline` | `libs/launchercore` | 0033 |

---

## v1.x — Platform Kernel (Frozen)

Go platform core: session manager, Steam executor, chaos/shadow/campaign validation.  
Infrastructure RFCs 0022–0025 (event bus, reducer, event log, snapshot) are frozen.  
No changes to core interfaces. Extension via plugin boundary only.
