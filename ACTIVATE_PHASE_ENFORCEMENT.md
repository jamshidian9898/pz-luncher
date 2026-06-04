# ACTIVATE_PHASE_ENFORCEMENT.md
## How to Make Phase 1 Boundaries Real (Not Hope)

**Time to complete**: 15 minutes  
**Difficulty**: Low (mostly copy-paste)  
**Outcome**: Concrete enforcement mechanisms active

---

## ⚠️ Important Note

This is not a best practice recommendation.

**This is a requirement for the Hybrid path to work.**

Without these mechanisms in place, Phase 1 becomes "forever-in-progress architecture."

---

## Step 1: Install Pre-commit Hook (5 minutes)

### Create the hook file:

```bash
cat > .git/hooks/pre-commit << 'EOF'
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
)

STAGED=$(git diff --cached --name-only)

VIOLATION=false
for file in $STAGED; do
  for frozen in "${FROZEN_FILES[@]}"; do
    if [[ "$file" == "$frozen" ]]; then
      echo "❌ FROZEN FILE: $file"
      echo "   Cannot modify frozen architecture files in Phase 1"
      echo ""
      echo "   If this is critical:"
      echo "   1. See PHASE_ENFORCEMENT_PLAN.md for exception process"
      echo "   2. File an exception PR (requires 3-day approval)"
      echo "   3. Add [PHASE_EXCEPTION_REASON] to your commit"
      VIOLATION=true
    fi
  done
done

if [ "$VIOLATION" = true ]; then
  exit 1
fi

exit 0
EOF

chmod +x .git/hooks/pre-commit
```

### Verify it's installed:

```bash
ls -la .git/hooks/pre-commit
# Should show: -rwxr-xr-x ... pre-commit
```

### Test it works:

```bash
# This should be BLOCKED:
echo "test change" >> src/event/LauncherEventDispatcher.ts
git add src/event/LauncherEventDispatcher.ts
git commit -m "test" 2>&1 | grep "FROZEN FILE"
# If you see "FROZEN FILE" message: ✅ Hook works
# Undo the test change:
git reset
git checkout src/event/LauncherEventDispatcher.ts
```

---

## Step 2: Create GitHub Actions Check (7 minutes)

### Create workflow file:

```bash
mkdir -p .github/workflows
cat > .github/workflows/phase-1-enforcement.yml << 'EOF'
name: Phase 1 Enforcement

on:
  pull_request:
    types: [opened, synchronize, reopened]

jobs:
  check-frozen-files:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
        with:
          fetch-depth: 0

      - name: Check for frozen file modifications
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
          )
          
          # Get list of changed files in this PR
          git diff origin/main... --name-only > /tmp/changed_files.txt
          
          VIOLATION=false
          while IFS= read -r file; do
            for frozen in "${FROZEN_FILES[@]}"; do
              if [[ "$file" == "$frozen" ]]; then
                # Check if PR title or body contains exception tag
                PR_TITLE="${{ github.event.pull_request.title }}"
                PR_BODY="${{ github.event.pull_request.body }}"
                FULL_TEXT="${PR_TITLE}\n${PR_BODY}"
                
                if echo -e "$FULL_TEXT" | grep -qi "PHASE_EXCEPTION"; then
                  echo "✓ Exception approved for: $file"
                else
                  echo "❌ Frozen file modified without exception: $file"
                  VIOLATION=true
                fi
              fi
            done
          done < /tmp/changed_files.txt
          
          if [ "$VIOLATION" = true ]; then
            echo ""
            echo "❌ PR modifies frozen architecture files without exception"
            echo "   See: PHASE_ENFORCEMENT_PLAN.md for exception process"
            echo ""
            echo "   To fix:"
            echo "   1. Add [PHASE_EXCEPTION_REASON] to PR title or description"
            echo "   2. Get approval from tech lead + product lead"
            echo "   3. Exception will be logged for audit"
            exit 1
          fi

      - name: Check for new RFCs
        run: |
          if git diff origin/main... --name-only | grep -q "^docs/rfc/.*\.md$"; then
            echo "❌ New RFC introduced in Phase 1"
            echo "   Architecture work is blocked until Phase 2 begins"
            echo "   See: PHASE_ENFORCEMENT_PLAN.md"
            exit 1
          fi

      - name: Verify build still works
        working-directory: apps/launcher-ui/frontend
        run: |
          npm ci --prefer-offline --no-audit
          npm run build -- --no-minify
          echo "✓ Build successful"
EOF
```

