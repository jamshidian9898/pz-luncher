# Progress Report

**Phase**: 1 — Product Execution  
**Decision**: [PRODUCT_DECISION.md](../PRODUCT_DECISION.md)  
**Roadmap**: [PRODUCT_ROADMAP.md](PRODUCT_ROADMAP.md)

---

## Summary

| Area | Status | Notes |
|------|--------|-------|
| Documentation (foundation) | ✅ | RFCs 0001–0021, domain, contracts |
| UI infrastructure | ✅ | RFC-0024/0025 implemented in `launcher-ui` |
| Go platform core | ✅ | Session, Steam, chaos/shadow/campaign |
| Domain RFCs 0030–0035 | ✅ Written | Implementation in progress |
| Domain implementation | 🟡 | Week 1: RFC-0030 |
| Launcher UI (product) | 🟡 | Shell + stores; join flow partial |
| Cloud microservices | ⏸️ | After MVP |

---

## Domain RFC implementation

| RFC | Spec | Code | Tests |
|-----|------|------|-------|
| 0030 Server Manifest v1 | ✅ | ✅ `libs/manifestv1` | ✅ |
| 0031 Mod Resolver | ✅ | ✅ `libs/modplan` | ✅ |
| 0032 Download Manager | ✅ | ✅ `libs/download` | — |
| 0033 Installation Pipeline | ✅ | ✅ `libs/pipeline` | CLI |
| 0034 Profile System | ✅ | ✅ via `profile` + snapshot | — |
| 0035 Launch Flow | ✅ | ✅ `pipeline.Launch` + UI | — |

Fixture: [fixtures/manifests/demo-survival.json](../fixtures/manifests/demo-survival.json)

---

## What works today

- **launcher-core**: offline resolve → profile → launch (demo)
- **libs/session**: download execution, Steam, validation CLIs
- **launcher-ui**: server list, downloads panel, settings, trace viewer, event system
- **Wails**: bindings for join, settings (mock + real adapters)

---

## Try the join pipeline (CLI)

```bash
go run ./apps/join-cli -server=demo-survival
go run ./apps/join-cli -server=demo-survival -launch
```

## Next (UI polish)

- [ ] TypeScript types aligned with RFC-0030 (`types.ts`)
- [ ] Load server list from registry in dev without Wails
- [ ] Error strings in UI for pipeline codes

---

## Not doing (until MVP ships)

- New infrastructure RFCs (0026+)
- Plugin system, multi-game, analytics platform
- `directory-service`, `registry-service`, `manifest-service` as separate deployables
- Hybrid phase enforcement setup

---

## Platform validation (optional, parallel)

Go campaign/SLO work in [STOP.md](../STOP.md) can continue in background; **product path does not block on 1000-run campaign**.

---

## Agent

- `.agent.md` — scaffolder from `ProjectBaseDocs`
- New features: follow [DOMAIN_RFC_INDEX.md](DOMAIN_RFC_INDEX.md)

---

*Last updated: 2026-06-04 — aligned with PRODUCT_DECISION*
