# RFC-0052: Backend Core API

**Status**: Active — v2.0.0 Phase A  
**Depends on**: [RFC-0050](0050-v2-architecture-rebaseline.md), [RFC-0051](0051-v2-phase-plan.md)  
**Feeds**: RFC-0053, RFC-0055

---

## Purpose

Define the canonical Backend API surface that the Launcher and Agents communicate with. This is the contract boundary between the Platform and its clients.

All endpoints are versioned under `/api/v1/`.

---

## Authentication

- Launcher authenticates with a bearer token issued during first-run (one-time installation token flow — RFC-0053)
- Agents authenticate with an enrollment token (RFC-0053)
- All authenticated endpoints require `Authorization: Bearer <token>` header
- Unauthenticated endpoints: `GET /api/v1/servers`, `GET /api/v1/health`

---

## Endpoints

### Server Registry

```http
GET /api/v1/servers
```

Returns the list of registered servers.

Response:

```json
{
  "servers": [
    {
      "id": "demo-survival",
      "name": "Demo Survival",
      "region": "eu-west",
      "gameVersion": "42.8",
      "playerCount": 12,
      "maxPlayers": 32,
      "status": "online",
      "tags": ["survival", "mods"],
      "lastSeen": "2026-06-05T14:00:00Z"
    }
  ]
}
```

```http
GET /api/v1/servers/{serverId}
```

Returns metadata for a single server.

---

### Join API

```http
POST /api/v1/join/{serverId}
```

The central Launcher entry point. Backend resolves the current manifest, checks content availability, and returns a complete download plan.

Request headers: `Authorization: Bearer <token>`

Response: see [RFC-0055](0055-join-response-contract.md)

Error codes:

| Code | HTTP | Meaning |
|------|------|---------|
| `JOIN_SERVER_NOT_FOUND` | 404 | Unknown serverId |
| `JOIN_SERVER_OFFLINE` | 409 | Server not currently healthy |
| `JOIN_MANIFEST_UNAVAILABLE` | 503 | No valid manifest from Agent |
| `JOIN_CONTENT_UNAVAILABLE` | 503 | Content not resolvable (Agent + storage + SteamCMD all failed) |
| `JOIN_AUTH_REQUIRED` | 401 | Missing or invalid token |

---

### Download API

```http
GET /api/v1/download/{sha256}
```

Returns the content blob for a given SHA256 hash. May redirect to a signed URL (S3/CDN) or stream directly.

- Response body: raw binary blob
- `Content-Length` header: byte size
- `X-SHA256` header: echoes the hash for client verification
- Supports `Range` requests for resume

```http
GET /api/v1/download/{sha256}/meta
```

Returns metadata only (no blob):

```json
{
  "sha256": "...",
  "sizeBytes": 123456,
  "cachedAt": "2026-06-01T00:00:00Z",
  "source": "agent"
}
```

---

### Agent Endpoints (Backend-internal; not called by Launcher)

```http
POST /api/v1/agents/register
```

Registers a new Agent with a one-time enrollment token.

Request:

```json
{
  "enrollmentToken": "...",
  "serverId": "demo-survival",
  "agentVersion": "0.1.0",
  "platform": "linux/amd64"
}
```

Response:

```json
{
  "agentId": "...",
  "accessToken": "...",
  "expiresAt": "2027-06-05T00:00:00Z"
}
```

```http
POST /api/v1/agents/heartbeat
```

Agent reports current server status. Authenticated with agent access token.

Request:

```json
{
  "serverId": "demo-survival",
  "status": "online",
  "playerCount": 12,
  "maxPlayers": 32,
  "gameVersion": "42.8",
  "timestamp": "2026-06-05T14:00:00Z"
}
```

```http
PUT /api/v1/agents/manifest
```

Agent submits the current server manifest to the Backend.

Request body: `ServerManifest` JSON (RFC-0030 schema).

Response: `{ "manifestId": "...", "version": "91" }`

---

### Installation Token

```http
POST /api/v1/tokens
```

Issues a one-time installation token for a new Launcher client (first-run flow).

Request: admin-authenticated or invite-flow (implementation detail of Backend).

Response:

```json
{
  "token": "...",
  "expiresAt": "2026-06-06T00:00:00Z",
  "singleUse": true
}
```

---

### Health

```http
GET /api/v1/health
```

Unauthenticated. Returns Backend operational status.

```json
{ "status": "ok", "version": "2.0.0" }
```

---

## Versioning

- All endpoints are under `/api/v1/`
- Breaking changes increment the version prefix: `/api/v2/`
- Launcher and Agent pin to a specific version prefix
- Backend may support multiple version prefixes simultaneously during transition

---

## Session IDs

- `POST /join` returns a `sessionId` (UUID)
- Launcher includes `X-Session-ID: <sessionId>` on subsequent download requests for tracing
- Session IDs are Backend-generated and opaque to the Launcher

---

## Error envelope

All error responses:

```json
{
  "error": {
    "code": "JOIN_SERVER_NOT_FOUND",
    "message": "Server 'xyz' not found in registry",
    "requestId": "..."
  }
}
```
