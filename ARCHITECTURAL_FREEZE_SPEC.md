# ARCHITECTURAL_FREEZE_SPEC.md
## Technical Implementation of Phase 1 Lock

**Purpose**: Transform from "forever-evolving architecture" into "shipped product with controlled evolution path"

**Audience**: Engineers, code reviewers, CI/CD maintainers

**Implementation**: Concrete, auditable, automated

---

## Executive Summary

```
BEFORE this spec:
  Architecture + Product coevolve
  → System stuck in "almost done"
  → Easy to rationalize "one more feature"

AFTER this spec:
  Architecture frozen (enforced)
  Product features only (encouraged)
  → Clear ship date
  → Easy to say "that's Phase 2"
```

---

## File Freeze Specification

### Category 1: LOCKED 🔴 (Exception process required)

```
STATUS: IMMUTABLE until Phase 2 decision log
CHANGE PROCESS: Full exception (3-day approval, logged)
RATIONALE: Part of production runtime contract

src/event/LauncherEventDispatcher.ts
├─ Why: Orchestrates all patch application
├─ What's frozen: All method signatures
├─ What's mutable: Implementation comments only
├─ Exception trigger: Any dispatch behavior change

src/event/LauncherStateReducer.ts
├─ Why: Pure event-to-patch transformation
├─ What's frozen: Return type, event handling logic
├─ What's mutable: Comment-level optimization notes
├─ Exception trigger: Any new event type handling

src/event/PatchSchemaRegistry.ts
├─ Why: Single source of validation truth
├─ What's frozen: Schema definitions, allowed keys
├─ What's mutable: Performance optimizations in lookups
├─ Exception trigger: Any new domain, new allowed key

src/event/StateReconstructor.ts
├─ Why: Production state recovery mechanism
├─ What's frozen: Reconstruction algorithm, timestamp accuracy
├─ What's mutable: Performance comments, logging
├─ Exception trigger: Any change to reconstruction order

src/event/SnapshotEngine.ts
├─ Why: O(1) state restoration enabler
├─ What's frozen: Snapshot format, intervals
├─ What's mutable: Compression algorithm optimization
├─ Exception trigger: Any format version change

src/event/EventCompaction.ts
├─ Why: Memory governance mechanism
├─ What's frozen: Compaction strategies, thresholds (20%)
├─ What's mutable: Performance optimization of algorithms
├─ Exception trigger: Any threshold change, new strategy

src/event/EventReplay.ts
├─ Why: Deterministic testing + debugging
├─ What's frozen: Replay contract, consistency checks
├─ What's mutable: UI for visualization
├─ Exception trigger: Any consistency rule change

src/event/PerformanceBoundaries.ts
├─ Why: Explicit performance governance
├─ What's frozen: All boundary constants, methods
├─ What's mutable: Comment documentation
├─ Exception trigger: Any constant value change

src/interfaces/LauncherEvent.ts
├─ Why: Event contract across system
├─ What's frozen: Type definitions, payload structure
├─ What's mutable: JSDoc comments, examples
├─ Exception trigger: Any interface property change
```

---

### Category 2: LOCKED 🔴 (Core stores - Exception process required)

```
STATUS: IMMUTABLE until Phase 2 decision log
CHANGE PROCESS: Full exception (3-day approval, logged)
RATIONALE: Part of production state contract

src/stores/eventLog.store.ts
├─ What's frozen: Store interface, query methods
├─ What's mutable: LRU eviction algorithm
├─ Exception trigger: Any field addition to EventLogEntry

src/stores/patchFailureLog.store.ts
├─ What's frozen: Failure tracking interface
├─ What's mutable: Memory cap tuning (within 50-500 bounds)
├─ Exception trigger: Any new property in PatchFailure

src/stores/snapshotStore.ts
├─ What's frozen: Snapshot CRUD contract
├─ What's mutable: Snapshot query optimization
├─ Exception trigger: Any change to snapshot data structure

src/stores/useDownloadsStore.ts
├─ What's frozen: State shape, patch receptors
├─ What's mutable: UI-only fields, computations
├─ Exception trigger: Any core state field change

src/stores/useTraceStore.ts
├─ What's frozen: Trace node structure, hierarchy
├─ What's mutable: Computed properties, formatting
├─ Exception trigger: Any core trace field change

src/stores/useSessionStore.ts
├─ What's frozen: Session state interface
├─ What's mutable: Session metadata fields
├─ Exception trigger: Any session ID format change

src/stores/useServersStore.ts
├─ What's frozen: Server state shape
├─ What's mutable: Server metadata
├─ Exception trigger: Any server address format change
```

