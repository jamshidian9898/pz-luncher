# RFC-0054: Backend Content Store

**Status**: Active — v2.0.0 Phase A  
**Depends on**: [RFC-0050](0050-v2-architecture-rebaseline.md), [RFC-0052](0052-backend-core-api.md)  
**Feeds**: RFC-0055

---

## Purpose

Define how the Backend stores, tracks, and resolves mod content. The content store is the Backend's internal cache that determines what download URLs it can issue to Launchers via `JoinResponse`.

---

## Content identity

All content is identified by SHA256 hash. Content is immutable once stored.

```text
key: sha256 (hex, 64 chars)
```

Two blobs with the same SHA256 are the same content. The store never stores duplicates.

---

## Content record

```json
{
  "sha256": "abc123...",
  "sizeBytes": 123456,
  "source": "agent",
  "serverId": "demo-survival",
  "modId": "example-mod",
  "cachedAt": "2026-06-05T00:00:00Z",
  "lastSeen": "2026-06-05T14:00:00Z",
  "storageBackend": "local"
}
```

| Field | Description |
|-------|-------------|
| `sha256` | Content identity key |
| `sizeBytes` | Blob size |
| `source` | How this content was acquired: `agent`, `steamcmd`, `upload` |
| `serverId` | Which server this content was first associated with |
| `modId` | Logical mod identifier (informational) |
| `cachedAt` | When first stored |
| `lastSeen` | Last time a `GET /join` referenced this hash |
| `storageBackend` | Where the blob lives: `local`, `s3`, `minio`, `r2` |

---

## Content sources (resolution order)

When `POST /join` requires a blob and the Backend must produce a download URL:

```text
1. Backend content store (sha256 lookup)
      ↓ miss
2. Agent content request (pull from relevant Agent)
      ↓ Agent offline or miss
3. SteamCMD fallback (last resort)
      ↓ success
4. Store in content store + issue URL
```

The Launcher sees none of this. It receives a URL and downloads the blob.

### Source: `agent`

Backend pulls the blob from the Agent that owns the server. Agent exposes a Backend-internal content endpoint. After pulling, the Backend stores the blob locally and can serve it without re-requesting the Agent.

### Source: `steamcmd`

Backend invokes SteamCMD for the Workshop item associated with the mod (via `workshopId` in the manifest). This is a slow, last-resort path. Once downloaded, the blob is stored and future requests are served from cache.

### Source: `upload`

Operators or CI systems may pre-upload content blobs directly to the Backend. Useful for private or custom mods without a Workshop presence.

---

## Storage backends

The content store is abstracted over a storage backend interface:

```text
Put(sha256, blob) error
Get(sha256) (blob, error)
Exists(sha256) (bool, error)
Delete(sha256) error
URL(sha256) (string, error)   // signed URL or direct path
```

Implementations in scope for v2.0.0:
- `local` — filesystem, for development and single-node deployments

Implementations for later:
- `s3` — AWS S3
- `minio` — self-hosted S3-compatible
- `r2` — Cloudflare R2
- `cdn` — CDN edge with origin pull

The Launcher never knows which storage backend is in use. All download URLs are issued by the Backend.

---

## Content expiry and garbage collection

- Content is never deleted if it is referenced by an active manifest
- Content may be evicted if: no manifest references it AND `lastSeen` is older than a configurable TTL (default: 90 days)
- GC runs as a background job, not inline with requests

---

## Deduplication

Because content is keyed by SHA256:
- The same mod version shared across multiple servers is stored exactly once
- When any server's Agent provides a blob, all servers benefit

---

## Non-goals

- Delta/binary-diff storage (v2.1+)
- Content replication across regions (v2.1+)
- Peer-to-peer distribution
- Client-side upload of content (Launcher never pushes blobs)
