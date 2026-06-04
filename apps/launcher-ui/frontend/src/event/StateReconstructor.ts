/**
 * StateReconstructor
 * Reconstructs LauncherState from event log for replay and debugging.
 * Enables audit trail verification and state timeline inspection.
 * Supports snapshot-accelerated reconstruction for large event logs.
 */

import { EventLogEntry, useEventLog } from '../stores/eventLog.store';
import { useDownloadsStore } from '../stores/downloads.store';
import { useTraceStore } from '../stores/trace.store';
import { useSessionStore } from '../stores/session.store';
import { useServersStore } from '../stores/servers.store';
import { useSnapshotStore } from '../stores/snapshotStore';
import { SnapshotEngine } from './SnapshotEngine';
import { PerformanceBoundaries } from './PerformanceBoundaries';

export interface StateSnapshot {
  timestamp: number;
  eventCount: number;
  downloads: ReturnType<typeof useDownloadsStore.getState>;
  trace: ReturnType<typeof useTraceStore.getState>;
  session: ReturnType<typeof useSessionStore.getState>;
  servers: ReturnType<typeof useServersStore.getState>;
  reconstructionTime?: number;
}

export class StateReconstructor {
  /**
   * Reconstruct state at a specific point in time from event log
   * Uses snapshots when available for faster reconstruction
   */
  static reconstructAtTimestamp(sessionId: string, targetTimestamp: number): StateSnapshot | null {
    const startReconstructionTime = Date.now();
    const eventLog = useEventLog.getState();
    const snapshotStore = useSnapshotStore.getState();

    // Find all events up to target timestamp
    const allEvents = eventLog.getEntriesBySession(sessionId);
    const targetEventIndex = allEvents.findIndex((e) => e.appliedAt > targetTimestamp);
    const targetEventNumber = targetEventIndex === -1 ? allEvents.length : targetEventIndex;

    // Try to find a snapshot to start from
    const availableSnapshots = snapshotStore.getSnapshotsForSession(sessionId);
    const startingSnapshot = SnapshotEngine.findBestSnapshotForReconstruction(availableSnapshots, targetEventNumber);

    if (!startingSnapshot) {
      // No snapshot, replay from beginning
      return this.reconstructFromEvents(sessionId, 0, targetEventNumber, startReconstructionTime);
    }

    // Restore from snapshot and replay remaining events
    SnapshotEngine.restoreSnapshot(startingSnapshot);
    const result = this.reconstructFromEvents(sessionId, startingSnapshot.eventNumber, targetEventNumber, startReconstructionTime);

    // Check if reconstruction was slow and should create a snapshot
    const reconstructionTime = Date.now() - startReconstructionTime;
    const perfCheck = PerformanceBoundaries.isReconstructionTimeAcceptable(reconstructionTime, targetEventNumber);

    if (perfCheck.shouldSnapshot && availableSnapshots.length < PerformanceBoundaries.MAX_SNAPSHOTS_PER_SESSION) {
      // Opportunistically create snapshot for next time
      const newSnapshot = SnapshotEngine.createSnapshot(sessionId, targetEventNumber);
      snapshotStore.addSnapshot(newSnapshot);
    }

    return result;
  }

  /**
   * Internal: Reconstruct from event range
   */
  private static reconstructFromEvents(
    sessionId: string,
    startEventNumber: number,
    endEventNumber: number,
    startTime: number
  ): StateSnapshot | null {
    const eventLog = useEventLog.getState();
    const entries = eventLog
      .getEntriesBySession(sessionId)
      .filter((e) => e.status === 'applied')
      .sort((a, b) => a.appliedAt - b.appliedAt)
      .slice(startEventNumber, endEventNumber);

    if (entries.length === 0) {
      return null;
    }

    // Replay patches in order
    entries.forEach((entry: EventLogEntry) => {
      this.applyPatchToStores(entry);
    });

    const reconstructionTime = Date.now() - startTime;

    return {
      timestamp: entries[entries.length - 1].appliedAt,
      eventCount: entries.length,
      downloads: useDownloadsStore.getState(),
      trace: useTraceStore.getState(),
      session: useSessionStore.getState(),
      servers: useServersStore.getState(),
      reconstructionTime,
    };
  }