---

### Category 3: LOCKED 🔴 (Adapters - Exception process required)

```
STATUS: IMMUTABLE until Phase 2 decision log
CHANGE PROCESS: Full exception (3-day approval, logged)
RATIONALE: Prevent Wails coupling leakage

src/adapters/wails-launcher.ts
├─ What's frozen: Wails event binding
├─ What's mutable: UX-facing behavior
├─ Exception trigger: Any new Wails method call

src/adapters/wails-session-manager.ts
├─ What's frozen: Session lifecycle contract
├─ What's mutable: Session metadata handling
├─ Exception trigger: Any session state structure change

src/adapters/wails-event-logger.ts
├─ What's frozen: Event logging interface
├─ What's mutable: Log format/buffering
├─ Exception trigger: Any event type not logged
```

---

### Category 4: MUTABLE 🟡 (UI components - Free to change)

```
STATUS: Free to modify, no exceptions needed
CHANGE PROCESS: Normal PR review (24h)
RATIONALE: Product layer, not architecture

src/components/EventLogDebugPanel.tsx
├─ Status: Free to improve UI/UX
├─ Constraint: Can't change EventLog query API
├─ Example: Can add visualization features

src/components/ModDiscovery.tsx
├─ Status: Free (new component)
├─ Example: Can add filters, sorting, pagination

src/components/DownloadQueue.tsx
├─ Status: Free (new component)
├─ Example: Can add pause/resume UI

src/components/GameLauncher.tsx
├─ Status: Free (new component)
├─ Example: Can add game options UI

src/components/LauncherUI.tsx
├─ Status: Free to refactor
├─ Constraint: Can't change event dispatch
├─ Example: Can reorganize layout

[All other components in src/components/]
├─ Status: Free to create/modify
└─ Constraint: None at architecture level
```

---

### Category 5: MUTABLE 🟡 (Services - Free to change)

```
STATUS: Free to create new services
CHANGE PROCESS: Normal PR review
RATIONALE: Product layer, not architecture

src/services/modSync.ts
├─ Status: NEW - Free to implement
└─ Example: Sync mod manifests

src/services/installManager.ts
├─ Status: NEW - Free to implement
└─ Example: Manage installation flow

src/services/gameRunner.ts
├─ Status: NEW - Free to implement
└─ Example: Launch game with mods

[All new service files]
├─ Status: Free to create
└─ Constraint: Must use frozen event system as interface
```

---

### Category 6: MUTABLE 🟡 (Utilities - Free to change)

```
STATUS: Free to modify
CHANGE PROCESS: Normal PR review
RATIONALE: Optimization + feature layer

src/utils/
├─ All files: Free to modify
├─ Constraint: Can't add to core exports
└─ Example: Can add formatting, parsing utilities

src/hooks/
├─ All files: Free to modify
├─ Constraint: Can't modify frozen store hooks
└─ Example: Can add new hooks for UX

src/constants/
├─ Constraint: Can't modify event-related constants
├─ Free: UI constants, configuration values
└─ Example: Can add UI colors, strings
```

---

### Category 7: DOCUMENTATION 🟢 (Always free)

```
STATUS: Free to modify always
CHANGE PROCESS: Normal PR review
RATIONALE: Clarification, not logic

docs/
├─ All files: Free to modify
└─ Constraint: None

README.md, etc.
├─ Status: Free to update
└─ Constraint: None

Code comments (internal)
├─ Status: Free to add/modify
└─ Constraint: None
```

