# RFC-0036: Launcher Settings

**Status**: Active — Product / Updated for v2.0.0 — see [RFC-0050](0050-v2-architecture-rebaseline.md)  

> **v2.0.0 delta**: `steamcmdPath` removed; `backendUrl` added as the single Backend API base URL.  
**Schema**: [settings.schema.json](../../shared/contracts/settings.schema.json)  
**Go**: `libs/settings`  
**Persistence**: `<workspace>/config/launcher-settings.json`

---

## Problem

Join and launch need stable paths for game binary, cache, and profiles. UI and pipeline must read the same structure.

---

## Contract

```json
{
  "gamePath": "/path/to/ProjectZomboid",
  "backendUrl": "https://api.pzlauncher.example.com",
  "cachePath": "/path/to/cache",
  "profilesPath": "/path/to/profiles",
  "concurrentDownloads": 3,
  "bandwidthLimitMbps": 0,
  "verifyChecksum": true
}
```

| Field | Purpose |
|-------|---------|
| `gamePath` | PZ install — `PZ_PATH` env override on launch |
| `backendUrl` | Backend API base URL — single control plane endpoint |
| `cachePath` | Content cache root (`sha256/` underneath) |
| `profilesPath` | Per-server profile roots |
| `concurrentDownloads` | Download queue parallelism |
| `verifyChecksum` | SHA256 verify after download |

`steamcmdPath` removed in v2.0.0 — SteamCMD is a Backend concern.

---

## Integration

- `libs/pipeline.Config` built from `settings.Load(workspaceRoot)`
- Wails `GetSettings` / `SaveSettings` use `libs/settings`
- Dev API `GET/PUT /api/settings`
- UI `SettingsPanel` binds to `LauncherSettings`

---

## Non-goals

- Cloud sync of settings
- Per-server settings overrides (v2)
