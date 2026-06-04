# RFC-0034: Profile System

**Status**: Active — Phase 1 Product  
**Depends on**: RFC-0033  
**Extends**: [profile-system.md](../profile-system.md), [RFC-0007](0007-profile-isolation.md), [RFC-0017](0017-save-isolation.md)

---

## Problem

Each server must have **isolated** mods, saves, and config so switching servers never corrupts another world's files.

---

## Goals

- One profile root per server (`profileId` default = `serverId`)
- Materialize mods from content cache into profile
- Launcher never writes to game's global mod folder in MVP
- Rollback = re-activate previous manifest version folder (optional v1.1)

---

## Directory layout

```text
<profilesLocation>/
  <profileId>/
    manifest-<version>.json     # copy of active manifest
    mods/
      <modId>/                  # symlink or copy from cache
    saves/                      # game saves for this server
    config/
      launch.ini                # generated launch hints
    join-trace-<sessionId>.json
    provider-trace.json         # existing Go trace
```

Content cache (shared across servers):

```text
<cacheLocation>/
  sha256/<hash>                 # content-addressable blobs
  downloads/<sessionId>/        # temp partials (RFC-0032)
```

---

## TypeScript

```ts
export interface ProfileConfig {
  profileId?: string;
  modLoadOrder?: string[];
  env?: Record<string, string>;
}

export interface ActiveProfile {
  profileId: string;
  serverId: string;
  manifestVersion: string;
  rootPath: string;
  modPaths: string[];           // ordered for launch
  savesPath: string;
  configPath: string;
}
```

From RFC-0030 `ServerManifest.profile`.

---

## Activation (after install)

1. Ensure `profiles/<profileId>/` exists
2. For each mod in `ResolvedModPlan.orderedMods`:
   - Link `cache/sha256/<hash>` → `mods/<modId>/`
   - Use symlinks on Unix; copy on Windows if symlink fails
3. Verify size + sha256 at link target
4. Write `config/launch.ini` with load order + `launchArgs`
5. Copy manifest snapshot to `manifest-<version>.json`

Reuse: `libs/profile/builder.go` (`ProfileBuilder`).

---

## Settings (launcher-ui)

From existing `Settings`:

- `profilesLocation`
- `cacheLocation`
- `steamcmdPath` (for workshop downloads)

---

## Invariants

| Rule | |
|------|--|
| No cross-profile symlinks into `saves/` | |
| Profile mod dir only contains mods from active manifest version | |
| Deleting profile must not delete shared cache blobs | |

---

## Events

- `profile.activated` — `{ profileId, modCount }`
- `profile.failed` — `{ code, message }`

---

## Week 5 exit criteria

- [ ] Two server profiles on disk with no shared mod paths
- [ ] Re-join updates only changed mods
- [ ] Settings paths respected on Windows + macOS