---

## Type-Level Freezing Mechanism

### Sealed Interfaces (Cannot extend or modify)

```typescript
// src/interfaces/LauncherEvent.ts
export interface LauncherEvent {
  readonly type: LauncherEventType;
  readonly timestamp: number;
  readonly sessionId: string;
  readonly version: number;
  readonly payload: LauncherEventPayload;
  // ✅ LOCKED: No new fields
  // ✅ LOCKED: No optional fields
}

// If you need new data: New event type, not new field
// Exception process: New LauncherEventType in enum
```

### Sealed Enums (Cannot add values without exception)

```typescript
// src/interfaces/LauncherEvent.ts
export enum LauncherEventType {
  // ✅ LOCKED: These values never change
  TRACE_NODE_CREATED = 'trace_node_created',
  TRACE_NODE_UPDATED = 'trace_node_updated',
  // ... more 16 types
  // ❌ NEW EVENT TYPE: Requires exception process
}
```

### Sealed Patch Domains (Cannot add without exception)

```typescript
// src/event/PatchSchemaRegistry.ts
const PATCH_DOMAINS = [
  'trace',     // ✅ LOCKED
  'downloads', // ✅ LOCKED
  'session',   // ✅ LOCKED
  'servers',   // ✅ LOCKED
  // ❌ NEW DOMAIN: Exception process required
] as const;
```

---

## Pre-commit Hook Implementation

### File: `.git/hooks/pre-commit`

```bash
#!/bin/bash
set -e

# Phase 1 Enforcement: Prevent frozen file changes without exception

FROZEN_FILES=(
  "src/event/LauncherEventDispatcher.ts"
  "src/event/LauncherStateReducer.ts"
  "src/event/PatchSchemaRegistry.ts"
  "src/event/StateReconstructor.ts"
  "src/event/SnapshotEngine.ts"
  "src/event/EventCompaction.ts"
  "src/event/EventReplay.ts"
  "src/event/PerformanceBoundaries.ts"
  "src/interfaces/LauncherEvent.ts"
  "src/stores/eventLog.store.ts"
  "src/stores/patchFailureLog.store.ts"
  "src/stores/snapshotStore.ts"
  "src/stores/useDownloadsStore.ts"
  "src/stores/useTraceStore.ts"
  "src/stores/useSessionStore.ts"
  "src/stores/useServersStore.ts"
  "src/adapters/wails-*.ts"
)

# Get staged files
STAGED=$(git diff --cached --name-only)

# Check for violations
VIOLATION=false
for file in $STAGED; do
  for frozen in "${FROZEN_FILES[@]}"; do
    if [[ "$file" == "$frozen" ]]; then
      # Check if commit message includes exception tag
      if ! git diff --cached HEAD -- "$file" | grep -q "PHASE_EXCEPTION_"; then
        echo "❌ FROZEN FILE: $file"
        echo "   Cannot modify frozen architecture files in Phase 1"
        echo "   See: PHASE_ENFORCEMENT_PLAN.md"
        echo ""
        echo "   If this is critical:"
        echo "   1. Add [PHASE_EXCEPTION_DATE] to commit message"
        echo "   2. File exception PR with 3-day approval"
        VIOLATION=true
      fi
    fi
  done
done

if [ "$VIOLATION" = true ]; then
  exit 1
fi

echo "✅ Frozen files check passed"
exit 0
```

### Setup (run once):

```bash
chmod +x .git/hooks/pre-commit
```

---

## GitHub Actions PR Check

### File: `.github/workflows/phase-1-enforcement.yml`

