# Project Zomboid Launcher

> **🛑 CRITICAL DECISION POINT**  
> **See**: `ARCHITECTURAL_DECISION.md` — System has evolved beyond launcher scope  
> **Required**: Choose between Product Mode or Platform Mode before proceeding  
> **Status**: Feature development blocked until decision is made

---

## Current Architecture Status

This repository has evolved from a simple launcher into an **event-sourced runtime engine** with production-grade optimization:

- ✅ **RFC-0024**: Event Log + Replay System (audit trail, deterministic reconstruction)
- ✅ **RFC-0025**: Snapshot + Compaction Engine (10x reconstruction speedup, memory governance)
- ✅ **RFCs 0001-0023**: Foundation systems (manifests, profiles, contracts, agents)

**The Question**: Is this infrastructure a **product** (launcher) or a **platform** (extensible system)?

See `ARCHITECTURAL_DECISION.md` for:
- Two possible paths forward (Product vs Platform)
- Trade-off analysis and decision framework
- Implementation roadmaps for each path
- Recommendation: Hybrid approach (launcher first, platform second)

---

This repository is a monorepo for a universal Project Zomboid launcher and mod ecosystem.  
**Current status**: Architecture validation complete. Awaiting strategic direction.

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