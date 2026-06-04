# Project Zomboid Launcher

> **🛑 STOP — Platform v1.0 Architecturally Complete**  
> **Status**: Pending validation (see `STOP.md`)  
> **Next**: Execute extended campaign, prove SLOs  
> **See**: `FINAL_SUMMARY.md` for complete status

---

This repository is a monorepo for a universal Project Zomboid launcher and mod ecosystem.
**Current focus**: Execution platform validation, not feature development.

The goal is to support:
- Server discovery and one-click join
- Automatic mod management and versioned manifests
- Profile isolation and rollback support
- Smart caching, delta updates, and content distribution

Scaffolded structure includes:
- `apps/` for launcher UI, API services, and agent runtime
- `libs/` for shared manifest, package, profile, cache, downloader, and contract libraries
- `docs/` for architecture, API contracts, and RFCs

Use `.agent.md` as the scaffolding agent when you want to generate initial project files and architecture docs.