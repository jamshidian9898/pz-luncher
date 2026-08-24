# ARCHITECTURAL DECISION: Product vs Platform

> **✅ RESOLVED** — **Path A (Product)** locked in [PRODUCT_DECISION.md](PRODUCT_DECISION.md).  
> Execute [docs/PRODUCT_ROADMAP.md](docs/PRODUCT_ROADMAP.md) and domain RFCs 0030–0035.  
> This document is kept for historical context only.

**Date**: June 4, 2026  
**Status**: ~~Under Review~~ **Superseded by PRODUCT_DECISION.md**  
**Priority**: ~~Critical~~ Reference only

---

## Executive Summary

The launcher-ui frontend has evolved from a simple React application into an **event-sourced runtime engine** with production-grade optimization layers. This decision document clarifies two fundamentally different paths forward:

1. **Path A: Product Mode** — Complete launcher, use runtime infrastructure as enabler
2. **Path B: Platform Mode** — Expand runtime into general-purpose event system

**Action Required**: Choose one path to guide all future architecture decisions.

---

## Current System Assessment

### What Exists (RFC-0024 + RFC-0025)

```
✅ Event-driven state machine
✅ Deterministic replay engine
✅ Full audit trail via EventLog
✅ Observable patch validation
✅ Snapshot-based acceleration (O(log n))
✅ Intelligent event compaction (20%+ savings)
✅ Explicit performance boundaries
✅ Memory governance (<10MB/session)
```

### Sophistication vs Domain Mismatch

| Dimension | Actual | Launcher Need | Gap |
|-----------|--------|---------------|-----|
| **Architecture** | Enterprise-grade | Medium | High |
| **Complexity** | Very High | Medium | High |
| **Optimization depth** | Database-level | App-level | High |
| **Feature count** | 15+ systems | 3-5 systems | High |

### The Real Question

```
We have a database-like engine.
We're building a launcher.

Is this intentional architecture or scope creep?
```

---

## Path A: Product Completion Mode

### Vision
> "Use the infrastructure wisely, complete the launcher, ship to users"

### Architecture
```
Runtime Infrastructure (RFC-0024/0025)
           ↓
    Launcher Features
           ↓
        UI/UX Polish
           ↓
      User-Facing Product
```

### What Gets Built (12-16 weeks)

#### Phase 1: Core Launcher Flow (6 weeks)
- Mod discovery & search UI
- Download queue management
- Installation progress visualization
- Game launch orchestration
- Basic error handling & recovery

#### Phase 2: User Experience (4 weeks)
- Mod organization & favorites
- Profile management
- Settings & preferences
- Theme / accessibility
- Performance monitoring UI

#### Phase 3: Reliability (2 weeks)
- End-to-end testing
- Error scenario coverage
- Performance tuning
- Documentation & help

#### Phase 4: Launch (4 weeks)
- Closed beta
- Feedback collection
- Bug fixes
- Release preparation

### Deliverables
- ✅ Functional launcher application
- ✅ User documentation
- ✅ Admin guides
- ✅ Performance metrics dashboard
- ✅ Bug tracking system

### Advantages
- ✅ Clear product timeline
- ✅ Users get value in 3-4 months
- ✅ Justifies infrastructure investment
- ✅ Infrastructure proven in production
- ✅ Foundation for Phase 2 features

### Disadvantages
- ❌ Infrastructure overcomplicated for launcher needs
- ❌ High learning curve for team
- ❌ Difficult to explain to stakeholders
- ❌ Maintenance burden for one-off features

### Metrics of Success
```
- Launch to users: ✓
- Average session: 2+ hours
- Mod download success: >95%
- User retention: >60% day 2
- Mean time to restore: <5 min
```

---

## Path B: Platform Expansion Mode

### Vision
> "Build an event-sourced platform for game launcher ecosystems, with launcher as proof-of-concept"

### Architecture
```
Event-Sourced Runtime Platform
    ├─ Launcher (UI instance)
    ├─ Mod Manager (service)
    ├─ Game Server (service)
    ├─ Analytics (consumer)
    └─ Plugin System (extensible)
```

