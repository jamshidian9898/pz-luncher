# Documentation

> **Active work**: [DOMAIN_RFC_INDEX.md](DOMAIN_RFC_INDEX.md) and [PRODUCT_ROADMAP.md](PRODUCT_ROADMAP.md)  
> **Decision**: [PRODUCT_DECISION.md](../PRODUCT_DECISION.md)

---

## Product (Phase 1 — build these)

| Doc | Purpose |
|-----|---------|
| [DOMAIN_RFC_INDEX.md](DOMAIN_RFC_INDEX.md) | Order and status of RFC 0030–0035 |
| [PRODUCT_ROADMAP.md](PRODUCT_ROADMAP.md) | 8-week execution plan |
| [rfc/0030-server-manifest-v1.md](rfc/0030-server-manifest-v1.md) | **Start here** |
| [rfc/0031-mod-dependency-resolver.md](rfc/0031-mod-dependency-resolver.md) | Dependency graph |
| [rfc/0032-download-manager.md](rfc/0032-download-manager.md) | Queue, retry, checksum |
| [rfc/0033-installation-pipeline.md](rfc/0033-installation-pipeline.md) | End-to-end join |
| [rfc/0034-profile-system.md](rfc/0034-profile-system.md) | Per-server isolation |
| [rfc/0035-game-launch-flow.md](rfc/0035-game-launch-flow.md) | Launch lifecycle |

---

## Vision & architecture (background)

- [vision.md](vision.md) — product goals
- [architecture.md](architecture.md) — monorepo layout
- [service-boundaries.md](service-boundaries.md) — services (mostly future)
- [api-contracts.md](api-contracts.md) — APIs
- [manifest-schema.md](manifest-schema.md) — legacy; see RFC-0030
- [profile-system.md](profile-system.md) — legacy; see RFC-0034
- [ROADMAP.md](ROADMAP.md) — original phased plan (superseded by PRODUCT_ROADMAP for execution)
- [progress.md](progress.md) — tracker

---

## Domain & contracts

- [domain/](domain/) — domain notes
- [contracts/](contracts/) — platform guarantees (Go)
- [architecture/execution-graph.md](architecture/execution-graph.md)

---

## RFC archive

- [rfc/](rfc/) — 0001–0025 foundation + infrastructure; **0030–0035** product

---

## Historical / optional

Strategic decision stack (Path A/B/C) — **decision made (Path A)**:

- `../ARCHITECTURAL_DECISION.md`, `../DECISION_GUIDE.md`, `../README_START_HERE.md`

Do not use for day-to-day execution unless revisiting strategy.