### Commit and push the workflow:

```bash
git add .github/workflows/phase-1-enforcement.yml
git commit -m "chore: activate phase 1 enforcement mechanisms"
git push origin main
```

### Verify it's active:

Go to: `https://github.com/[your-repo]/actions`

Look for: "Phase 1 Enforcement" workflow  
Should show: ✅ Successful on main branch

---

## Step 3: Create Exception Log File (2 minutes)

```bash
cat > docs/PHASE_1_EXCEPTIONS.md << 'EOF'
# Phase 1 Exception Log

**Purpose**: Track all exceptions to frozen files during Phase 1

**Format**: Each exception requires:
- Date approved
- File modified
- Reason for exception
- Approvers (tech lead + product lead)
- Risk level (1=low, 2=medium, 3=high)

---

## Active Exceptions

(None yet - start recording here)

---

## Exception Policy

- **Approval time**: Minimum 3 days
- **Approvers required**: Tech lead + Product lead + 1 reviewer
- **Risk levels**:
  - Level 1: Low risk, well-tested, clear rationale
  - Level 2: Medium risk, requires testing, needs monitoring
  - Level 3: High risk, requires full phase impact analysis
- **Threshold**: If > 3 Level-3 exceptions, reassess Phase 1 scope

---

## Denied Exceptions

(Record rejections here with rationale)

---

## Phase 1 Statistics

| Metric | Value |
|--------|-------|
| Start Date | [To be filled] |
| Target End Date | [To be filled] |
| Total Exceptions Approved | 0 |
| Total Exceptions Denied | 0 |
| Average Approval Time | N/A |
| Build Size (current) | [Check: npm run build] |

EOF

git add docs/PHASE_1_EXCEPTIONS.md
git commit -m "chore: initialize exception log"
```

---

## Step 4: Create Decision Log & Seal (2 minutes)

```bash
cat > PHASE_1_DECISION_LOG.md << 'EOF'
# Phase 1 Lock-in Decision Log

## Decision Timestamp
[FILL DATE HERE]

## Chosen Path
- [ ] Path A (Product only)
- [ ] Path B (Platform only)
- [X] Path C (Hybrid: Launcher first → Platform second)

## Phase 1 Scope
- **Duration**: 8 weeks
- **Target**: Ship launcher to users
- **Constraint**: Zero architecture changes (exception process available)

## Frozen Components
- ✅ Event system (src/event/*)
- ✅ Core stores (src/stores/*)
- ✅ Adapters (src/adapters/*)
- ✅ Event interfaces (src/interfaces/*)

## Free Components
- ✅ UI components (src/components/*)
- ✅ Services (src/services/*)
- ✅ Utilities (src/utils/*)
- ✅ Documentation (docs/*)

## Enforcement Active
- ✅ Pre-commit hook installed
- ✅ GitHub Actions workflow active
- ✅ Exception log initialized
- ✅ Weekly sync scheduled

## Success Criteria
- [ ] Launcher ships to production
- [ ] Zero frozen file violations (or all logged)
- [ ] Build size ≤ 200 KB
- [ ] No performance regressions
- [ ] Team on schedule

## Team Sign-off
- [ ] Engineering lead: ___________ Date: _____
- [ ] Product lead: ___________ Date: _____
- [ ] Tech lead: ___________ Date: _____

**Once signed: This log is SEALED. No changes allowed until Phase 2 decision.**

EOF

git add PHASE_1_DECISION_LOG.md
git commit -m "chore: initialize phase 1 decision log"
```

---

## Step 5: Mark Frozen Files with Comments (1 minute)

### Add comment to top of each frozen file:

