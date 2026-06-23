# RFC-0062: Game Version Selection

**Status**: Active — v2.1.0  
**Depends on**: [RFC-0058](0058-game-version-management.md), [RFC-0059](0059-content-registry.md), [RFC-0061](0061-cache-manager.md)

---

## Purpose

When a player joins a server, the Launcher automatically detects the required game version from the server manifest and downloads it if not present. The player never manually selects a version — the Launcher handles it transparently.

---

## Version selection flow

```
Player clicks "Join" on server
        ↓
Backend returns JoinResponse with gameVersion: "42.16"
        ↓
Launcher checks: do we have v42.16 locally?
        ↓
    YES → Use it, skip download
    NO  → Show version selector (if multiple sources available)
        ↓
Download from selected source (Registry / Agent)
        ↓
Verify SHA256 after download
        ↓
Profile setup uses correct version
```

---

## Version availability detection

The Launcher queries the registry to discover available versions:

```http
GET /api/v1/registry/catalog?game=pz

Response:
{
  "versions": [
    { "gameVersion": "42.16", "platform": "windows-x64", "trustLevel": "verified", ... },
    { "gameVersion": "42.20", "platform": "windows-x64", "trustLevel": "verified", ... }
  ]
}
```

---

## Download source selector UI

If a version is not locally cached and multiple sources are available, show user a selector:

```
┌──────────────────────────────────────────────────┐
│  دانلود v42.16                                  │
│                                                  │
│  منبع دانلود را انتخاب کن:                       │
│                                                  │
│  ● Registry (PZ Launcher)     3.2 GB  ✅ بهترین  │
│  ● Agent (server.example.com) 3.2 GB  ✅ سریع   │
│  ● Hoster Upload              3.2 GB  ⚠️ تأیید نشده │
│                                                  │
│  [دانلود]                                       │
└──────────────────────────────────────────────────┘
```

Selection priority (auto-select first available):
1. Registry (verified, cached locally, fastest)
2. Agent (server-direct, no extra bandwidth to us)
3. Hoster-uploaded (unverified warning)

---

## No manual version selection in MVP

The Launcher does **not** offer a "choose any version you want" UI in v2.0.0. The version is locked by the server manifest. This is intentional:

- **Why**: Prevents players from joining wrong version, breaking gameplay
- **Future**: Post-MVP, admin API can let servers unlock older versions for backward compat

---

## Out of scope

- Multi-game version management (single PZ for MVP)
- Automatic game updates (player controls via version management UI)
- Version downgrade warnings (all versions treated equally)
