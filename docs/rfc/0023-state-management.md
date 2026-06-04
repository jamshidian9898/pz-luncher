# RFC 0023: State Management with Zustand

**Status**: Draft  
**Date**: 2026-06-04  
**Target**: Phase 2.2

---

## Summary

Zustand stores for UI state management. Separate concerns: servers, downloads, sessions, settings, trace.

---

## Store Structure

```typescript
// stores/servers.store.ts
interface ServersState {
  servers: ServerInfo[];
  selectedServer: ServerInfo | null;
  loading: boolean;
  error: string | null;
  
  // Actions
  fetchServers: () => Promise<void>;
  selectServer: (server: ServerInfo | null) => void;
  joinServer: (serverId: string) => Promise<void>;
}

// stores/downloads.store.ts  
interface DownloadsState {
  sessions: Map<string, SessionStatus>;
  activeDownloads: SessionStatus[];
  completedDownloads: SessionStatus[];
  
  // Actions
  updateSession: (session: SessionStatus) => void;
  completeSession: (sessionId: string) => void;
  clearCompleted: () => void;
}

// stores/trace.store.ts
interface TraceState {
  traces: Map<string, TraceNode[]>;
  activeTrace: string | null;
  
  // Actions
  addTraceEvent: (sessionId: string, event: TraceNode) => void;
  setActiveTrace: (sessionId: string | null) => void;
  getTraceForSession: (sessionId: string) => TraceNode[];
}

// stores/settings.store.ts
interface SettingsState {
  settings: Settings | null;
  saving: boolean;
  
  // Actions
  loadSettings: () => Promise<void>;
  saveSettings: (settings: Settings) => Promise<void>;
  updateSetting: (key: keyof Settings, value: any) => void;
}
```

---

## Event to Store Flow

```
Wails Event (launcher:event)
        ↓
useLauncherEvents hook
        ↓
Store action (e.g., downloadsStore.updateSession)
        ↓
Component re-renders with new state
```

---

## Implementation

```bash
cd apps/launcher-ui/frontend
npm install zustand
```

---

## Why Zustand?

- Simple, no boilerplate
- Works with React 18
- No provider needed
- TypeScript native
- DevTools support
