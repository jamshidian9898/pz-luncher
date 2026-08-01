# Progress Report

**Phase**: Phase A/B/C complete — v2.1.0 line (Game Version Management + Content Registry)
**Decision**: [PRODUCT_DECISION.md](../PRODUCT_DECISION.md)
**RFC index**: [DOMAIN_RFC_INDEX.md](DOMAIN_RFC_INDEX.md)

---

## Summary

| Area | Status | Notes |
|------|--------|-------|
| Documentation (foundation) | ✅ | RFCs 0001–0021, domain, contracts |
| UI infrastructure | ✅ | RFC-0024/0025 implemented in `launcher-ui` |
| Go platform core | ✅ | Session, Steam, chaos/shadow/campaign |
| Domain RFCs 0030–0036 | ✅ | Manifest → resolve → download → profile → launch → settings |
| Backend + Agent (0050–0056) | ✅ | `apps/backend`, `apps/pz-agent`, Bearer-token enrollment |
| Admin UI (0057) | 🟡 Basic | `apps/admin-ui` exists, functional but minimal |
| Game version mgmt + registry (0058–0062) | ✅ | Content Registry, community upload, cache manager, version selection |
| Launcher UI (product) | ✅ | Full join/download/launch flow, translated to English |
| Cloud microservices | ⏸️ | Not needed — backend is the single service |

---

## Verified state (2026-08-01)

- `go build ./...` — clean, no errors
- `go test ./...` — all packages with tests pass (`backend/internal/content`, `launcher-core`, `launcher-ui`, `manifestv1`, `modplan`)
- `apps/launcher-ui/frontend`: `npm run build` (tsc + vite) — clean
- Git: `main` branch, working tree clean as of commit `b14cef2`

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
| 0036 Settings | ✅ | ✅ `libs/settings` | — |
| 0050–0055 Backend/Agent/Join | ✅ | ✅ `apps/backend`, `apps/pz-agent` | partial |
| 0057 Admin UI | 🟡 | 🟡 `apps/admin-ui` (basic) | — |
| 0058 Game Version Management | ✅ | ✅ `apps/launcher-ui` (versionselect.go) | ✅ |
| 0059 Content Registry | ✅ | ✅ `apps/backend/internal/registry` | ✅ |
| 0060 Community Upload | ✅ | ✅ `contribute.go` | — |
| 0061 Cache Manager | ✅ | ✅ `cache.go` | ✅ |
| 0062 Game Version Selection | ✅ | ✅ `versionselect.go` | ✅ |

Fixture: [fixtures/manifests/demo-survival.json](../fixtures/manifests/demo-survival.json)

---

## What works today

- **Full v2 loop (E2E validated 2026-07-03)**: agent scans a real mods dir →
  pushes deterministic-zip blobs + versioned manifest to backend → launcher
  `POST /join` downloads, verifies, and extracts a byte-identical profile
- **launcher-core**: offline resolve → profile → launch (demo)
- **libs/session**: download execution, Steam, validation CLIs
- **launcher-ui**: server list with status badges, downloads panel (Steam-style queue), settings, game process tracking (Stop button, auto-detect exit), Cache Manager, Community Upload, Game Version Selection — UI fully in English
- **backend**: Content Registry API, agent enrollment (Bearer tokens), join pipeline, Prometheus metrics
- **Wails**: bindings for join, settings, cache, contribute, version select (mock + real adapters)

---

## Try the join pipeline (CLI)

```bash
# offline fixture path
go run ./apps/join-cli -server=demo-survival
go run ./apps/join-cli -server=demo-survival -launch

# live backend path (same pipeline as the UI)
go run ./apps/backend/cmd/backend &
go run ./apps/pz-agent/cmd/agent -server=my-server -mods=/path/to/mods -interval=0
go run ./apps/join-cli -server=my-server -backend=http://localhost:8080
```

---

## Real-environment test labs (single-machine E2E, real PZ dedicated server)

Three ready-made labs prove the full loop (agent scans mods → deterministic-zip
blobs + manifest → backend → `POST /join` → byte-identical extracted profile →
game launched with `-cachedir=<profile>`) against a **real** PZ Dedicated
Server (steamcmd app 380870), not a fixture:

| Lab | Platform | Entry point |
|-----|----------|-------------|
| Windows | Single Windows 10/11 machine, native services | [deploy/windows/lab.ps1](../deploy/windows/lab.ps1) |
| Linux/WSL | Single Ubuntu/WSL machine | [deploy/linux/lab.sh](../deploy/linux/lab.sh) |
| VMs | Two Ubuntu VMs (QEMU/KVM, works inside WSL2) | [deploy/vms/vms.sh](../deploy/vms/vms.sh) |

Each supports `install` (skippable game-server download with
`-SkipGameServers`/`--skip-game-servers`), `up`, `verify` (exit 0 = PASS,
asserts backend health, agent registration/heartbeat, manifest publish, join,
launch args, and byte-identical mod content), `status`, `down`, `clean`.

## Recent activity (last commits on `main`)

- Two-VM game-server lab (QEMU/KVM + Ansible) with real PZ servers
- Linux/WSL and single-Windows test labs with automated servers, agents, E2E verify
- Real profile isolation via `-cachedir`; native Windows agent service mode
- Deterministic zip content identity for directory mods (byte-identical E2E loop)
- Display server names in downloads panel; allow launching any ready server
- Improve contribute version detection, directory hashing/upload
- Full UI translation from Persian to English
- Expanded test coverage; fixed Content Registry submission count bug

---

## Next

- [ ] Flesh out Admin UI (0057) beyond basic screens
- [ ] Run a real-environment test pass on actual hardware using the labs above
- [ ] Error strings in UI for pipeline codes

---

## Not doing (until further notice)

- New infrastructure RFCs (0026+)
- Plugin system, multi-game, analytics platform
- `directory-service`, `registry-service`, `manifest-service` as separate deployables
- Hybrid phase enforcement setup

---

## Platform validation (optional, parallel)

Go campaign/SLO work in [STOP.md](../STOP.md) can continue in background; product path does not block on 1000-run campaign.

---

## Agent (scaffolding, legacy)

- `.agent.md` — scaffolder from `ProjectBaseDocs`
- New features: follow [DOMAIN_RFC_INDEX.md](DOMAIN_RFC_INDEX.md)

---

*Last updated: 2026-08-01 — verified against actual build/test run and git log, not just RFC docs*