### What Gets Built (20-24 weeks)

#### Phase 1: Multi-Instance Infrastructure (6 weeks)
- Service-to-service event streaming
- Distributed snapshot system
- Cross-instance replay
- Event versioning & migration
- Remote state reconstruction

#### Phase 2: Launcher as Service (4 weeks)
- Decompose UI into services
- Event producer/consumer architecture
- Plugin interface definition
- Service discovery & coordination
- Inter-service protocols

#### Phase 3: Plugin Ecosystem (6 weeks)
- Plugin SDK & examples
- Custom event types
- Snapshot extension points
- Validation plugin interface
- Marketplace / registry

#### Phase 4: Advanced Features (4 weeks)
- Distributed transactions
- Multi-player sessions
- Collaborative features
- Analytics aggregation
- Debugger tooling

#### Phase 5: Platform Hardening (4 weeks)
- Security model
- Rate limiting & quotas
- Resource governance
- SLA & monitoring
- Operations playbooks

### Deliverables
- ✅ Open-source platform framework
- ✅ Launcher as reference implementation
- ✅ Plugin SDK with examples
- ✅ Docker deployment templates
- ✅ Operations guide
- ✅ API documentation
- ✅ Community onboarding

### Advantages
- ✅ Reusable by other game/app projects
- ✅ Strong differentiation in market
- ✅ Potential licensing/commercial opportunities
- ✅ Attracts engineering talent
- ✅ Proves event-sourcing at scale

### Disadvantages
- ❌ 6+ month timeline to platform readiness
- ❌ Launcher users wait longer
- ❌ Higher complexity = more bugs
- ❌ Requires platform thinking (not game-centric)
- ❌ Larger team needed

### Metrics of Success
```
- Platform adoption: 3+ external projects
- Plugin count: 10+ community plugins
- Event throughput: 10k+ events/sec
- Multi-instance deployments: 5+ active
- Developer satisfaction: >4/5 stars
```

---

## Comparison Matrix

| Aspect | Path A (Product) | Path B (Platform) |
|--------|-----------------|-----------------|
| **Timeline to ship** | 3-4 months | 6+ months |
| **User value delivery** | Fast | Slow |
| **Infrastructure reuse** | None | 20+ projects |
| **Complexity** | High (justified) | Very High (justified) |
| **Team size needed** | 3-4 | 6-8 |
| **Revenue potential** | Direct (users) | Licensing/platform |
| **Risk** | Medium | High |
| **Market differentiation** | Launcher | Event platform |
| **Maintenance burden** | Medium | High |

---

## Hidden Trade-offs

### Path A Downsides
```
Why build database-level infrastructure for 1 launcher?

→ Team complexity
→ Onboarding difficulty  
→ Over-engineered for domain
→ Hard to justify to stakeholders
→ Potential future debt
```

### Path B Downsides
```
Why delay launcher for platform infrastructure?

→ Launcher users lose
→ Higher failure risk
→ Requires platform expertise (different skillset)
→ Unproven market for "launcher platform"
→ Competition from existing platforms
```

---

## Decision Framework

### Choose Path A IF...

- ✅ Primary goal is **user-facing launcher**
- ✅ Timeline matters (quarters, not years)
- ✅ Team prefers shipping over architecture
- ✅ Budget/runway is limited
- ✅ Want to prove infrastructure in real usage first
- ✅ Game launcher is primary business

### Choose Path B IF...

- ✅ Primary goal is **build a platform**
- ✅ Timeline is flexible (6+ months OK)
- ✅ Team wants long-term architecture play
- ✅ Budget/runway allows for exploration
- ✅ Want to prove business model before users
- ✅ Launcher is proof-of-concept, not the business

---

## Recommendation: The Hybrid Approach

### "Path A + B Progressive"

**Start with Path A, unlock Path B**

```
Months 1-4: Path A
- Complete launcher (users ship)
- Prove infrastructure in production
- Collect real metrics

Months 5-8: Path A + B
- Extract platform from launcher
- Maintain launcher excellence
- Build plugin system

Months 9+: Path B
- Platform as primary focus
- Launcher as reference app
- Ecosystem growth
```

