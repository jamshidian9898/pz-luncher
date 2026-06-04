import { useEffect } from 'react';
import { LauncherEvent, LauncherEventType } from '../interfaces/LauncherEvent';
import { eventsApi } from '../wails';
import { useDownloadsStore, DownloadsState } from '../stores/downloads.store';
import { useTraceStore, TraceNode, TraceState } from '../stores/trace.store';
import { useServersStore, ServersState } from '../stores/servers.store';
import { useSessionStore, SessionState, LaunchState } from '../stores/session.store';

/**
 * Phase 2.1: Real Event Integration
 *
 * Connects launcher events to Zustand stores
 * RFC 0022: UI Events & Progress Streaming
 */
export function useLauncherEvents() {
  useEffect(() => {
    const unsubscribe = eventsApi.onLauncherEvent((event: LauncherEvent) => {
      console.log('[LauncherEvents] Received:', event.type, event);

      const downloadsStore = useDownloadsStore.getState();
      const traceStore = useTraceStore.getState();
      const serversStore = useServersStore.getState();
      const sessionStore = useSessionStore.getState();

      switch (event.type) {
        case LauncherEventType.SessionStarted:
          handleSessionStarted(event, downloadsStore, traceStore, sessionStore);
          break;

        case LauncherEventType.SessionCompleted:
          handleSessionCompleted(event, downloadsStore, traceStore, serversStore, sessionStore);
          break;

        case LauncherEventType.DownloadStarted:
          handleDownloadStarted(event, downloadsStore, traceStore);
          break;

        case LauncherEventType.DownloadProgress:
          handleDownloadProgress(event, downloadsStore, traceStore);
          break;

        case LauncherEventType.DownloadCompleted:
          handleDownloadCompleted(event, downloadsStore, traceStore);
          break;

        case LauncherEventType.LaunchStateChanged:
          handleLaunchStateChanged(event, sessionStore);
          break;

        default:
          console.log('[LauncherEvents] Unhandled event type:', event.type);
      }
    });

    return () => {
      unsubscribe();
    };
  }, []);
}

function handleSessionStarted(
  event: LauncherEvent,
  downloadsStore: DownloadsState,
  traceStore: TraceState,
  sessionStore: SessionState
) {
  downloadsStore.updateSession({
    sessionId: event.sessionId,
    state: 'resolving',
    progress: 0,
    currentMod: 'Resolving mods...',
  });

  traceStore.addTraceEvent(event.sessionId, {
    id: `session-start-${event.timestamp}`,
    timestamp: event.timestamp,
    type: 'resolve',
    modId: 'session',
    modName: 'Session Started',
    state: 'resolving',
  });

  traceStore.setActiveTrace(event.sessionId);
  sessionStore.setCurrentSession(event.sessionId);
  sessionStore.setLaunchState('resolving');
}

function handleSessionCompleted(
  event: LauncherEvent,
  downloadsStore: DownloadsState,
  traceStore: TraceState,
  serversStore: ServersState,
  sessionStore: SessionState
) {
  downloadsStore.completeSession(event.sessionId);

  traceStore.addTraceEvent(event.sessionId, {
    id: `session-complete-${event.timestamp}`,
    timestamp: event.timestamp,
    type: 'complete',
    modId: 'session',
    modName: 'Session Completed',
  });

  serversStore.setJoining(false);
  sessionStore.setLaunchState('complete');
}

function handleDownloadStarted(
  event: LauncherEvent,
  downloadsStore: DownloadsState,
  traceStore: TraceState
) {
  const packageId = event.payload?.packageId ?? 'unknown';

  downloadsStore.updateSessionMod(event.sessionId, packageId);

  traceStore.addTraceEvent(event.sessionId, {
    id: `download-start-${event.timestamp}`,
    timestamp: event.timestamp,
    type: 'download',
    modId: packageId,
    modName: packageId,
    state: 'downloading',
    progress: { current: 0, total: 100, percent: 0 },
  });
}

function handleDownloadProgress(
  event: LauncherEvent,
  downloadsStore: DownloadsState,
  traceStore: TraceState
) {
  const progress = event.payload?.progress;
  if (!progress) return;

  downloadsStore.updateSessionProgress(event.sessionId, progress.percent);

  const packageId = event.payload?.packageId ?? 'unknown';
  const traceNodes = traceStore.getTraceForSession(event.sessionId);
  const downloadNode = [...traceNodes].reverse().find(
    (n) => n.type === 'download' && n.modId === packageId
  );

  if (downloadNode) {
    traceStore.updateTraceProgress(event.sessionId, downloadNode.id, progress.percent, progress.speed);
  }
}

function handleDownloadCompleted(
  event: LauncherEvent,
  downloadsStore: DownloadsState,
  traceStore: TraceState
) {
  const packageId = event.payload?.packageId ?? 'unknown';
  const traceNodes = traceStore.getTraceForSession(event.sessionId);
  const downloadNode = traceNodes.find(
    (n: TraceNode) => n.type === 'download' && n.modId === packageId
  );

  if (downloadNode) {
    traceStore.completeTraceNode(event.sessionId, downloadNode.id);
  }

  traceStore.addTraceEvent(event.sessionId, {
    id: `verify-${event.timestamp}`,
    timestamp: event.timestamp,
    type: 'verify',
    modId: packageId,
    modName: packageId,
    state: 'complete',
  });
}

function handleLaunchStateChanged(event: LauncherEvent, sessionStore: SessionState) {
  const state = event.payload?.metadata?.state as LaunchState | undefined;
  if (state) {
    sessionStore.setLaunchState(state);
  }
}