```yaml
name: Phase 1 Enforcement

on:
  pull_request:
    types: [opened, synchronize]

jobs:
  check-frozen-files:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Check frozen files
        run: |
          FROZEN_FILES=(
            "src/event/LauncherEventDispatcher.ts"
            "src/event/LauncherStateReducer.ts"
            "src/event/PatchSchemaRegistry.ts"
            "src/event/StateReconstructor.ts"
            "src/event/SnapshotEngine.ts"
            "src/event/EventCompaction.ts"
            "src/event/EventReplay.ts"
            "src/event/PerformanceBoundaries.ts"
            "src/interfaces/LauncherEvent.ts"
            "src/stores/eventLog.store.ts"
            "src/stores/patchFailureLog.store.ts"
            "src/stores/snapshotStore.ts"
            "src/stores/useDownloadsStore.ts"
            "src/stores/useTraceStore.ts"
            "src/stores/useSessionStore.ts"
            "src/stores/useServersStore.ts"
            "src/adapters/wails-*.ts"
          )
          
          # Get changed files
          CHANGED=$(git diff origin/main --name-only)
          
          # Check for violations
          VIOLATION=false
          for file in $CHANGED; do
            for frozen in "${FROZEN_FILES[@]}"; do
              if [[ "$file" == "$frozen" ]]; then
                PR_BODY="${{ github.event.pull_request.body }}"
                if ! echo "$PR_BODY" | grep -q "PHASE_EXCEPTION_"; then
                  echo "❌ Frozen file modified: $file"
                  VIOLATION=true
                fi
              fi
            done
          done
          
          if [ "$VIOLATION" = true ]; then
            echo "❌ PR modifies frozen architecture files"
            echo "   See: PHASE_ENFORCEMENT_PLAN.md for exception process"
            exit 1
          fi
          
      - name: Check for new RFCs
        run: |
          if git diff origin/main --name-only | grep -q "docs/rfc/"; then
            VIOLATION=true
            echo "❌ New RFC added in Phase 1"
            echo "   Architecture work blocked until Phase 2"
            exit 1
          fi
```

---

## Exception Logging

### File: `docs/PHASE_1_EXCEPTIONS.log`

```markdown
# Phase 1 Exception Log

## Exception 001
- Date: 2026-06-05
- File: src/event/PerformanceBoundaries.ts
- Reason: Memory boundary adjusted from 10MB to 12MB based on production data
- Changed by: @engineer-name
- Approved by: @tech-lead, @product-lead
- Duration: 3 days (approved 2026-06-08)
- Risk level: 1 (monitoring added)
- Impact: Allows 20% more events per session in edge cases
- Review: No validation failures in test environment

## Exception 002
- Date: 2026-06-12
- File: src/event/LauncherStateReducer.ts
- Reason: Handle new event type from user research
- Changed by: @engineer-name
- Denied: Blocked (new event type requires Platform phase)
- Recommendation: Defer to Phase 2, implement workaround in UI

## Exception 003
- Date: 2026-06-15
- File: src/event/PatchSchemaRegistry.ts
- Reason: Add validation for user-generated mod metadata
- Changed by: @engineer-name
- Approved by: @tech-lead, @product-lead
- Duration: 2 days (approved 2026-06-17)
- Risk level: 2 (new validation rule, tested)
- Impact: Rejects invalid metadata earlier
- Review: Added comprehensive test coverage
```

---

## Weekly Governance Sync Agenda

### Schedule: Monday 10:00 AM (30 minutes)

```
Attendees: Tech lead, Product lead, Engineering team

1. Exception PRs (10 min)
   - Review any pending exceptions
   - Approve/deny with rationale
   - Log decisions

2. Frozen File Violations (5 min)
   - Check for pre-commit hook violations
   - Discuss blocked PRs
   - Reinforce boundaries

3. Phase 1 Metrics (5 min)
   - Build size (target: ≤ 200 KB)
   - Build time (target: < 1 second)
   - Error count (target: 0)

4. Shipping Status (5 min)
   - Product scope: On track?
   - Target launch date: Still valid?
   - Any phase 1 scope changes?

5. Phase 2 Planning (5 min)
   - When can we trigger Phase 2?
   - What preparation is needed?
   - Platform research status?
```

---

## Decision Log (Fill at Phase 1 Start)

### File: `PHASE_1_DECISION_LOG.md`

