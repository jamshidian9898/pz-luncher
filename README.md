# Project Zomboid Launcher

> **v2.0.0 Architecture** — [RFC-0050](docs/rfc/0050-v2-architecture-rebaseline.md) is the canonical spec.  
> Launcher communicates exclusively with the Backend. Agents and SteamCMD are Backend-internal infrastructure.

> **Phase 1: Product Execution** — [PRODUCT_DECISION.md](PRODUCT_DECISION.md)  
> Build a **player-ready launcher**. Domain RFCs **0030–0035**, not new infrastructure RFCs.  
> Roadmap: [docs/PRODUCT_ROADMAP.md](docs/PRODUCT_ROADMAP.md)

---

## Current focus

| Layer | Status |
|-------|--------|
| UI infrastructure (RFC-0024/0025) | ✅ Done — use, don't extend |
| Go session / download core | ✅ Usable — wire to product flow |
| Domain (manifest → launch) | 🟡 **Active** — RFC-0030 → 0035 |
| Cloud microservices | ⏸️ After MVP |

**Start here**: [docs/rfc/0030-server-manifest-v1.md](docs/rfc/0030-server-manifest-v1.md)

---

This repository is a monorepo for a universal Project Zomboid launcher and mod ecosystem.

The goal is to support:
- Server discovery and one-click join
- Automatic mod management and versioned manifests
- Profile isolation and rollback support
- Smart caching, delta updates, and content distribution

Scaffolded structure includes:
- `apps/` for launcher UI, API services, and agent runtime
- `libs/` for shared manifest, package, profile, cache, downloader, and contract libraries
- `docs/` for architecture, API contracts, and RFCs

- [docs/DOMAIN_RFC_INDEX.md](docs/DOMAIN_RFC_INDEX.md) — active specs  
- [STATUS.md](STATUS.md) — Go platform status  
- `.agent.md` — scaffolding agent (legacy; prefer domain RFCs for new work)