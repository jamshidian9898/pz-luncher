# Phase Enforcement Plan
## Making Hybrid Actually Work (Not Just Hope)

**Context**: System reached production-grade infrastructure (RFC-0024/0025). Risk now is "forever-in-progress architecture" if boundaries aren't enforced.

**Solution**: Concrete, auditable phase boundaries. No soft transitions. No "we'll refactor later."

**Status**: CRITICAL — Must be implemented before Phase 1 kicks off

---

## The Problem We're Solving

```
❌ Without boundaries:
  Month 3:  "Launcher mostly done, but architecture needs..."
  Month 5:  "Platform extraction blocked by launcher UX debt..."
  Month 8:  "Can't ship launcher, can't start platform, system stuck"

✅ With hard boundaries:
  Month 3:  Launcher done. Architecture frozen. Ship to users.
  Month 4:  Extract happens. Clear scope, no launcher changes.
  Month 5:  Platform foundation ready. New work starts.
```

---

## What "Phase 1 Lock" Means

### ✅ ALLOWED in Phase 1
- UX components (React)
- User flows
- Performance optimization within event system
- Bug fixes in shipped code
- Documentation
- Testing

### ❌ FORBIDDEN in Phase 1
- New architecture components
- New RFCs
- Changing core event system
- Adding new validation rules
- Modifying store interfaces
- Extracting new abstractions

### ⚠️ REQUIRES EXCEPTION in Phase 1
- Any file in `src/event/`
- Any file in `src/stores/` (core stores)
- `LauncherEventDispatcher`
- `PatchSchemaRegistry`
- `StateReconstructor`

---

## Concrete File Freeze List

### 🔴 FROZEN (No changes without exception board)
```
src/event/
├── LauncherEventDispatcher.ts          ← Core dispatcher
├── LauncherStateReducer.ts             ← Pure reducer
├── PatchSchemaRegistry.ts              ← Validation source of truth
├── StateReconstructor.ts               ← Reconstruction engine
├── SnapshotEngine.ts                   ← Snapshot system
├── EventCompaction.ts                  ← Compaction logic
├── EventReplay.ts                      ← Replay engine
├── PerformanceBoundaries.ts            ← Performance constants

src/stores/
├── eventLog.store.ts                   ← Event persistence
├── patchFailureLog.store.ts            ← Failure tracking
├── snapshotStore.ts                    ← Snapshot persistence
├── useDownloadsStore.ts                ← Core store
├── useTraceStore.ts                    ← Core store
├── useSessionStore.ts                  ← Core store
├── useServersStore.ts                  ← Core store

src/interfaces/
├── LauncherEvent.ts                    ← Event contract
```

**Exception process**: Requires:
1. Pull request with rationale
2. Architecture review (2 reviewers)
3. Product owner sign-off
4. Tech lead approval
5. Documented in exception log

---

### 🟡 MUTABLE (Only UX-level changes)
```
src/components/
├── EventLogDebugPanel.tsx              ← Debug UI (can improve)
├── [Other launcher UI]                 ← All UX components free to change

src/hooks/
├── [All custom hooks]                  ← Free to use, not export

src/services/
├── [Adapter services]                  ← Free to implement

docs/
├── architecture/                       ← Free to document
├── [Non-RFC docs]                      ← Free to expand
```

---

### 🟢 NEW (Phase 1 only)
```
✅ src/components/ModDiscovery/        ← New launcher features
✅ src/components/DownloadQueue/       ← New launcher features
✅ src/components/GameLauncher/        ← New launcher features
✅ src/services/modSync.ts             ← New product services
✅ src/adapters/wails-*.ts             ← Adapter layers
```

---

## APIs That Become Immutable

### Event Interface (LOCKED)
```typescript
// ✅ This NEVER changes in Phase 1
interface LauncherEvent {
  type: LauncherEventType;
  timestamp: number;
  sessionId: string;
  version: number;
  payload: LauncherEventPayload;
}

// Any change = full exception process
```

### Patch Schema (LOCKED)
```typescript
// ✅ This NEVER changes in Phase 1
interface LauncherEventPatch {
  domain: 'trace' | 'downloads' | 'session' | 'servers';
  operations: PatchOperation[];
  timestamp: number;
}

// Adding new domain = full exception process
```