For example, in `src/event/LauncherEventDispatcher.ts`:

```typescript
/**
 * ⚠️  PHASE 1 FREEZE: This file is locked during Phase 1 (Launcher)
 *
 * Changes require exception process (3-day minimum approval)
 * See: PHASE_ENFORCEMENT_PLAN.md
 *
 * Reason: Core event dispatch orchestration - production contract
 * 
 * Last modified: [current date]
 * Last exception: None yet
 */

import { ... }
```

---

## Step 6: Schedule Weekly Governance Sync (1 minute)

### Add to engineering team calendar:

```
Weekly Phase 1 Governance Sync
Monday 10:00 AM (30 minutes)

Attendees:
- Engineering lead
- Product lead
- Tech lead
- Team (optional)

Agenda:
1. Review exception PRs (10 min)
2. Check frozen file violations (5 min)
3. Monitor Phase 1 metrics (5 min)
4. Shipping status (5 min)
5. Any scope changes needed? (5 min)
```

---

## Step 7: Create Slack Channel for Communication (1 minute)

```bash
# In Slack: Create #phase-1-enforcement

# Post this message:

📌 **Phase 1 Enforcement Active**

This project is now under Phase 1 lock-in (Hybrid path activated).

**What this means:**
✅ Product features: Encouraged (fast iteration)
❌ Architecture changes: Blocked (exception process required)

**Key documents:**
- PHASE_ENFORCEMENT_PLAN.md - Full policy
- ARCHITECTURAL_FREEZE_SPEC.md - Technical details
- docs/PHASE_1_EXCEPTIONS.md - Exception log

**If you need to modify a frozen file:**
1. Read PHASE_ENFORCEMENT_PLAN.md
2. File exception PR with [PHASE_EXCEPTION_REASON]
3. Get approval from tech lead + product lead
4. Exception logged for audit

**Questions?** Ping @tech-lead in this channel.
```

---

## Step 8: Run Verification (2 minutes)

```bash
# Verify everything is active:

# Test 1: Pre-commit hook
echo "test" >> src/event/PerformanceBoundaries.ts
git add src/event/PerformanceBoundaries.ts
git commit -m "test" 2>&1 | grep -i "frozen\|phase"
# Expected: Message about frozen files
git reset
git checkout src/event/PerformanceBoundaries.ts

# Test 2: GitHub Actions is deployed
git push origin main
# Go to: https://github.com/[repo]/actions
# Should see: "Phase 1 Enforcement" workflow listed and passing

# Test 3: Exception log exists
ls -la docs/PHASE_1_EXCEPTIONS.md
# Should exist

# Test 4: Decision log exists  
ls -la PHASE_1_DECISION_LOG.md
# Should exist

# If all 4 pass: ✅ PHASE 1 ENFORCEMENT ACTIVE
```

---

## Checklist for Day 1

- [ ] Pre-commit hook installed and tested
- [ ] GitHub Actions workflow deployed and passing
- [ ] Exception log created and linked
- [ ] Decision log created and ready to sign
- [ ] Slack channel created with announcement
- [ ] Weekly sync scheduled
- [ ] Frozen files annotated with comments
- [ ] Team briefing completed (30 min meeting)
- [ ] First product PR created (should pass enforcement)
- [ ] Verification checklist all passing

**Time to complete**: ~30 minutes total  
**Result**: Boundaries are now real and enforceable

---

## Common Scenarios

### Scenario 1: Developer tries to modify frozen file

```bash
cd src/event/
echo "// optimization idea" >> PerformanceBoundaries.ts
git add .
git commit -m "optimize performance"

# Result:
# ❌ FROZEN FILE: src/event/PerformanceBoundaries.ts
# Cannot modify frozen architecture files in Phase 1
# (commit fails, change not made)
```

### Scenario 2: Developer files exception

