# Project Zomboid Launcher

This repository is a monorepo scaffold for a universal Project Zomboid launcher and mod ecosystem.

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