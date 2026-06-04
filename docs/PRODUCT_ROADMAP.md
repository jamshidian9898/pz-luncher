# Product Roadmap — Phase 1 Execution (8 weeks)

**Status**: Active  
**Decision**: [PRODUCT_DECISION.md](../PRODUCT_DECISION.md)  
**Rule**: Domain RFCs + implementation. No infrastructure RFCs.

---

## MVP outcome

A player can:

1. Browse and select a server  
2. Join (resolve manifest → download → install)  
3. See progress and understandable errors  
4. Launch Project Zomboid with the correct mods and profile  
5. Switch servers without manual mod/save cleanup  

---

## Week-by-week plan

### Week 1 — RFC-0030: Server Manifest v1

- [ ] Finalize `ServerManifest` schema (JSON + TypeScript types)
- [ ] Validator + sample fixtures under `fixtures/manifests/`
- [ ] Wire manifest load in `launcher-core` (local URL or file)
- [ ] Map legacy [RFC-0001](rfc/0001-manifest-format.md) fields → v1

**Exit**: One real or demo manifest parses and validates end-to-end.

---

### Week 2 — RFC-0031: Mod Dependency Resolver

- [ ] Implement `ResolvedModPlan` from `mods[]`
- [ ] Cycle detection + version conflict errors (user-visible codes)
- [ ] Load order output for profile build
- [ ] Unit tests: diamond deps, cycle, missing dep

**Exit**: Resolver tests green; plan feeds download layer.

---

### Week 3 — RFC-0032: Download Manager

- [ ] Queue states: Pending → Downloading → Paused / Failed / Completed
- [ ] Retry policy + resume + checksum verify
- [ ] Connect to existing `libs/session` executor where possible
- [ ] UI: `DownloadPanel` shows per-mod progress

**Exit**: Single mod downloads with retry; queue visible in UI.

---

### Week 4 — RFC-0033: Installation Pipeline

- [ ] Pipeline: resolve → plan → download → verify → install → profile ready
- [ ] Idempotent re-run (skip completed steps)
- [ ] Trace file per join under `profiles/<serverId>/`

**Exit**: Full pipeline runs offline with fixtures.

---

### Week 5 — RFC-0034: Profile System

- [ ] Per-server dirs: `mods/`, `saves/`, `config/`
- [ ] Symlink/copy from content cache ([profile builder](../profile-system.md))
- [ ] Settings: `profilesLocation`, `cacheLocation`

**Exit**: Two servers = two isolated profiles on disk.

---

### Week 6 — RFC-0035: Game Launch Flow

- [ ] States: Ready → Launching → Running → Exited → Cleanup
- [ ] Map to [RFC-0021](rfc/0021-launch-state-machine.md) where aligned
- [ ] Launch game binary with `launchArgs` from manifest

**Exit**: Game process starts from launcher-core; exit detected.

---

### Week 7 — UI + Error UX

- [ ] Replace mock adapters with Wails bindings where missing
- [ ] Human-readable errors (manifest, resolver, download, launch)
- [ ] Settings panel: paths, concurrency
- [ ] Optional: hide debug trace behind dev flag

**Exit**: Non-developer can complete join flow from UI.

---

### Week 8 — MVP Release

- [ ] E2E test script / checklist
- [ ] Wails build (Windows primary; macOS if required)
- [ ] `docs/INSTALL.md` for players
- [ ] Tag `v0.1.0-mvp`

**Exit**: Binary handed to 2–3 beta users.

---

## Out of scope (this phase)

See [PRODUCT_DECISION.md](../PRODUCT_DECISION.md) — plugins, multi-game, cloud microservices, new event-system RFCs.

---

## Success metrics

| Metric | Target |
|--------|--------|
| Join success (demo server) | ≥ 95% |
| Time to ready (cached) | < 2 min |
| Time to ready (cold) | Depends on mod size; show ETA |
| Crash on launch | 0 in smoke test |

---

## Links

- [Domain RFC index](DOMAIN_RFC_INDEX.md)
- [Progress tracker](progress.md)