```markdown
# Phase 1 Lock-in Decision Log

## Decision Timestamp
[To be filled: Date of decision]

## Chosen Path
- [ ] Path A (Product only)
- [ ] Path B (Platform only)
- [X] Path C (Hybrid: Launcher first, platform second)

## Architecture Freeze Details
- Freeze Date: [To be filled]
- Freeze Duration: 8 weeks (target: end-of-week 8)
- Enforcement: Pre-commit hooks + GitHub Actions active
- Exception Process: Active (3-day minimum approval)

## Phase 1 Commitments
- [X] No new RFCs
- [X] No architecture changes
- [X] Event system: FROZEN
- [X] Core stores: FROZEN
- [X] Adapters: FROZEN
- [X] Product features: ENCOURAGED
- [X] UX components: FREE TO MODIFY

## Team Commitments
- Eng lead: Will enforce frozen files
- Product lead: Will not request architecture changes
- Team: Will work within Phase 1 boundaries

## Success Criteria (Phase 1 Done)
- [ ] Launcher ships to users
- [ ] Zero frozen file violations (or logged exceptions)
- [ ] Build size ≤ 200 KB
- [ ] Performance acceptable
- [ ] Architecture unchanged

## Signed Off By
- Engineering lead: _________________ Date: _____
- Product lead: _________________ Date: _____
- Leadership: _________________ Date: _____

## Phase 2 Trigger
Phase 2 begins when ALL of the following are true:
1. Launcher stable in production (2+ weeks no major bugs)
2. Phase 1 success criteria met
3. Decision log sealed (no changes allowed)
4. Platform research complete (RFC-0026 drafted)
5. New team phase assigned
```

---

## Implementation Checklist (Day 1 of Phase 1)

Before any product development starts:

- [ ] Pre-commit hooks installed and tested
- [ ] GitHub Actions workflow deployed
- [ ] Exception log created and linked
- [ ] Decision log filled and sealed
- [ ] Weekly governance calendar created
- [ ] Frozen files annotated with comments
- [ ] Team briefing completed
- [ ] Slack channel #phase-1-enforcement created
- [ ] First PR tests the enforcement (should succeed)
- [ ] Second PR tests the exception process (should fail, then approve)

---

## What Happens If Boundaries Break

### Scenario 1: Someone commits to frozen file

```
❌ Pre-commit hook blocks it
→ Commit fails locally
→ Developer must remove change or file exception
→ Exception goes to 3-day review
→ Exception logged
→ Decision log audit catches it
```

### Scenario 2: Someone force-pushes around hook

```
❌ GitHub Actions blocks the PR merge
→ PR review blocks merge
→ GitHub branch protection enforces block
→ Merge impossible without override
→ Override is auditable + logged
```

### Scenario 3: Someone adds new RFC

```
❌ GitHub Actions blocks the PR
→ PR review comments: "Phase 1 blocks new RFCs"
→ Cannot merge during Phase 1
→ Deferral to Phase 2 documented
```

---

## Final Enforceability Test

Run this command to verify enforcement is active:

```bash
# Test 1: Try to commit to frozen file
# Expected: Blocked by pre-commit hook
echo "test" >> src/event/LauncherEventDispatcher.ts
git add src/event/LauncherEventDispatcher.ts
git commit -m "test"
# Should fail with: "❌ FROZEN FILE: ..."
# If it succeeds: Pre-commit hooks not installed

# Test 2: Create exception PR
# Expected: Blocked by GitHub Actions initially, approved with exception tag
# Create PR changing frozen file with [PHASE_EXCEPTION_TEST] in message
# PR should pass all checks
```

---

## This Is The Whole System

With these mechanisms:

1. ✅ Boundaries are real (pre-commit + GitHub)
2. ✅ Exceptions are tracked (logged, auditable)
3. ✅ Process is clear (3-day exception with approval)
4. ✅ Phase can be enforced (decision log locked)
5. ✅ No "forever-in-progress" scenario
6. ✅ Ship date is defensible

**Without these mechanisms: This is just hope.**

**With these mechanisms: This is engineering discipline.**

Choose one.
