# RFC-0060: Community Upload & Cross-Validation

**Status**: Active — v2.1.0  
**Depends on**: [RFC-0059](0059-content-registry.md)  
**Feeds**: RFC-0059 trust level system

---

## Purpose

Define how game client binaries enter the Registry without relying on Steam or The Indie Stone's distribution. Since Steam is inaccessible in Iran and similar regions, trusted community members upload game binaries. Cross-validation between independent uploaders replaces traditional DRM verification.

---

## Core principle

> If N independent people, who do not know each other, submit a file with the same SHA256 hash, the probability that all of them have a tampered or malicious copy is negligible.

We do not perform virus scanning. We do not verify purchase. We verify **consensus**.

---

## Uploader roles

```
Anonymous Player
  → cannot upload binaries
  → can download verified content

Trusted Uploader
  → can submit game binary versions
  → earned by: manual approval from admin
  → initially: founding community members

Agent (automatic)
  → pushes mod files to registry automatically
  → counted as one trusted source for mods
  → not counted for game binary validation (agent has server binary, not client)
```

---

## Upload flow (game binary)

```
Trusted uploader has PZ v42.16 client (Windows)
        ↓
Opens Launcher → Settings → "کمک به جامعه"
        ↓
Selects local game installation path
        ↓
Launcher computes SHA256 of binary archive (background)
        ↓
Launcher submits:
  POST /api/v1/registry/upload
  {
    "gameId": "pz",
    "gameVersion": "42.16",
    "platform": "windows-x64",
    "sha256": "abc123...",
    "sizeBytes": 3200000000
  }
        ↓
Backend checks: does this SHA256 already exist?

  Case A — New hash, first submission:
    → Status: Pending (1/3)
    → Backend asks uploader to upload the actual file
    → File stored, tagged Pending

  Case B — Hash matches existing Pending entry:
    → Upload count incremented (2/3, 3/3...)
    → No file re-upload needed (dedup by SHA256)
    → At threshold (3) → Status upgraded to Verified ✅

  Case C — Hash does NOT match existing Pending entry:
    → Conflict recorded
    → Admin notified
    → Both entries stay Pending until resolved
```

---

## Cross-validation threshold

| Trust level | Condition |
|-------------|-----------|
| Pending | 1–2 independent sources |
| Verified | 3+ independent sources with matching SHA256 |
| Rejected | Any mismatch between submitted hashes |

"Independent" means: different uploader accounts, different IP subnets. The same person submitting twice does not count as two sources.

---

## What uploaders submit

Uploaders do not re-upload large files if the hash already exists in the registry. The upload protocol is:

```
1. Client computes SHA256 locally (before any upload)
2. Client sends: { sha256, gameVersion, platform, sizeBytes }
3. Backend responds:
   - "hash known, count incremented" → done, no file transfer
   - "hash unknown, please upload" → client streams the file
```

This minimizes bandwidth — only the first uploader of a given hash pays the upload cost. All subsequent uploaders just confirm the hash.

---

## Conflict resolution

When two uploaders submit different SHA256 for the same `(gameVersion, platform)`:

```
Admin panel shows:
  v42.16 / windows-x64 — CONFLICT
    Hash A: abc123... — 2 sources
    Hash B: def456... — 1 source

Admin action:
  → Request more submissions from community
  → When one hash reaches threshold and the other stays behind → resolve in favor of threshold
  → If both reach threshold → escalate (very rare — indicates a legitimate version split, e.g. retail vs GOG)
```

---

## Mod upload (via Agent)

Mod files follow a simpler flow since they come from running game servers:

```
Agent discovers mod → computes SHA256 → pushes to Backend
Backend checks: same SHA256 from 2+ independent agents → Verified
```

For mods, threshold is 2 (lower than game binary) because:
- Mod content is smaller and easier to validate contextually
- Multiple servers often run the same workshop mod with the same files

---

## UI — uploader experience

In Launcher settings, a section "کمک به جامعه" (Contribute):

```
┌────────────────────────────────────────────┐
│  کمک به پایگاه داده نسخه‌ها                │
│                                            │
│  نسخه‌هایی که روی PC تو وجود دارن:        │
│  ✅ v42.20 — تأییدشده (قبلاً ارسال شده)   │
│  ⬜ v42.16 — نیاز به تأیید بیشتر دارد     │
│                                            │
│  [ارسال v42.16 برای تأیید جامعه]          │
└────────────────────────────────────────────┘
```

The Launcher auto-detects all locally installed versions and shows which ones the registry still needs more validators for.

---

## Privacy

- Uploader identity is stored server-side for conflict resolution only
- Public-facing pages never show who submitted a specific hash
- IP subnets are stored (for independence check), not full IPs
- Uploader accounts are optional pseudonyms (no real name required)

---

## Out of scope

- Automated virus scanning (cross-validation is the trust model)
- Purchase verification (not required — contributing is opt-in)
- Paid uploader tiers
- Automated scraping of game files from any source
