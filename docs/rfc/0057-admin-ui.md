# RFC-0057: Admin UI

**Status**: Implemented (2026-08-25) — full scope complete  
**Depends on**: [RFC-0052](0052-backend-core-api.md), [RFC-0053](0053-agent-enrollment.md)  
**Code**: `apps/admin-ui/`

---

## Purpose

A web dashboard for Backend operators to monitor servers, agents, and content store health. Not used by players. Not required for MVP.

---

## Scope (post-MVP only)

```
✅ Server list + health status
✅ Per-server agent status (Healthy / Offline / Revoked)
✅ Content store browser (SHA256 blobs, size, source) — GET /api/v1/blobs
✅ Agent enrollment token generation — /api/v1/admin/tokens, guarded by -admin-token
✅ Manifest viewer per server — ServerDetail "Manifest" tab
✅ Download statistics — persisted per-blob counter, "DLs" column in Content page
```

---

## Current state

A skeleton exists at `apps/admin-ui/` (React 18 + TypeScript + Vite + TailwindCSS):

- `Dashboard.tsx` — server list, agent status cards
- `ServerDetail.tsx` — per-server detail view

The UI calls `GET /api/v1/servers` and `GET /api/v1/agents` from RFC-0052.  
No auth is wired. No write operations. Read-only monitoring only.

---

## Not in scope (MVP)

- Agent token issuance via UI (operator uses CLI or API directly)
- Player management
- Log streaming
- Deployment or server start/stop controls

---

## Tech stack

| Layer | Choice |
|-------|--------|
| Framework | React 18 + TypeScript |
| Build | Vite |
| Styling | TailwindCSS + shadcn/ui |
| Icons | lucide-react |
| Dev server | `npm run dev` (default port 5173) |

---

## Running locally

```bash
cd apps/admin-ui
npm install
npm run dev
```

Proxies API calls to `http://localhost:8080` by default.
