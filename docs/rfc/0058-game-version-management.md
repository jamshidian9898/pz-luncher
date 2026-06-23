# RFC-0058: Game Version Management

**Status**: Active — v2.1.0  
**Depends on**: [RFC-0033](0033-installation-pipeline.md), [RFC-0059](0059-content-registry.md)  
**Feeds**: RFC-0061

---

## Problem

PZ server hosts lock their server to a specific game version (e.g. v42.16) for stability — to prevent character loss, item corruption, and RP continuity. Players whose client is on a different version (e.g. v42.20) cannot connect.

Steam auto-updates the game. In Iran and similar restricted regions, Steam is not reachable at all. Players need the correct client version without relying on Steam.

---

## Goal

The Launcher manages multiple PZ client versions side-by-side on the player's PC. When joining a server that requires v42.16, the Launcher uses v42.16 — regardless of what version the player originally installed.

---

## Version detection

On join, the Launcher reads the required version from the server manifest:

```json
{
  "gameVersion": "42.16",
  "buildId": "14058291"
}
```

It then checks the local version cache:

```
~/.pz-launcher/versions/
  42.16/   ← exists → use it
  42.20/   ← exists → use it
```

If the required version is not cached → trigger download flow (see RFC-0059).

---

## Local version store

Each version is stored in its own directory, isolated from others:

```
~/.pz-launcher/versions/
  {gameVersion}/
    ProjectZomboid64.exe   (Windows)
    ProjectZomboid64       (Linux)
    ProjectZomboid.app/    (macOS)
    zombie/
    media/
    .version               ← { "gameVersion": "42.16", "sha256": "...", "verifiedAt": "..." }
```

Versions are **never modified** after download. They are read-only. A profile symlinks into this directory.

---

## Version integrity

Every version in the local store has a `.version` file containing:

```json
{
  "gameVersion": "42.16",
  "buildId": "14058291",
  "sha256": "abc123...",
  "source": "registry",
  "verifiedAt": "2026-06-23T10:00:00Z",
  "trustLevel": "verified"
}
```

On first use after download, the Launcher recomputes SHA256 of the main executable and compares against the stored value. Mismatch → version is quarantined, user is notified.

---

## Download flow

```
Required: v42.16
Local cache: miss

→ Query registry for available sources (RFC-0059):
    Source A: Registry (verified ✅)  — 3.2 GB
    Source B: Agent URL (server host) — 3.2 GB
    Source C: Hoster upload (⚠️ unverified) — 3.2 GB

→ Show user source selector:
  ┌──────────────────────────────────────────┐
  │  دانلود v42.16                           │
  │                                          │
  │  ● Registry (تأییدشده) ✅    3.2 GB     │
  │  ● Agent سرور            ✅    3.2 GB     │
  │  ● آپلود hoster           ⚠️   3.2 GB     │
  └──────────────────────────────────────────┘

→ Download from selected source
→ Verify SHA256 after download
→ Store in ~/.pz-launcher/versions/42.16/
```

---

## Cache eviction

Versions are never auto-deleted. The Launcher suggests deletion when:

1. A version has not been used for N days (default: 30, configurable)
2. The user opens the Cache Manager UI

Eviction rules:

| Condition | Action |
|-----------|--------|
| Not used for 30+ days, no active server uses it | Suggest deletion |
| Not used for 30+ days, but a saved server uses it | Warn before delete |
| Currently in use (game running) | Block deletion |

The user always confirms before any version is deleted.

---

## Version catalog

The Launcher maintains a local catalog of known versions, synced from the registry:

```json
{
  "versions": [
    {
      "gameVersion": "42.16",
      "buildId": "14058291",
      "releasedAt": "2025-11-03",
      "trustLevel": "verified",
      "sourcesAvailable": ["registry", "agent"]
    },
    {
      "gameVersion": "42.20",
      "buildId": "14201033",
      "releasedAt": "2026-01-15",
      "trustLevel": "verified",
      "sourcesAvailable": ["registry"]
    }
  ]
}
```

Catalog is refreshed on Launcher startup. Works offline with last-known catalog.

---

## Multi-version coexistence

A player can be a member of two servers requiring different versions:

```
profiles/
  survival-rp/          → uses versions/42.16/
  build-server/         → uses versions/42.20/
```

Switching servers never breaks either profile. The Launcher passes the correct `-Xmx`, `-gamedir`, and mod load order for each version independently.

---

## Out of scope

- Steam credential handling (not required — registry is the source)
- Auto-update of game (user controls versions explicitly)
- Game version for non-PZ games (future RFC)
