# RFC-0059: Content Registry

**Status**: Active — v2.1.0  
**Depends on**: [RFC-0054](0054-backend-content-store.md), [RFC-0060](0060-community-upload.md)  
**Feeds**: RFC-0058, RFC-0061

---

## Purpose

Define the Backend's Content Registry — the central store for game client binaries and mod files. The Registry serves players who cannot reach Steam (e.g. Iran) and provides a verified backup when agent servers are offline.

The Registry is **self-hosted** on infrastructure accessible from restricted regions (Hetzner Germany/Finland). No dependency on Cloudflare, Steam CDN, or Google infrastructure.

---

## Content types

| Type | Example | Source | Size |
|------|---------|--------|------|
| Game client binary | PZ v42.16 (Windows) | Community upload | ~3.2 GB |
| Game client binary | PZ v42.16 (Linux) | Community upload | ~3.0 GB |
| Mod archive | workshop-2392709985.zip | Agent push | 1 MB – 500 MB |

---

## Trust levels

Every piece of content in the Registry has a trust level:

```
✅ Verified
   → Multiple independent uploaders submitted this content
   → All SHA256 hashes match
   → Threshold: min 3 independent sources (configurable)
   → Safe to serve to all players

⚠️ Pending
   → Uploaded by fewer than threshold sources
   → SHA256 not yet cross-validated
   → Shown to users with warning

❌ Rejected
   → SHA256 mismatch between uploaders
   → Quarantined, not served
   → Admin notified
```

---

## Content record schema

```json
{
  "sha256": "abc123...",
  "contentType": "game-binary",
  "gameId": "pz",
  "gameVersion": "42.16",
  "platform": "windows-x64",
  "sizeBytes": 3200000000,
  "trustLevel": "verified",
  "uploadCount": 4,
  "firstUploadedAt": "2026-06-01T10:00:00Z",
  "lastVerifiedAt": "2026-06-20T08:00:00Z",
  "storageBackend": "local"
}
```

---

## Storage layout

```
/registry/
  versions/
    pz/
      42.16/
        windows-x64.zip      ← game client binary archive
        windows-x64.sha256   ← checksum file
        linux-x64.zip
        linux-x64.sha256
      42.20/
        ...
  mods/
    {sha256}/
      content.zip
      meta.json
```

Storage backend for MVP: local disk on the same VPS as Backend.  
Future: pluggable (MinIO, S3, Backblaze B2).

---

## API endpoints

### List available versions

```http
GET /api/v1/registry/versions?game=pz&platform=windows-x64

Response:
{
  "versions": [
    {
      "gameVersion": "42.16",
      "platform": "windows-x64",
      "sha256": "abc123...",
      "sizeBytes": 3200000000,
      "trustLevel": "verified",
      "downloadUrl": "/api/v1/registry/versions/pz/42.16/windows-x64"
    }
  ]
}
```

### Download a version

```http
GET /api/v1/registry/versions/{game}/{version}/{platform}

→ Streams binary archive (zip)
→ 404 if not in registry
→ Headers include X-SHA256, Content-Length
```

### Get mod from registry

```http
GET /api/v1/registry/mods/{sha256}

→ Streams mod archive
→ Same as RFC-0054 /download/{sha256} — unified endpoint
```

### Get version catalog (for Launcher)

```http
GET /api/v1/registry/catalog?game=pz

→ Full version list with trust levels and available platforms
→ Cached 1 hour, served stale-while-revalidate
→ Small payload — Launcher downloads on startup
```

---

## Source resolution order (on join)

When a player joins a server and needs content, the Launcher queries sources in this order:

```
1. Local cache (already downloaded)          → fastest, free
2. Agent URL (server host serves directly)   → verified, hoster bandwidth
3. Registry (our server)                     → verified, our bandwidth
4. Hoster upload in registry (unverified)    → user must confirm
```

The Launcher always shows the user which sources are available and lets them choose.

---

## Bandwidth management (self-hosted constraint)

Since the Registry runs on a single VPS with limited bandwidth:

- Game binaries are served with rate limiting (default: 10 MB/s per connection)
- Agents are preferred as the primary source — registry is backup
- Future: torrent seeding between Launcher clients (post-MVP)

---

## Infrastructure requirements

| Requirement | Spec |
|-------------|------|
| Server location | Hetzner Germany or Finland |
| Accessibility | Must be reachable from Iran without VPN |
| Storage | Minimum 50 GB for initial version set |
| Bandwidth | Hetzner includes 20 TB/month — sufficient for early stage |
| Protocol | Plain HTTPS, no WebSocket, no Cloudflare proxy |
| TLS | Let's Encrypt (auto-renew) |

---

## Out of scope

- Virus scanning of uploaded binaries (cross-validation is the trust mechanism)
- CDN layer (add post-MVP when bandwidth becomes a bottleneck)
- Paid storage tiers
- Non-PZ games (architecture is game-agnostic, registration is future)
