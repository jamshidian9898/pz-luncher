# RFC-0031: Mod Dependency Resolver

**Status**: Active — Phase 1 Product / v2.0.0 compatible  

> **v2.0.0 note**: Resolver logic is unchanged. The input `ServerManifest` now originates from the Backend `JoinResponse` rather than a local file or direct URL fetch. The `workshopId` field on `ResolvedMod` is informational only; `downloadUrl` is not used by the resolver — download URLs come from the Backend download plan.  


**Depends on**: [RFC-0030](0030-server-manifest-v1.md)  
**Extends**: [RFC-0013](0013-manifest-resolution.md), [RFC-0019](0019-package-resolution.md)  
**Feeds**: RFC-0032, RFC-0033

---

## Problem

Given `ModEntry[]` from a server manifest, the launcher must produce a **deterministic install plan**: load order, transitive deps, and clear failures for cycles and version conflicts.

---

## Goals

- Build dependency graph from manifest edges
- Detect cycles before download
- Resolve version conflicts with explicit errors
- Output `ResolvedModPlan` for download + profile build

## Non-goals

- Cross-server mod deduplication policy (cache layer handles bytes)
- Runtime mod compatibility with game API (trust manifest + game version)

---

## API

### Input

```ts
type ResolverInput = {
  manifest: ServerManifest;
  installedMods?: Record<string, string>; // id → version, optional local hint
};
```

### Output

```ts
export interface ResolvedModPlan {
  serverId: string;
  manifestVersion: string;
  gameVersion: string;

  /** Topologically sorted install order */
  orderedMods: ResolvedMod[];

  /** Mods skipped (already satisfied in cache/profile) */
  skipped: string[];

  warnings: ResolverWarning[];
}

export interface ResolvedMod {
  id: string;
  name: string;
  version: string;
  sha256: string;
  workshopId?: string;
  downloadUrl?: string;
  depth: number;              // 0 = no deps
  dependsOn: string[];
}

export interface ResolverWarning {
  code: string;
  message: string;
  modId?: string;
}
```

---

## Algorithm

```text
1. Index mods by id
2. For each mod, add edges: mod → each dependency (dep must exist)
3. If optional mod omitted and not required by chain → exclude
4. DFS/BFS cycle detection → fail with RESOLVER_CYCLE
5. Version check: if installedMods[id] present and ≠ required → RESOLVER_VERSION_CONFLICT
6. Topological sort (Kahn or DFS post-order)
7. If profile.modLoadOrder present → stable-sort within topo layers
8. Emit ResolvedModPlan
```

---

## Error codes (user-visible)

| Code | Meaning |
|------|---------|
| `RESOLVER_UNKNOWN_DEP` | Dependency id not in manifest |
| `RESOLVER_CYCLE` | Circular dependency |
| `RESOLVER_VERSION_CONFLICT` | Installed version ≠ required |
| `RESOLVER_EMPTY_MANIFEST` | No mods to install |
| `RESOLVER_MISSING_SOURCE` | No source reference (v1.x only; v2.0.0: Backend issues all URLs) |

---

## Go alignment

Reuse / wrap:

- `libs/resolver` — package graph from contracts
- `libs/contracts/package_graph.go`

New function surface (names indicative):

```go
func ResolveModPlan(manifest *ManifestV1, opts ResolveOpts) (*ResolvedModPlan, error)
```

---

## Tests (required)

| Case | Expected |
|------|----------|
| Linear A→B→C | Order C,B,A or topo equivalent |
| Diamond A→B,C→D | D after B,C after A |
| Cycle A↔B | `RESOLVER_CYCLE` |
| Unknown dep | `RESOLVER_UNKNOWN_DEP` |
| Explicit load order override | Honors order when compatible |

Fixtures: `fixtures/resolver/*.json`

---

## Events

- `resolver.completed` — `{ modCount, skippedCount }`
- `resolver.failed` — `{ code, message, modId? }`

---

## Week 2 exit criteria

- [ ] Resolver library with ≥ 5 fixture tests
- [ ] `launcher-core` calls resolver after manifest validate
- [ ] UI shows resolver errors in join flow
