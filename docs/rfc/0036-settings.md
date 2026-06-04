# RFC-0036: Launcher Settings

**Status**: Active — Product  
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
  "steamcmdPath": "",
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
| `cachePath` | Content cache root (`sha256/` underneath) |
| `profilesPath` | Per-server profile roots |
| `concurrentDownloads` | Download queue parallelism |
| `verifyChecksum` | SHA256 verify after download |

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
