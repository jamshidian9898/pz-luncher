# RFC-0032: Download Manager

**Status**: Active — Phase 1 Product  
**Depends on**: [RFC-0031](0031-mod-dependency-resolver.md)  
**Integrates with**: `libs/session` (existing executor), [RFC-0011](0011-download-session.md)  
**Feeds**: RFC-0033

---

## Problem

Players need reliable, resumable downloads with visible queue state—not ad-hoc per-mod calls.

---

## Goals

- Per-mod download queue with explicit states
- Retry with backoff; resume partial files
- SHA256 verification before marking complete
- Map to existing session executor where possible

## Non-goals

- P2P / torrent
- Bandwidth shaping beyond existing settings (`maxConcurrent`, `bandwidthLimit`)
- Multi-server parallel joins (one active join session in MVP)

---

## Queue model

```ts
export type DownloadState =
  | 'Pending'
  | 'Downloading'
  | 'Paused'
  | 'Failed'
  | 'Completed';

export interface DownloadItem {
  modId: string;
  state: DownloadState;
  bytesDone: number;
  bytesTotal: number;
  speedBps?: number;
  etaSeconds?: number;
  attempt: number;
  lastError?: string;
  checksumExpected: string;   // sha256
  checksumActual?: string;
  localPath?: string;         // temp path while downloading
}

export interface DownloadQueue {
  sessionId: string;
  serverId: string;
  items: DownloadItem[];
  startedAt: string;
  updatedAt: string;
}
```

### State transitions

```text
Pending → Downloading
Downloading → Completed | Failed | Paused
Paused → Downloading
Failed → Pending (retry) | terminal after max attempts
```

---

## Retry policy

| Setting | Default |
|---------|---------|
| `maxAttempts` | 3 |
| Backoff | 1s, 3s, 10s |
| Timeout per mod | 5 min (align with platform guarantee) |

On failure: set `Failed`, emit event; user may retry join (idempotent session).

---

## Resume

- Temp file: `cache/downloads/<sessionId>/<modId>.part`
- On restart: if `.part` exists and size < total → HTTP Range or re-fetch policy per provider
- Steam/workshop: delegate to `libs/session/steam_executor.go`

---

## Checksum

After download completes:

1. Compute SHA256 of file
2. Compare to `ResolvedMod.sha256`
3. Mismatch → `Failed`, code `DOWNLOAD_CHECKSUM_MISMATCH`, delete corrupt file

---

## Concurrency

- Respect `Settings.maxConcurrent` (default 2)
- Queue processes `ResolvedModPlan.orderedMods` in order; up to N active `Downloading`

---

## Persistence

`profiles/<serverId>/download-queue.json` — optional snapshot for UI resume after app restart.

---

## Events (UI)

| Event | Payload |
|-------|---------|
| `download.queued` | `{ modId, position }` |
| `download.progress` | `{ modId, bytesDone, bytesTotal, speedBps }` |
| `download.completed` | `{ modId }` |
| `download.failed` | `{ modId, code, message }` |

Patch schema: extend `downloadsPatchSchema` in `PatchSchemaRegistry`.

---

## Implementation map

| Layer | Responsibility |
|-------|----------------|
| `libs/session` | Execute single package download |
| New `libs/download` or `launcher-core/download` | Queue, retry, checksum orchestration |
| `launcher-ui` | `downloads.store` + `DownloadPanel` |

---

## Week 3 exit criteria

- [ ] Queue drives ≥ 1 mod through all states in fixture mode
- [ ] Checksum failure surfaced in UI
- [ ] Retry works after simulated network error
