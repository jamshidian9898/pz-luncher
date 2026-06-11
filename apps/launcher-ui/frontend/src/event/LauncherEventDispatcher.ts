import { LauncherEvent } from '../interfaces/LauncherEvent';
import { useDownloadsStore } from '../stores/downloads.store';
import { useTraceStore } from '../stores/trace.store';
import { useSessionStore } from '../stores/session.store';
import { useServersStore } from '../stores/servers.store';
import { usePatchFailureLog } from '../stores/patchFailureLog.store';
import { useEventLog } from '../stores/eventLog.store';
import { reduceLauncherEvent, validateLauncherEventPatch, LauncherEventPatch, TraceNodePayload } from './LauncherStateReducer';

function addTraceEvents(sessionId: string, event: LauncherEvent, nodes: TraceNodePayload[]) {
  const traceStore = useTraceStore.getState();
  nodes.forEach((node) => {
    traceStore.addTraceEvent(sessionId, {
      ...node,
      id: `${event.type}-${node.modId}-${event.timestamp}`,
      timestamp: event.timestamp,
    });
  });
}

function applyTracePatch(sessionId: string, patch: LauncherEventPatch['trace'], event: LauncherEvent) {
  const traceStore = useTraceStore.getState();
  const traceNodes = traceStore.getTraceForSession(sessionId);

  if (!patch) return;

  if (patch.activeTrace !== undefined) {
    traceStore.setActiveTrace(patch.activeTrace);
  }

  if (patch.addEvents?.length) {
    addTraceEvents(sessionId, event, patch.addEvents);
  }

  if (patch.updateEventProgress) {
    const packageId = patch.updateEventProgress.packageId;
    const node = [...traceNodes].reverse().find((traceNode) => traceNode.type === 'download' && traceNode.modId === packageId);
    if (node) {
      traceStore.updateTraceProgress(
        sessionId,
        node.id,
        patch.updateEventProgress.progress,
        patch.updateEventProgress.speed
      );
    }
  }

  if (patch.completeNode) {
    const packageId = patch.completeNode.packageId;
    const node = traceNodes.find((traceNode) => traceNode.type === 'download' && traceNode.modId === packageId);
    if (node) {
      traceStore.completeTraceNode(sessionId, node.id);
    }
  }
}

function applyDownloadsPatch(patch: LauncherEventPatch['downloads']) {
  const downloadsStore = useDownloadsStore.getState();
  if (!patch) return;

  if (patch.sessionUpdate) {
    downloadsStore.updateSession(patch.sessionUpdate);
  }

  if (patch.completeSessionId) {
    downloadsStore.completeSession(patch.completeSessionId);
  }

  if (patch.failSession) {
    downloadsStore.failSession(patch.failSession.sessionId, patch.failSession.error);
  }
}

function applySessionPatch(patch: LauncherEventPatch['session']) {
  const sessionStore = useSessionStore.getState();
  if (!patch) return;

  if (patch.resetSession) {
    sessionStore.resetSession();
  }

  if (patch.currentSessionId !== undefined) {
    sessionStore.setCurrentSession(patch.currentSessionId);
  }

  if (patch.launchState) {
    sessionStore.setLaunchState(patch.launchState);
  }

  if (patch.lastError !== undefined) {
    sessionStore.setLastError(patch.lastError);
  }

  if (patch.currentServer !== undefined) {
    sessionStore.setCurrentServer(patch.currentServer);
  }
}

function applyServersPatch(patch: LauncherEventPatch['servers']) {
  const serversStore = useServersStore.getState();
  if (!patch) return;

  if (patch.joining !== undefined) {
    serversStore.setJoining(patch.joining);
  }
}

export function dispatchLauncherEvent(event: LauncherEvent) {
  const downloadsStore = useDownloadsStore.getState();
  const eventLog = useEventLog.getState();
  const failureLog = usePatchFailureLog.getState();

  const currentSession = downloadsStore.getSession(event.sessionId);
  const patch = reduceLauncherEvent(event, currentSession);
  const validationErrors = validateLauncherEventPatch(patch);
  const timestamp = Date.now();

  // Log the event regardless of validation status
  if (validationErrors.length > 0) {
    // Event was rejected due to validation errors
    console.warn('[LauncherEventDispatcher] Invalid LauncherEventPatch, skipping apply', {
      event,
      patch,
      validationErrors,
    });

    // Record failure
    failureLog.addFailure({
      eventId: `event-${event.timestamp}`,
      eventType: event.type,
      domain: 'trace', // This would need to be determined better
      reason: validationErrors,
      payload: patch,
      timestamp,
      sessionId: event.sessionId,
    });

    // Log as rejected event
    eventLog.addEntry({
      event,
      patch,
      validationErrors,
      appliedAt: timestamp,
      status: 'rejected',
      sessionId: event.sessionId,
    });

    return;
  }

  // When a real session.start arrives, remove any optimistic placeholder session
  if (event.type === 'session.start' && event.sessionId) {
    const allSessions = useDownloadsStore.getState().sessions;
    for (const [id] of allSessions) {
      if (id.startsWith('optimistic-')) {
        useDownloadsStore.getState().removeSession(id);
      }
    }
  }

  // Apply patches
  applyDownloadsPatch(patch.downloads);
  applyTracePatch(event.sessionId, patch.trace, event);
  applySessionPatch(patch.session);
  applyServersPatch(patch.servers);

  // Log as successfully applied
  eventLog.addEntry({
    event,
    patch,
    validationErrors: [],
    appliedAt: timestamp,
    status: 'applied',
    sessionId: event.sessionId,
  });
}
