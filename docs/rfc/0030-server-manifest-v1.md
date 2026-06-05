# RFC-0030: Server Manifest v1

**Status**: Active — Phase 1 Product (v1.x) / Updated for v2.0.0 — see [RFC-0050](0050-v2-architecture-rebaseline.md)  
**Priority**: P0 (blocks all domain work)  
**Supersedes**: Partially clarifies [RFC-0001](0001-manifest-format.md) for launcher v1  
**Consumers**: RFC-0031, RFC-0032, RFC-0033, RFC-0034, RFC-0035

---

## Problem

Players join servers via a **single, versioned contract** that describes game build, mods, launch parameters, and profile hints. Earlier docs ([manifest-schema.md](../manifest-schema.md), RFC-0001) defined a minimal JSON shape; this RFC is the **implementation contract** for Phase 1.

---

## Goals

- One manifest per server revision (`serverId` + `version`)
- Deterministic parsing in Go (`libs/contracts`) and TypeScript (`launcher-ui`)
- Validation before any download or profile work
- Backward-compatible mapping from RFC-0001 JSON where possible

## Non-goals

- Manifest service HTTP API (local file / URL fetch only in v1)
- Optional mod packs / feature flags (v2)
- Server browser metadata (name, players) — lives on `ServerInfo`, not manifest

---

## TypeScript contract (canonical for UI)

```ts
export interface ServerManifest {
  serverId: string;
  version: string;           // manifest revision, semver or integer string
  gameVersion: string;       // PZ build, e.g. "42.8"

  mods: ModEntry[];

  launchArgs: string[];      // extra CLI args for dedicated client launch

  profile: ProfileConfig;
}

export interface ModEntry {
  id: string;                // stable package id (slug)
  name: string;              // display name
  version: string;
  sha256: string;            // expected content hash
  sizeBytes?: number;

  workshopId?: string;       // INFORMATIONAL ONLY (v2.0.0) — provenance record, not used by Launcher for download
  // downloadUrl removed in v2.0.0 — download URLs are issued by Backend in JoinResponse, not embedded in manifest

  dependencies: string[];    // ids of required mods, in manifest only

  optional?: boolean;        // default false
}

export interface ProfileConfig {
  profileId?: string;        // default: serverId
  modLoadOrder?: string[];     // explicit order; if omitted, resolver orders
  env?: Record<string, string>;
}
```

---

## JSON on disk (wire format)

```json
{
  "serverId": "demo-survival",
  "version": "91",
  "gameVersion": "42.8",
  "mods": [
    {
      "id": "example-mod",
      "name": "Example Mod",
      "version": "1.2.3",
      "sha256": "abc123...",
      "workshopId": "1234567890",
      "dependencies": []
    }
  ],
  "launchArgs": ["-nosteam"],
  "profile": {
    "profileId": "demo-survival"
  }
}
```

### Mapping from RFC-0001

| RFC-0001 | RFC-0030 |
|----------|----------|
| `manifestVersion` | `version` |
| `mods[].id` | `mods[].id` |
| (missing) | `serverId` required at root |
| (missing) | `launchArgs`, `profile` |

---

## Validation rules

| Rule | Error code |
|------|------------|
| `serverId`, `version`, `gameVersion` non-empty | `MANIFEST_INVALID_META` |
| Each mod has `id`, `version`, `sha256` | `MANIFEST_INVALID_MOD` |
| Mod `id` unique within manifest | `MANIFEST_DUPLICATE_MOD` |
| `dependencies[]` reference existing mod ids | `MANIFEST_UNKNOWN_DEP` |
| `sha256` hex, 64 chars | `MANIFEST_INVALID_HASH` |
| `gameVersion` matches launcher-supported range (config) | `MANIFEST_UNSUPPORTED_GAME` |

---

## Fetch & storage (v1)

```text
User selects server
    → launcher resolves manifest URL (server descriptor or bundled fixture)
    → GET file / read local path
    → validate → pass to RFC-0031
```

- Cache parsed manifest: `profiles/<serverId>/manifest-<version>.json`
- On version change: trigger re-resolve (RFC-0033)

---

## Events (UI patch targets)

Emit after validation (integrate with existing `LauncherEvent`):

- `manifest.loaded` — `{ serverId, version, modCount }`
- `manifest.failed` — `{ code, message }`

---

## Implementation checklist

- [ ] `libs/contracts/manifest_v1.go` (or extend `manifest.go`)
- [ ] `fixtures/manifests/demo-survival.json`
- [ ] `launcher-core`: load + validate on join
- [ ] `launcher-ui`: types in `types.ts` aligned with this RFC

---

## v2.0.0 notes

- `downloadUrl` is removed from `ModEntry`. Download URLs are issued by the Backend `POST /join/{serverId}` response (`JoinResponse.downloadPlan`).
- `workshopId` is retained as **informational provenance only**. The Launcher MUST NOT use it to construct download URLs.
- Manifest is fetched from the Backend; local file / URL fetch (v1.x) is superseded.

## Open questions (v1.x, resolved)

1. Manifest URL: per-server field in directory vs hardcoded demo? — **v2: manifest from Backend join response**
2. Semver vs integer for `version`? — integer ok
3. Require `workshopId` OR `downloadUrl` per mod? — **v2: neither required for download; Backend issues URLs**