  /**
   * Reconstruct entire session timeline: array of snapshots at key points
   */
  static reconstructSessionTimeline(sessionId: string): StateSnapshot[] {
    const eventLog = useEventLog.getState();
    const entries = eventLog
      .getEntriesBySession(sessionId)
      .filter((e) => e.status === 'applied')
      .sort((a, b) => a.appliedAt - b.appliedAt);

    if (entries.length === 0) {
      return [];
    }

    const timeline: StateSnapshot[] = [];

    // Take snapshot after every 10 events or at the end
    entries.forEach((entry, index) => {
      if (index % 10 === 0 || index === entries.length - 1) {
        const snapshot = this.reconstructAtTimestamp(sessionId, entry.appliedAt);
        if (snapshot) {
          timeline.push(snapshot);
        }
      }
    });

    return timeline;
  }

  /**
   * Compare state between two timestamps to identify what changed
   */
  static diffStates(sessionId: string, startTime: number, endTime: number): Record<string, unknown> {
    const startSnapshot = this.reconstructAtTimestamp(sessionId, startTime);
    const endSnapshot = this.reconstructAtTimestamp(sessionId, endTime);

    if (!startSnapshot || !endSnapshot) {
      return { error: 'Cannot reconstruct states for comparison' };
    }

    const diff: Record<string, unknown> = {};

    // Compare each store state
    if (JSON.stringify(startSnapshot.downloads) !== JSON.stringify(endSnapshot.downloads)) {
      diff.downloads = { before: startSnapshot.downloads, after: endSnapshot.downloads };
    }

    if (JSON.stringify(startSnapshot.trace) !== JSON.stringify(endSnapshot.trace)) {
      diff.trace = { before: startSnapshot.trace, after: endSnapshot.trace };
    }

    if (JSON.stringify(startSnapshot.session) !== JSON.stringify(endSnapshot.session)) {
      diff.session = { before: startSnapshot.session, after: endSnapshot.session };
    }

    if (JSON.stringify(startSnapshot.servers) !== JSON.stringify(endSnapshot.servers)) {
      diff.servers = { before: startSnapshot.servers, after: endSnapshot.servers };
    }

    return diff;
  }

  /**
   * Find the event that caused a state anomaly
   */
  static findAnomalyEvent(
    sessionId: string,
    anomalyCheck: (snapshot: StateSnapshot) => boolean
  ): EventLogEntry | null {
    const eventLog = useEventLog.getState();
    const entries = eventLog
      .getEntriesBySession(sessionId)
      .filter((e) => e.status === 'applied')
      .sort((a, b) => a.appliedAt - b.appliedAt);

    for (const entry of entries) {
      const snapshot = this.reconstructAtTimestamp(sessionId, entry.appliedAt);
      if (snapshot && anomalyCheck(snapshot)) {
        return entry;
      }
    }

    return null;
  }

  /**
   * Internal: Apply a patch to stores for reconstruction
   */
  private static applyPatchToStores(entry: EventLogEntry): void {
    const { patch } = entry;

    const downloadsStore = useDownloadsStore.getState();
    const traceStore = useTraceStore.getState();
    const sessionStore = useSessionStore.getState();
    const serversStore = useServersStore.getState();

    // Apply downloads patch
    if (patch.downloads) {
      const p = patch.downloads;
      if (p.sessionUpdate) downloadsStore.updateSession(p.sessionUpdate);
      if (p.completeSessionId) downloadsStore.completeSession(p.completeSessionId);
      if (p.failSession) downloadsStore.failSession(p.failSession.sessionId, p.failSession.error);
    }

    // Apply trace patch
    if (patch.trace) {
      const p = patch.trace;
      if (p.activeTrace !== undefined) traceStore.setActiveTrace(p.activeTrace);
      if (p.addEvents?.length) {
        p.addEvents.forEach((node: any) => {
          traceStore.addTraceEvent(entry.sessionId, {
            ...node,
            id: `${entry.event.type}-${node.modId}-${entry.event.timestamp}`,
            timestamp: entry.event.timestamp,
          });
        });
      }
      if (p.completeNode) {
        const packageId = p.completeNode.packageId;
        const traceNodes = traceStore.getTraceForSession(entry.sessionId);
        const node = traceNodes.find((tn) => tn.type === 'download' && tn.modId === packageId);
        if (node) {
          traceStore.completeTraceNode(entry.sessionId, node.id);
        }
      }
    }

    // Apply session patch
    if (patch.session) {
      const p = patch.session;
      if (p.resetSession) sessionStore.resetSession();
      if (p.currentSessionId !== undefined) sessionStore.setCurrentSession(p.currentSessionId);
      if (p.launchState) sessionStore.setLaunchState(p.launchState);
      if (p.currentServer !== undefined) sessionStore.setCurrentServer(p.currentServer);
    }

    // Apply servers patch
    if (patch.servers) {
      const p = patch.servers;
      if (p.joining !== undefined) serversStore.setJoining(p.joining);
    }
  }
}
