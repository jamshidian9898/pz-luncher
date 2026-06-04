# RFC 0022: UI Events & Progress Streaming

**Status**: Draft  
**Date**: 2026-06-04  
**Author**: pzlauncher team  
**Target**: Phase 2 - Consumer Layer

---

## Summary

Define event system for UI updates from launcher-core to Wails frontend. Real-time progress, state changes, and completion notifications.

---

## Motivation

UI needs real-time updates during:
- Server join (mod resolution)
- Download progress
- Installation status
- Error states

Without streaming events, UI would poll or hang.

---

## Design

### Event Types

```go
// Core event types
type UIEventType string

const (
    EventModResolveStart    UIEventType = "mod.resolve.start"
    EventModResolveComplete UIEventType = "mod.resolve.complete"
    EventDownloadStart      UIEventType = "download.start"
    EventDownloadProgress   UIEventType = "download.progress"
    EventDownloadComplete   UIEventType = "download.complete"
    EventInstallStart       UIEventType = "install.start"
    EventInstallComplete    UIEventType = "install.complete"
    EventError              UIEventType = "error"
    EventSessionComplete    UIEventType = "session.complete"
)

// Event payload
type UIEvent struct {
    Type      UIEventType `json:"type"`
    Timestamp int64       `json:"timestamp"`
    SessionID string      `json:"sessionId"`
    PackageID string      `json:"packageId,omitempty"`
    Progress  *Progress   `json:"progress,omitempty"`
    Error     string      `json:"error,omitempty"`
    Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type Progress struct {
    Current int64 `json:"current"`
    Total   int64 `json:"total"`
    Percent int   `json:"percent"`
    Speed   int64 `json:"speed,omitempty"` // bytes/sec
    ETA     int   `json:"eta,omitempty"` // seconds
}
```

---

## Wails Integration

```go
// App struct with event emitter
type App struct {
    ctx    context.Context
    events chan UIEvent
}

// JS callable
func (a *App) JoinServer(serverID string) {
    // Start session
    go func() {
        for event := range sessionEvents {
            wails.EventsEmit(a.ctx, "launcher:event", event)
        }
    }()
}

// Frontend subscribes:
// window.runtime.EventsOn("launcher:event", (event) => { ... })
```

---

## React Hook

```typescript
// useLauncherEvents.ts
export function useLauncherEvents() {
  const [events, setEvents] = useState<UIEvent[]>([]);
  
  useEffect(() => {
    const unsubscribe = EventsOn("launcher:event", (event: UIEvent) => {
      setEvents(prev => [...prev, event]);
    });
    return unsubscribe;
  }, []);
  
  return events;
}
```

---

## Event Flow

```
User clicks [Join Server]
        ↓
Wails: JoinServer(serverID)
        ↓
launcher-core.CreateSession()
        ↓
Event: mod.resolve.start
        ↓
Resolver resolves mods
        ↓
Event: mod.resolve.complete
        ↓
For each missing mod:
  Event: download.start
        ↓
Event: download.progress (streaming)
        ↓
Event: download.complete
        ↓
Event: session.complete
        ↓
UI shows [Play] button
```

---

## Open Questions

1. Buffer size for event channel?
2. Event persistence for reconnection?
3. Backpressure if UI is slow?

---

## Decision

**Proceed**: Wails EventsEmit + React hooks pattern.  
Buffer: 100 events, drop old if full.
