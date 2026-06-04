# RFC-0033: Installation Pipeline

**Status**: Active — Phase 1 Product  
**Depends on**: RFC-0030, RFC-0031, RFC-0032  
**Aligns with**: [RFC-0020](0020-game-installation.md), [RFC-0015](0015-profile-build.md)

---

## Problem

Join flow must be one **orchestrated pipeline**, not disconnected steps. Users see a single progress story; developers get idempotent, traceable stages.

---

## Pipeline (canonical)

```text
resolve manifest          (RFC-0030)
        ↓
resolve mod plan          (RFC-0031)
        ↓
create download plan      (RFC-0032)
        ↓
download                  (RFC-0032 + session executor)
        ↓
verify                    (checksum per mod)
        ↓
install                   (extract/link into content cache)
        ↓
activate profile          (RFC-0034)
        ↓
ready to launch           (RFC-0035)
```

---

## Stage definitions

| Stage | Input | Output | Failure codes |
|-------|--------|--------|----------------|
| `ResolveManifest` | serverId | `ServerManifest` | `PIPELINE_MANIFEST` |
| `ResolveMods` | manifest | `ResolvedModPlan` | `PIPELINE_RESOLVER` |
| `PlanDownloads` | plan | `DownloadQueue` | `PIPELINE_PLAN` |
| `Download` | queue | completed queue | `PIPELINE_DOWNLOAD` |
| `Verify` | files + hashes | verify report | `PIPELINE_VERIFY` |
| `Install` | verified blobs | cache entries | `PIPELINE_INSTALL` |
| `ActivateProfile` | plan + cache | profile path | `PIPELINE_PROFILE` |
| `Ready` | profile | launch token | — |

---

## Idempotency

Re-running join for same `(serverId, manifest.version)`:

| Stage | Skip if |
|-------|---------|
| Download | mod in cache with matching sha256 |
| Verify | already verified in session trace |
| Install | profile mod path exists + hash ok |
| ActivateProfile | profile manifest version matches |

Session id: deterministic from `serverId + version` (existing session manager behavior).

---

## Trace artifact

Write after each stage:

`profiles/<serverId>/join-trace-<sessionId>.json`

```json
{
  "serverId": "...",
  "manifestVersion": "...",
  "stages": [
    { "name": "ResolveManifest", "status": "ok", "ms": 12 },
    { "name": "Download", "status": "ok", "ms": 45000, "mods": 3 }
  ]
}
```

UI `TraceViewer` reads this file (already exists).

---

## Coordinator API

```ts
export interface JoinPipeline {
  start(serverId: string): Promise<void>;
  cancel(): Promise<void>;
  getStage(): PipelineStage;
}

export type PipelineStage =
  | 'ResolveManifest'
  | 'ResolveMods'
  | 'PlanDownloads'
  | 'Download'
  | 'Verify'
  | 'Install'
  | 'ActivateProfile'
  | 'Ready'
  | 'Failed';
```

Go: orchestrate inside `launcher-core` `JoinServer()` — single entry point.

---

## UI mapping

`SessionStatus.state` (existing):

| Pipeline stage | UI state |
|----------------|----------|
| Resolve* | `resolving` |
| Download, Verify | `downloading` |
| Install, ActivateProfile | `installing` |
| Ready | `complete` |
| Failed | `error` |

---

## Week 4 exit criteria

- [ ] `JoinServer` runs full pipeline on demo fixture
- [ ] Trace JSON written and visible in UI
- [ ] Second join skips completed mods (idempotency test)
