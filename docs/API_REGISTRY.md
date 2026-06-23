# Content Registry API (RFC-0059)

**Base URL**: `http://registry.pzlauncher.io` (or your self-hosted instance)

---

## Authentication

No authentication required for read operations (public registry).

Write operations (upload) are not gated by token in MVP — open submission model with cross-validation.

---

## Endpoints

### GET `/api/v1/registry/catalog?game=pz`

List all versions in the registry for a game.

**Parameters:**
- `game` (string): Game ID, e.g. `pz`

**Response:**
```json
{
  "versions": [
    {
      "gameVersion": "42.16",
      "platform": "windows-x64",
      "sizeBytes": 3200000000,
      "trustLevel": "verified",
      "uploadCount": 5,
      "firstUploadedAt": "2026-06-01T10:00:00Z",
      "lastVerifiedAt": "2026-06-20T08:00:00Z"
    },
    {
      "gameVersion": "42.20",
      "platform": "windows-x64",
      "sizeBytes": 3250000000,
      "trustLevel": "verified",
      "uploadCount": 3,
      "firstUploadedAt": "2026-06-15T14:30:00Z",
      "lastVerifiedAt": "2026-06-22T12:00:00Z"
    }
  ]
}
```

**Status**: `200 OK`

---

### GET `/api/v1/registry/versions?game=pz&platform=windows-x64&trust=verified`

List versions with optional filters.

**Parameters:**
- `game` (string): Game ID
- `platform` (string, optional): Filter by platform (e.g. `windows-x64`, `linux-x64`)
- `trust` (string, optional): Filter by trust level (`pending`, `verified`, `rejected`)

**Response:**
```json
{
  "versions": [
    {
      "gameVersion": "42.16",
      "platform": "windows-x64",
      "sizeBytes": 3200000000,
      "trustLevel": "verified",
      "uploadCount": 5
    }
  ]
}
```

**Status**: `200 OK`

---

### GET `/api/v1/registry/versions/{game}/{version}/{platform}`

Download a game client binary.

**Parameters:**
- `game` (path): Game ID (e.g. `pz`)
- `version` (path): Game version (e.g. `42.16`)
- `platform` (path): Platform (e.g. `windows-x64`)

**Response:**
- Binary file (octet-stream)
- Headers:
  - `X-Content-SHA256`: SHA256 hash of the binary
  - `X-Trust-Level`: Trust level (verified / pending / rejected)
  - `Content-Length`: File size in bytes
  - `Cache-Control: public, max-age=31536000, immutable`

**Status**: 
- `200 OK` — Download started
- `404 Not Found` — Version not in registry
- `409 Conflict` — Version marked as rejected (hash conflict)
- `503 Service Unavailable` — Blob store not configured

**Example:**
```bash
curl -o pz-42.16.zip \
  -H "Range: bytes=0-1024" \
  http://registry.pzlauncher.io/api/v1/registry/versions/pz/42.16/windows-x64
```

---

### POST `/api/v1/registry/versions/{game}/{version}/{platform}/submit`

Submit a hash for cross-validation (no file transfer).

**Parameters:**
- `game` (path): Game ID
- `version` (path): Game version
- `platform` (path): Platform

**Request Body:**
```json
{
  "sha256": "abc123def456...",
  "sizeBytes": 3200000000
}
```

**Response:**
```json
{
  "status": "upload_required",
  "trustLevel": "pending",
  "uploadCount": 1,
  "uploadUrl": "/api/v1/registry/versions/pz/42.16/windows-x64"
}
```

| Status | Meaning |
|--------|---------|
| `upload_required` | New hash; please upload the file via PUT |
| `counted` | Hash already uploaded by another subnet; no file needed |
| `already_counted` | Your subnet already submitted this hash |
| `conflict` | Different SHA256 exists for this version; flagged for review |

**Response Status:**
- `200 OK` — Hash submitted
- `409 Conflict` — Hash conflict detected

---

### PUT `/api/v1/registry/versions/{game}/{version}/{platform}`

Upload the binary file.

**Parameters:**
- `game` (path): Game ID
- `version` (path): Game version
- `platform` (path): Platform

**Headers:**
- `X-Content-SHA256` (required): SHA256 hash of the file being uploaded

**Request Body:**
- Binary file (multipart or raw)

**Response:**
```json
{
  "status": "ok"
}
```

**Status:**
- `200 OK` — File already in registry
- `201 Created` — File uploaded and stored
- `400 Bad Request` — Missing X-Content-SHA256 or checksum mismatch
- `409 Conflict` — SHA256 does not match registered hash
- `503 Service Unavailable` — Blob store not configured

**Example:**
```bash
curl -X PUT \
  -H "X-Content-SHA256: abc123..." \
  --data-binary @pz-42.16.zip \
  http://registry.pzlauncher.io/api/v1/registry/versions/pz/42.16/windows-x64
```

---

## Trust Model

| Level | Definition | Downloads | Use case |
|-------|-----------|-----------|----------|
| **Verified** | 3+ independent sources with matching SHA256 | Yes | Safe for production |
| **Pending** | 1-2 sources, waiting for more confirmations | Allowed | Community validation in progress |
| **Rejected** | SHA256 mismatch between sources | No | Admin review required |

---

## Error Responses

All errors return JSON:

```json
{
  "error": {
    "code": "REGISTRY_VERSION_NOT_FOUND",
    "message": "no binary registered for pz/42.16/windows-x64"
  }
}
```

**Common error codes:**
- `REGISTRY_VERSION_NOT_FOUND` — Version doesn't exist
- `REGISTRY_BLOB_NOT_FOUND` — Version registered but blob missing
- `REGISTRY_HASH_CONFLICT` — Hash mismatch detected
- `REGISTRY_SUBMIT_INVALID` — Invalid submission parameters
- `REGISTRY_UPLOAD_ERROR` — Upload failed

---

## Rate Limiting

Downloads: None (public)
Submissions: 1 per subnet per version per day
Uploads: 1 per file per day

---

## Example Workflow

**Community member uploads PZ v42.16:**

```bash
# Step 1: Compute SHA256 locally
SHA256=$(sha256sum pz-42.16.zip | cut -d' ' -f1)

# Step 2: Submit hash
RESPONSE=$(curl -X POST \
  -H "Content-Type: application/json" \
  -d "{\"sha256\":\"$SHA256\",\"sizeBytes\":3200000000}" \
  http://registry.pzlauncher.io/api/v1/registry/versions/pz/42.16/windows-x64/submit)

# Step 3: If upload_required, upload file
curl -X PUT \
  -H "X-Content-SHA256: $SHA256" \
  --data-binary @pz-42.16.zip \
  http://registry.pzlauncher.io/api/v1/registry/versions/pz/42.16/windows-x64

# Step 4: Done! Registry now has v42.16
```

---

## Metrics (Prometheus)

- `pz_registry_submit_total` — Hash submissions by game/version/status
- `pz_registry_upload_total` — Binary uploads by game/version/platform
- `pz_registry_upload_bytes_total` — Upload volume in bytes
- `pz_registry_download_total` — Binary downloads by game/version/platform