```bash
# Pull request title:
"[PHASE_EXCEPTION_BUGFIX] Fix event validation crash"

# PR description:
"Production crash in validator when handling edge case events.
PHASE_EXCEPTION: Critical production bug fix
Affects: src/event/PatchSchemaRegistry.ts
Risk: Low (single validation case, fully tested)"

# GitHub Actions result: ✅ PASS (exception tag detected)

# Approval process:
- @tech-lead approves (verifies production impact)
- @product-lead approves (verifies business priority)
- @reviewer approves (code review)
# Exception logged in PHASE_1_EXCEPTIONS.md
# PR merges after 3 days
```

### Scenario 3: Developer tries to add new RFC

```bash
# Files changed:
- docs/rfc/0026-new-architecture.md (NEW)
- src/event/LauncherEventDispatcher.ts (modified to support it)

# GitHub Actions result: ❌ FAIL
# "New RFC introduced in Phase 1"
# "Architecture work is blocked until Phase 2"

# Resolution:
# Defer to Phase 2 (6 weeks later)
# Implement workaround in product layer instead
```

---

## If Enforcement Doesn't Work

### Problem: Pre-commit hook not running

```bash
# Fix:
git config core.hooksPath .git/hooks
chmod +x .git/hooks/pre-commit

# Verify:
cd src/event
echo "test" >> LauncherEventDispatcher.ts
git add .
git commit -m "test"
# Should fail with FROZEN FILE message
```

### Problem: GitHub Actions not running

```bash
# Fix:
1. Go to: GitHub repo → Settings → Actions
2. Check: "Allow all actions and reusable workflows"
3. Go to: Actions tab
4. See: "Phase 1 Enforcement" should be there
5. Trigger manually to verify
```

### Problem: Someone force-pushed around checks

```bash
# Fix (GitHub):
1. Go to: Settings → Branches
2. Find: main branch
3. Enable: "Require status checks to pass"
4. Add check: "Phase 1 Enforcement"
5. Now force-push is blocked
```

---

## Final Check: System is Locked

Run this to verify everything is working:

```bash
#!/bin/bash

echo "🔍 Phase 1 Enforcement Verification"
echo ""

# Check 1: Pre-commit hook
if [ -x .git/hooks/pre-commit ]; then
  echo "✅ Pre-commit hook installed"
else
  echo "❌ Pre-commit hook missing"
  exit 1
fi

# Check 2: GitHub Actions workflow
if [ -f .github/workflows/phase-1-enforcement.yml ]; then
  echo "✅ GitHub Actions workflow exists"
else
  echo "❌ GitHub Actions workflow missing"
  exit 1
fi

# Check 3: Exception log
if [ -f docs/PHASE_1_EXCEPTIONS.md ]; then
  echo "✅ Exception log exists"
else
  echo "❌ Exception log missing"
  exit 1
fi

# Check 4: Decision log
if [ -f PHASE_1_DECISION_LOG.md ]; then
  echo "✅ Decision log exists"
else
  echo "❌ Decision log missing"
  exit 1
fi

# Check 5: Build still works
echo ""
echo "Testing build..."
cd apps/launcher-ui/frontend
npm run build 2>&1 | grep -q "✓\|built"
if [ $? -eq 0 ]; then
  echo "✅ Build succeeds"
else
  echo "❌ Build failed"
  exit 1
fi

echo ""
echo "🎉 Phase 1 Enforcement is ACTIVE"
echo ""
echo "Next: Sign PHASE_1_DECISION_LOG.md and commit"
```

Save as `verify-phase-enforcement.sh`, run with `bash verify-phase-enforcement.sh`

---

## You're Done

Phase 1 is now locked in. The boundaries are:

✅ **Real** (pre-commit hook)  
✅ **Enforced** (GitHub Actions)  
✅ **Audited** (exception log)  
✅ **Enforceable** (branch protection)

No more "hope." Now it's engineering discipline.

Start building products, not architecture.

---

**Reference**:
- Full policy: PHASE_ENFORCEMENT_PLAN.md
- Technical details: ARCHITECTURAL_FREEZE_SPEC.md
- Decision log: PHASE_1_DECISION_LOG.md
- Exception log: docs/PHASE_1_EXCEPTIONS.md

**Questions?** See PHASE_ENFORCEMENT_PLAN.md or ask @tech-lead.