### Why This Works

1. **Users get launcher fast** (4 months)
2. **Infrastructure proven** (real usage data)
3. **Path forward clear** (don't pivot mid-stream)
4. **Risk reduced** (iterate based on learnings)
5. **Team grows naturally** (scale as needed)

### Transition Points

```
Path A → Path B requires:
- ✅ Launcher shipping (proof of architecture)
- ✅ Real user metrics (10k+ events data)
- ✅ Stability achieved (99%+ uptime)
- ✅ Team scaling (add platform engineers)
- ✅ Platform funding (business model)
```

---

## Current State Impact

### What This Decision Changes

**Path A**: 
- Simplify non-critical systems
- Focus on launcher UX/features
- Timeline: Ready in 4 months

**Path B**:
- Double down on infrastructure
- Build service boundaries now
- Timeline: Ready in 8 months

**Hybrid**:
- Build launcher features in Phase 1 (3-4 months)
- Extract infrastructure in Phase 2 (2-3 months)
- Total: 6 months to launcher + platform foundation

### Files/RFCs Affected

**Path A**: 
- EventLogDebugPanel becomes developer-only tool
- SnapshotStore remains internal optimization
- RFCs 0024/0025 are "implementation details"

**Path B**:
- EventLogDebugPanel becomes user-facing feature
- SnapshotStore becomes public API
- RFCs 0024/0025 become "core specifications"
- Need new RFCs: 0026 (service architecture), 0027 (plugin system)

**Hybrid**:
- Phase 1: Treat as Path A
- Phase 2: Refactor per Path B requirements
- Keep RFCs flexible (versioning/migration)

---

## Next Steps (Required)

### Week 1: Decision
```
[ ] Stakeholder alignment meeting
[ ] Technical team consensus
[ ] Document decision in DECISION.md
[ ] Update all RFCs with path clarity
```

### Week 2: Implementation Roadmap
```
If Path A:
  [ ] Create 12-week product roadmap
  [ ] Assign UX/Feature team
  [ ] Simplify infrastructure (remove non-essentials)

If Path B:
  [ ] Create 24-week platform roadmap
  [ ] Assign platform/infrastructure team
  [ ] Design service boundaries
  [ ] API contract definitions

If Hybrid:
  [ ] Phase 1 roadmap (launcher, 4 weeks)
  [ ] Phase 2 roadmap (extraction, 2 weeks)
  [ ] Platform roadmap (6+ weeks)
```

### Week 3: Kickoff
```
[ ] Team training / alignment
[ ] Development environment setup
[ ] First feature implementation starts
```

---

## Appendix: Code/Docs Impact

### Files That Change Based on Decision

**Product Mode Changes**:
- `docs/rfc/0024-event-log-replay.md` → Mark as "internal implementation"
- `docs/rfc/0025-snapshot-compaction.md` → Mark as "performance optimization"
- `src/event/EventCompaction.ts` → Hide/simplify UI exposure
- `src/components/EventLogDebugPanel.tsx` → Dev-only component
- New: `docs/PRODUCT_ROADMAP.md`

**Platform Mode Changes**:
- `docs/rfc/0024-event-log-replay.md` → Core specification
- `docs/rfc/0025-snapshot-compaction.md` → Core specification
- New: `docs/rfc/0026-service-architecture.md`
- New: `docs/rfc/0027-plugin-system.md`
- New: `src/platform/` (new directory)
- New: `docs/PLATFORM_ROADMAP.md`
- New: `docs/API_SPECIFICATION.md`

**Hybrid Mode Changes**:
- Phase 1: Same as Product Mode
- Phase 2: Gradual transition
- New: `docs/HYBRID_ROADMAP_PHASE1_2.md`
- Versioned RFCs: `0024-v1`, `0024-v2` (after extraction)

---

## Decision Log

- **Decision Date**: [To be filled]
- **Decision**: Path A / Path B / Hybrid
- **Rationale**: [To be documented]
- **Stakeholders**: [List involved]
- **Sign-off**: [Required signatures]

---

**This decision determines the next 6+ months of architecture. Choose wisely.**