### Store Interfaces (LOCKED)
```typescript
// ✅ These signatures NEVER change
export interface DownloadsStore {
  downloadedMods: Record<string, ModDownload>;
  activeDownloads: string[];
  // ...
}

// Any new field = full exception process
```

### Dispatcher Contract (LOCKED)
```typescript
// ✅ This dispatch signature LOCKED
dispatchLauncherEvent(event: LauncherEvent): Promise<DispatchResult>

// Change signatures = full exception process
```

---

## Metrics That Define "Phase 1 Done"

### Code Metrics
```
✅ Zero errors in `npm run build`
✅ TypeScript strict mode: 100% passing
✅ Event system: no patches rejected in production
✅ Build size: ≤ 200 KB gzipped (no bloat)
```

### Delivery Metrics
```
✅ Launcher UX complete
✅ All critical user flows implemented
✅ No "architectural debt blockers" for users
✅ Performance acceptable (state reconstruction < 1 second)
```

### Architecture Metrics
```
✅ Zero RFC changes
✅ Zero core store changes
✅ Zero event system changes
✅ All exceptions documented and approved
```

---

## Phase 1 Timeline & Checkpoints

### Week 1-2: Launcher Features
```
Checkpoint: Core mod discovery working
Allowed work: UX components only
Frozen work: Event system + stores
Exception allowed: 0 (architecture locked)
```

### Week 3-4: Install & Launch
```
Checkpoint: Download + installation working
Allowed work: UX components only
Frozen work: Event system + stores
Exception allowed: 0 (architecture locked)
```

### Week 5-6: Polish & Stability
```
Checkpoint: User-facing features complete
Allowed work: Bug fixes + UX refinement
Frozen work: Event system + stores
Exception allowed: Critical bugs only
```

### Week 7-8: Launch Ready
```
Checkpoint: All blockers cleared
Allowed work: Documentation + testing
Frozen work: All product code (code freeze)
Exception allowed: 0
```

**Result**: Shippable, stable launcher with frozen architecture

---

## Hard Stop Criteria (When Phase 1 ENDS)

### ✅ Phase 1 Ends When ALL are true:

1. **Launcher ships to users**
   - Binary available
   - Users running it
   - Basic telemetry working

2. **Architecture unchanged**
   - Zero core system modifications
   - All exceptions logged
   - RFC count: still 25

3. **Code freeze enforced**
   - No new runtime changes
   - Only docs/comments/cosmetic changes allowed
   - Build artifacts sealed

### ❌ Phase 1 Does NOT end when:
- "Architecture looks good but needs refactoring"
- "Platform ideas are clear now"
- "Let's improve event system while we're at it"
- "One more RFC would really help"

**If any of these occur: EXCEPTION PROCESS required**

---

## What Phase 2 Means (After Hard Stop)

### When Phase 1 is genuinely done:

```
✅ Launcher running in production
✅ Architecture frozen (decision log sealed)
✅ Event system proven (no failures)
✅ Performance stable (no surprises)

THEN → Phase 2 planning begins
```

### Phase 2 Can Touch:

```
✅ src/platform/              ← NEW directory, service layer
✅ Plugin system              ← NEW RFC-0026
✅ Multi-game abstraction     ← NEW RFC-0027
✅ Extract core from launcher ← Refactoring allowed
✅ Event versioning           ← NEW capability
```

### Phase 2 Cannot Touch:

```
❌ src/event/ (already proven)
❌ src/stores/ (already proven)
❌ LauncherEvent contract (already in use)
❌ Patch schema (already validated)
```

---

## Enforcement Mechanisms

### 1. Pre-commit Hook
```bash
# Prevents commits to frozen files
# Exception: requires tag in commit message

if [ frozen_file_changed ] && [ no_PHASE_EXCEPTION_* ]; then
  echo "❌ Cannot modify frozen files in Phase 1"
  echo "   See: PHASE_ENFORCEMENT_PLAN.md"
  exit 1
fi
```

### 2. PR Checks
```yaml
# GitHub Actions checks:
- No changes to src/event/* without exception PR
- No changes to src/stores/* without exception PR
- No new RFCs allowed in Phase 1
- No API signature changes
```

