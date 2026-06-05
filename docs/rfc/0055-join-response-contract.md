# RFC-0055: JoinResponse Contract

**Status**: Active — v2.0.0 Phase A  
**Depends on**: [RFC-0050](0050-v2-architecture-rebaseline.md), [RFC-0052](0052-backend-core-api.md), [RFC-0054](0054-backend-content-store.md)  
**Consumed by**: Launcher pipeline (RFC-0033), RFC-0056

---

## Purpose

Define the exact contract of `POST /api/v1/join/{serverId}` → `JoinResponse`.

This is the **most important contract in v2.0.0**. It is the boundary between the Platform and the Launcher. Everything the Launcher needs to complete a join flow is in this response. The Launcher knows nothing about infrastructure after this point.

---

## The boundary

```text
Platform side                    Launcher side
─────────────────────────────────────────────
Backend resolves manifest        Launcher receives JoinResponse
Backend checks content store     Launcher resolves dep graph locally
Backend pulls from Agent         Launcher checks local cache (SHA256)
Backend falls back to SteamCMD   Launcher downloads from issued URLs
Backend issues signed URLs       Launcher verifies SHA256
─────────────────────────────────────────────
                     ↑
              JoinResponse
         (this RFC defines this line)
```

---

## Request

```http
POST /api/v1/join/{serverId}
Authorization: Bearer <launcher-token>
Content-Type: application/json
```

Optional body (for future use, empty in v2.0.0):

```json
{}
```

---

## Response (success — 200 OK)

```json
{
  "sessionId": "a3f2c1d4-...",
  "server": {
    "id": "demo-survival",
    "name": "Demo Survival",
    "gameVersion": "42.8",
    "region": "eu-west"
  },
  "manifest": {
    "serverId": "demo-survival",
    "version": "91",
    "gameVersion": "42.8",
    "mods": [
      {
        "id": "example-mod",
        "name": "Example Mod",
        "version": "1.2.3",
        "sha256": "abc123...",
        "sizeBytes": 123456,
        "workshopId": "1234567890",
        "dependencies": [],
        "optional": false
      }
    ],
    "launchArgs": ["-nosteam"],
    "profile": {
      "profileId": "demo-survival"
    }
  },
  "downloadPlan": [
    {
      "modId": "example-mod",
      "sha256": "abc123...",
      "sizeBytes": 123456,
      "url": "https://backend.example.com/api/v1/download/abc123...",
      "urlExpiresAt": "2026-06-05T15:00:00Z"
    }
  ],
  "issuedAt": "2026-06-05T14:00:00Z"
}
```

---

## Field definitions

### Top level

| Field | Type | Description |
|-------|------|-------------|
| `sessionId` | string (UUID) | Opaque session identifier for tracing. Include in `X-Session-ID` header on download requests. |
| `server` | object | Server metadata snapshot at join time |
| `manifest` | object | Current server manifest (RFC-0030 schema) |
| `downloadPlan` | array | One entry per mod requiring download |
| `issuedAt` | ISO 8601 | When this response was generated |

### `downloadPlan` item

| Field | Type | Description |
|-------|------|-------------|
| `modId` | string | Matches `manifest.mods[].id` |
| `sha256` | string | Expected SHA256 of the blob |
| `sizeBytes` | number | Expected byte size |
| `url` | string | Backend-issued download URL |
| `urlExpiresAt` | ISO 8601 | URL validity deadline. Launcher should begin download before this time. |

### `manifest.mods[].workshopId`

Informational provenance only. The Launcher MUST NOT use this field to construct download URLs. The `downloadPlan[].url` is the authoritative download source.

---

## Launcher behavior on receipt

1. Extract `manifest.mods[]` → feed to dependency resolver (RFC-0031)
2. For each mod in `downloadPlan`:
   - Check local cache by `sha256`
   - If cached and SHA256 matches → skip download
   - If not cached → download from `url` before `urlExpiresAt`
3. Verify SHA256 of each downloaded blob
4. Install to profile (RFC-0034)
5. Launch (RFC-0035)

The Launcher does not need to inspect `manifest.mods[].workshopId` or construct any URL. All URLs are in `downloadPlan`.

---

## Partial download plan

If all mods are already in the Launcher's local cache (SHA256 match), `downloadPlan` may be empty. The Launcher proceeds directly to install.

The Backend always returns `downloadPlan` entries for all mods; it does not know the Launcher's local cache state. Cache check is Launcher-local.

---

## URL expiry

Download URLs expire. Default TTL: 1 hour.

If the Launcher starts a download session and a URL expires mid-download:
- HTTP Range resume will fail with 403/410
- Launcher calls `POST /join` again to get a fresh `JoinResponse` with new URLs
- Session is idempotent: re-joining with the same `(serverId, manifest.version)` produces the same install result

---

## Invariants

1. `downloadPlan[].sha256` MUST match `manifest.mods[].sha256` for the corresponding `modId`
2. `downloadPlan` contains exactly the mods from `manifest.mods[]` that are not optional, plus any optional mods that are required by dependencies
3. All URLs in `downloadPlan` MUST be reachable by the Launcher at the time of response
4. `sessionId` is unique per response

---

## Error responses

See RFC-0052 for error codes (`JOIN_*`). All errors use the standard error envelope.
