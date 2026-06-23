# RFC-0061: Cache Manager

**Status**: Active — v2.1.0  
**Depends on**: [RFC-0058](0058-game-version-management.md), [RFC-0034](0034-profile-system.md)  

---

## Purpose

Define how the Launcher manages local disk usage across game versions, mods, and profiles. The player should always know what is stored on their PC and have full control over it — without needing to manually dig through folders.

---

## What the Launcher stores locally

```
~/.pz-launcher/
  versions/
    42.16/        ← game client binary (read-only after download)
    42.20/
  mods/
    {sha256}/     ← mod archives (shared across all servers)
  profiles/
    survival-rp/  ← symlinks into versions/ and mods/
    build-server/
  catalog.json    ← version catalog from registry (RFC-0059)
  settings.json   ← launcher settings (RFC-0036)
```

---

## Cache entries

### Game version entry

```json
{
  "gameVersion": "42.16",
  "platform": "windows-x64",
  "sizeBytes": 3200000000,
  "downloadedAt": "2026-05-01T10:00:00Z",
  "lastUsedAt": "2026-06-20T18:30:00Z",
  "usedByProfiles": ["survival-rp"],
  "sha256": "abc123...",
  "trustLevel": "verified"
}
```

### Mod entry

```json
{
  "sha256": "def456...",
  "modId": "workshop-2392709985",
  "name": "Britas Armor Pack",
  "sizeBytes": 45000000,
  "downloadedAt": "2026-05-10T12:00:00Z",
  "lastUsedAt": "2026-06-22T19:00:00Z",
  "usedByProfiles": ["survival-rp", "build-server"]
}
```

---

## Eviction policy

The Launcher **never deletes anything automatically**. It only suggests deletion.

```
Suggestion triggers:

Game version:
  → Not used for 30+ days (configurable in settings)
  → AND no active profile references it

Mod:
  → Not used for 60+ days (configurable)
  → AND no active profile references it

Never suggest:
  → Content used in the last N days
  → Content referenced by a profile the user joined in the last N days
  → Content currently being downloaded
  → Content while the game is running
```

---

## Cache Manager UI

Accessible from: Launcher → Settings → مدیریت فضا

```
┌──────────────────────────────────────────────────────┐
│  مدیریت فضا                                          │
│  کل: ۱۲.۴ GB  │  قابل حذف: ۳.۸ GB                  │
├──────────────────────────────────────────────────────┤
│  نسخه‌های بازی                           ۶.۴ GB      │
│                                                      │
│  ✅ v42.16 — ۳.۲ GB  │ آخرین استفاده: ۲ روز پیش    │
│     استفاده در: Survival RP                          │
│                                                      │
│  ⚠️ v42.20 — ۳.۲ GB  │ آخرین استفاده: ۴۵ روز پیش  │
│     استفاده در: —                                    │
│     [حذف]                                           │
├──────────────────────────────────────────────────────┤
│  مودها                                   ۶.۰ GB      │
│                                                      │
│  ✅ Brita's Armor Pack — ۴۵ MB  │ ۱ روز پیش        │
│  ✅ Hydrocraft — ۲۱۰ MB         │ ۲ روز پیش        │
│  ⚠️ [۱۴ مود دیگر — بلااستفاده]  │ ۴۵+ روز پیش     │
│     [حذف همه بلااستفاده]                            │
└──────────────────────────────────────────────────────┘
```

---

## Deletion confirmation

Before deleting a game version:

```
┌─────────────────────────────────────────┐
│  حذف v42.20؟                            │
│                                         │
│  ۳.۲ GB آزاد می‌شه                      │
│                                         │
│  ⚠️ هیچ سروری الان به این نیاز نداره   │
│  اگه بعداً لازم شد دوباره دانلود میشه  │
│                                         │
│  [حذف کن]    [نگه‌دار]                 │
└─────────────────────────────────────────┘
```

If a profile still references the version being deleted:

```
⛔ v42.16 در حال استفاده توسط "Survival RP" هست.
   ابتدا سرور رو از لیست حذف کن، بعد نسخه رو پاک کن.
```

---

## Settings

All thresholds configurable in `settings.json`:

```json
{
  "cache": {
    "versionsPath": "~/.pz-launcher/versions",
    "modsPath": "~/.pz-launcher/mods",
    "suggestVersionDeleteAfterDays": 30,
    "suggestModDeleteAfterDays": 60,
    "showCacheWarningAtGB": 20
  }
}
```

If total cache exceeds `showCacheWarningAtGB`, a banner appears in the main Launcher UI:

```
💾 فضای ذخیره‌سازی: ۲۳.۱ GB — [مدیریت فضا]
```

---

## Shared mod cache

Mods are stored once per SHA256, shared across all server profiles:

```
profiles/
  survival-rp/
    mods/ → symlink → mods/def456/   (Brita's Armor)
    mods/ → symlink → mods/ghi789/   (Hydrocraft)

  build-server/
    mods/ → symlink → mods/def456/   (same Brita's Armor, no duplicate)
```

A mod is only eligible for deletion when **no active profile** references it.

---

## Space estimation before join

Before joining a new server, the Launcher shows a preview:

```
┌──────────────────────────────────────────┐
│  پیش‌نیازهای "Survival RP"              │
│                                          │
│  v42.16 — دارم ✅           ۰ MB        │
│  ۱۲ مود جدید                ۱۸۰ MB      │
│  ۵ مود موجود ✅             ۰ MB        │
│                                          │
│  کل دانلود: ۱۸۰ MB                      │
│  [ادامه]                                │
└──────────────────────────────────────────┘
```

---

## Out of scope

- Automatic deletion without user confirmation (never)
- Remote cache management (user's PC is user's PC)
- Compression of stored versions (stored as-is after extraction)
