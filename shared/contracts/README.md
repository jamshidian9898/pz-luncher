# Shared contracts

**Single source of truth** for Go and TypeScript.

| Schema | Purpose |
|--------|---------|
| [manifest.schema.json](manifest.schema.json) | RFC-0030 ServerManifest |
| [launcher-events.schema.json](launcher-events.schema.json) | RFC-0022 pipeline/UI events |
| [settings.schema.json](settings.schema.json) | RFC-0036 LauncherSettings |

## Regenerate types

```bash
make contracts
# or
go run ./tools/gencontracts
```

Outputs:

- `libs/sharedtypes/types_gen.go`
- `apps/launcher-ui/frontend/src/contracts/generated.ts`
- `apps/launcher-ui/frontend/public/registry/` (synced from `examples/servers.json`)

## Rules

1. Edit **schema** first, then run `make contracts`.
2. Do not hand-edit `types_gen.go` or `generated.ts`.
3. `libs/manifestv1` re-exports manifest types from `sharedtypes`.
