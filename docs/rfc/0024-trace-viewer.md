# RFC 0024: Trace Viewer UI

**Status**: Draft  
**Date**: 2026-06-04  
**Target**: Phase 2.4

---

## Summary

Visualize session execution trace in real-time. Timeline view of provider decisions, state transitions, and performance metrics.

---

## Motivation

Users need visibility into:
- Which provider was chosen for each mod
- Download progress per mod
- Verification steps
- State transitions

Debug tool for understanding launcher behavior.

---

## Trace Node Structure

```typescript
interface TraceNode {
  id: string;
  timestamp: number;
  type: 'resolve' | 'download' | 'verify' | 'install' | 'complete' | 'error';
  modId: string;
  modName: string;
  
  // Provider info
  provider?: string; // 'LocalCache' | 'Steam' | 'SteamCMD' | 'HTTP'
  providerReason?: string;
  
  // Progress info
  progress?: {
    current: number;
    total: number;
    speed?: number;
  };
  
  // State
  state?: string;
  error?: string;
  
  // Performance
  duration?: number; // ms
}
```

---

## Visual Design

### Timeline View

```
Session: session-123456
├─ [12:34:56.123] mod-a: Brita Weapons
│  ├─ Resolve → LocalCache ✓ (12ms)
│  └─ Complete ✓
│
├─ [12:34:56.456] mod-b: Common Sense
│  ├─ Resolve → Steam (45ms)
│  ├─ Download ████████████░░░ 78% (2.3 MB/s)
│  ├─ Verify SHA256 ✓ (120ms)
│  └─ Complete ✓
│
└─ [12:34:58.789] Session Complete ✓
```

### Tree View

```
Session
├── mod-a (Brita Weapons)
│   ├── Provider: LocalCache ✓
│   └── State: Complete
│
├── mod-b (Common Sense)
│   ├── Provider: Steam
│   ├── Download: 78%
│   └── State: Downloading
│
└── Summary
    ├── Total: 3 mods
    ├── Cached: 1
    ├── Downloaded: 1
    └── Time: 2.3s
```

---

## Components

```typescript
// TraceTimeline.tsx - Main component
interface TraceTimelineProps {
  sessionId: string;
  autoScroll?: boolean;
}

// TraceNode.tsx - Single node
interface TraceNodeProps {
  node: TraceNode;
  depth: number;
  isLast: boolean;
}

// ProviderBadge.tsx - Provider indicator
interface ProviderBadgeProps {
  provider: string;
  reason?: string;
}

// ProgressBar.tsx - Animated progress
interface ProgressBarProps {
  progress: number;
  speed?: number;
  eta?: number;
}
```

---

## Features

1. **Real-time updates** - WebSocket style via Wails events
2. **Auto-scroll** - Keep latest events visible
3. **Filter** - By type, provider, mod
4. **Export** - JSON trace for debugging
5. **Performance metrics** - Duration per operation

---

## Event Mapping

```
launcher:event
├── mod.resolve.start → TraceNode(type: 'resolve')
├── mod.resolve.complete → Update with provider
├── download.progress → Update progress
├── download.complete → Mark complete
├── session.complete → Finalize trace
└── error → Mark error state
```

---

## Open Questions

1. Max trace history per session?
2. Persist traces to disk?
3. Compare traces across sessions?