### 3. Exception Log (Auditable)
```markdown
# Phase 1 Exceptions Log

## Exception #1
- Date: 2026-06-10
- File: src/event/PerformanceBoundaries.ts
- Reason: Memory threshold adjustment (production data showed 8% utilization)
- Approved by: [Engineering lead], [Product lead]
- Impact: Risk level 2 (monitoring added)
```

### 4. Weekly Sync
```
Monday 10 AM: Architecture Governance
- Review any exception PRs
- Check frozen file violations
- Confirm phase boundaries respected
- Document decisions
```

---

## Decision Log Template

When Phase 1 starts, fill in:

```markdown
# PHASE 1 LOCK-IN

## Decision Timestamp
June 4, 2026, 2:00 PM

## Chosen Path
✅ Hybrid (Launcher → Platform)

## Phase 1 Commitments
- [ ] No new architecture
- [ ] Event system: FROZEN
- [ ] Core stores: FROZEN
- [ ] RFCs: No new ones
- [ ] Pre-commit hooks: ACTIVE
- [ ] PR checks: ACTIVE

## Phase 1 Duration
Weeks 1-8 (Approx 2 months to launch)

## Phase 1 Done Criteria
- [ ] Launcher ships
- [ ] Zero frozen file violations
- [ ] Build size stable
- [ ] Performance acceptable

## Signed Off By
- Engineering: ___________
- Product: ___________
- Leadership: ___________

## Phase 2 Begins When
All Phase 1 criteria met AND decision log sealed
```

---

## What Breaks If We Don't Do This

### Scenario: No Boundaries
```
Month 3: "Launcher feature needs architecture change"
  → "Just one small RFC"
  → 5 days of architecture work
  → Users delayed

Month 5: "Platform work blocked by launcher UX debt"
  → "Let's refactor event system"
  → 2 weeks of changes
  → Platform delayed

Month 8: "Neither product shipped nor platform ready"
  → "System stuck in 'almost done' state"
  → Team demoralized
  → Customers lost
```

### Scenario: With Boundaries (This Plan)
```
Month 3: "Launcher feature needs architecture change"
  → Exception process: Denied (not critical)
  → Ship without it
  → Users happy

Month 5: "Platform work needs refactoring"
  → Phase 2 begins
  → Refactoring is now ON PLAN
  → Proceeding on schedule

Month 8: "Both launcher in production and platform foundation ready"
  → Clear evolution path
  → Team confident
  → Customers expanding
```

---

## Enforcement Is The Whole Point

This document only works if:

1. **Pre-commit hooks are active** (day 1)
2. **PR checks are configured** (day 1)
3. **Exception process is real** (requires 3-day approval)
4. **Weekly governance sync** (every Monday)
5. **Decision log is sealed** (unchangeable once Phase 1 starts)

**Without enforcement mechanisms: This is just a hope document.**

With enforcement: This is a **shipping commitment**.

---

## Next Steps (Before Phase 1 Starts)

### Week of decision:
- [ ] Implement pre-commit hook
- [ ] Configure GitHub Actions PR checks
- [ ] Create exception tracking sheet
- [ ] Schedule weekly governance sync
- [ ] Seal decision log

### Day 1 of Phase 1:
- [ ] Frozen files marked as such (comments)
- [ ] Team briefing on boundaries
- [ ] Pre-commit hook verified
- [ ] First PR submitted against new rules

### Ongoing:
- [ ] Weekly sync (30 min)
- [ ] Monthly exception review
- [ ] Phase 1 metrics dashboard
- [ ] Public commitment to boundaries

---

## The Most Important Rule

> **If architecture work is needed in Phase 1: It's a sign Phase 1 scope was wrong.**
>
> Don't relax boundaries. Re-scope Phase 1.

This is how you know if hybrid actually works:

- ✅ **Architecture phase truly done**
- ✅ **Product phase has no architecture needs**
- ✅ **Zero pressure to break boundaries**

If you feel pressure: That's signal that boundaries are working correctly.

---

## Final Check

Before starting Phase 1, ask:

```
Q: Can we honestly ship a launcher without touching src/event/?
❌ If NO → Phase 1 scope is too ambitious
✅ If YES → Phase 1 boundaries are realistic
```

If YES: Run pre-commit hooks, start Phase 1 timer.

If NO: Adjust Phase 1 scope until answer is YES.

**That's it. Concrete. Enforceable. Real.**
