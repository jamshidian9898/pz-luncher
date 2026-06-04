# RFC-0035: Game Launch Flow

**Status**: Active — Phase 1 Product  
**Depends on**: RFC-0034  
**Aligns with**: [RFC-0016](0016-game-launcher.md), [RFC-0021](0021-launch-state-machine.md), `libs/game`

---

## Problem

After profile is ready, the launcher must **start Project Zomboid**, track process lifecycle, and clean up on exit—without leaving zombie state in the UI.

---

## State machine (product layer)

```text
Ready
  ↓  user clicks Launch (or auto after join — product choice)
Launching
  ↓  process started
Running
  ↓  process exit
Exited
  ↓  optional cleanup
Cleanup → Ready (idle for this server)
```

### Mapping to launcher-core (RFC-0021)

| Product state | Core state |
|---------------|------------|
| Ready | `Idle` / post-`BuildingProfile` |
| Launching | `Launching` |
| Running | `Running` |
| Exited / Cleanup | `Stopped` |
| Any fatal | `Failed` |

Join pipeline ends in **Ready**; launch is a **separate** user action in MVP (recommended for clearer UX). Auto-launch may be a setting in v0.2.

---

## Launch request

```ts
export interface LaunchRequest {
  serverId: string;
  profileId: string;
  gameExecutable: string;      // from settings or auto-detect
  launchArgs: string[];        // manifest + user overrides
  workingDirectory?: string;
  env?: Record<string, string>;
}

export interface LaunchResult {
  pid?: number;
  startedAt: string;
  exitCode?: number;
  error?: string;
}
```

Go: `libs/game/launcher.go` — extend existing `Launch()`.

---

## Arguments

Combine in order:

1. Platform base args (OS-specific)
2. `ServerManifest.launchArgs`
3. Mod classpath / `-mod=` flags per PZ conventions (document in `docs/domain/` when fixed)
4. Profile paths injected via env or cfg

**MVP**: use same args as current `launcher-core` demo + manifest `launchArgs`.

---

## Process monitoring

- Poll or wait on `cmd.Process` (Go)
- Emit:
  - `launch.started` — `{ pid, serverId }`
  - `launch.exited` — `{ exitCode }`
  - `launch.failed` — `{ code, message }`

UI session store: add `launching` | `running` states or extend `SessionStatus`.

---

## Cleanup

On `Exited`:

- Flush trace buffers
- Set UI to Ready
- Do **not** delete profile or cache
- Optional: kill child processes if hung (timeout 30s after window close signal)

---

## Error codes

| Code | Meaning |
|------|---------|
| `LAUNCH_EXE_NOT_FOUND` | Game path invalid |
| `LAUNCH_ALREADY_RUNNING` | Block second instance (configurable) |
| `LAUNCH_PROFILE_NOT_READY` | Pipeline not at Ready |
| `LAUNCH_PROCESS_FAILED` | Start error |

---

## Week 6 exit criteria

- [ ] Launch starts PZ (or stub binary in CI) from profile
- [ ] Exit returns UI to Ready
- [ ] `LAUNCH_EXE_NOT_FOUND` shown in settings-driven path test

---

## Week 7–8 (UI)

- Launch button on server detail when `Ready`
- Running indicator in sidebar
- Error strings localized / user-friendly (Persian optional later)
