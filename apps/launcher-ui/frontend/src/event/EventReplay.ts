/**
 * EventReplay Engine
 * Replays events from the log to reconstruct state or test scenarios.
 * Critical for debugging, auditing, and state verification.
 */

import { useEventLog, EventLogEntry } from '../stores/eventLog.store';
import { dispatchLauncherEvent } from './LauncherEventDispatcher';
import { useDownloadsStore } from '../stores/downloads.store';
import { useTraceStore } from '../stores/trace.store';
import { useSessionStore } from '../stores/session.store';
import { useServersStore } from '../stores/servers.store';

export interface ReplayOptions {
  startTime?: number;
  endTime?: number;
  onlyApplied?: boolean;
  skipValidation?: boolean;
  printStats?: boolean;
}

export interface ReplayResult {
  eventsReplayed: number;
  eventsSkipped: number;
  duration: number;
  startTime: number;
  endTime: number;
  errors: string[];
}

export class EventReplay {
  /**
   * Replay events for a specific session
   */
  static replaySession(sessionId: string, options: ReplayOptions = {}): ReplayResult {
    const startTime = Date.now();
    const eventLog = useEventLog.getState();

    let entries = eventLog.getEntriesBySession(sessionId);

    // Filter by time range if provided
    if (options.startTime || options.endTime) {
      entries = entries.filter((e) => {
        const afterStart = !options.startTime || e.appliedAt >= options.startTime;
        const beforeEnd = !options.endTime || e.appliedAt <= options.endTime;
        return afterStart && beforeEnd;
      });
    }

    // Filter by status if requested
    if (options.onlyApplied) {
      entries = entries.filter((e) => e.status === 'applied');
    }

    // Sort by timestamp to ensure proper order
    entries.sort((a, b) => a.appliedAt - b.appliedAt);

    const errors: string[] = [];
    let replayed = 0;
    let skipped = 0;

    // Clear stores before replay
    this.clearAllStores();

    // Replay each event
    entries.forEach((entry, index) => {
      try {
        // Re-dispatch the event using the original reducer/dispatcher
        dispatchLauncherEvent(entry.event);
        replayed++;
      } catch (error) {
        const errorMsg = error instanceof Error ? error.message : String(error);
        errors.push(`Event ${index} (${entry.event.type}): ${errorMsg}`);
        skipped++;
      }
    });

    const endTime = Date.now();
    const duration = endTime - startTime;

    if (options.printStats) {
      console.log('[EventReplay]', {
        sessionId,
        eventsReplayed: replayed,
        eventsSkipped: skipped,
        durationMs: duration,
      });
    }

    return {
      eventsReplayed: replayed,
      eventsSkipped: skipped,
      duration,
      startTime: entries[0]?.appliedAt || 0,
      endTime: entries[entries.length - 1]?.appliedAt || 0,
      errors,
    };
  }

  /**
   * Replay all events across all sessions
   */
  static replayAll(options: ReplayOptions = {}): ReplayResult {
    const eventLog = useEventLog.getState();
    const allEntries = eventLog.entries;

    const groupBySession = new Map<string, EventLogEntry[]>();
    allEntries.forEach((entry) => {
      if (!groupBySession.has(entry.sessionId)) {
        groupBySession.set(entry.sessionId, []);
      }
      groupBySession.get(entry.sessionId)!.push(entry);
    });

    let totalReplayed = 0;
    let totalSkipped = 0;
    let totalErrors: string[] = [];
    let minTime = Infinity;
    let maxTime = 0;

    groupBySession.forEach((_, sessionId) => {
      const result = this.replaySession(sessionId, { ...options, printStats: false });
      totalReplayed += result.eventsReplayed;
      totalSkipped += result.eventsSkipped;
      totalErrors = totalErrors.concat(result.errors);
      minTime = Math.min(minTime, result.startTime);
      maxTime = Math.max(maxTime, result.endTime);
    });

    const duration = Date.now() - Date.now();

    if (options.printStats) {
      console.log('[EventReplay] All sessions', {
        sessionsReplayed: groupBySession.size,
        eventsReplayed: totalReplayed,
        eventsSkipped: totalSkipped,
        durationMs: duration,
        errorCount: totalErrors.length,
      });
    }

    return {
      eventsReplayed: totalReplayed,
      eventsSkipped: totalSkipped,
      duration,
      startTime: minTime,
      endTime: maxTime,
      errors: totalErrors,
    };
  }

  /**
   * Replay events step-by-step for debugging
   */
  static *replaySessionStepwise(sessionId: string): Generator<EventLogEntry, void, void> {
    const eventLog = useEventLog.getState();
    const entries = eventLog
      .getEntriesBySession(sessionId)
      .sort((a, b) => a.appliedAt - b.appliedAt);

    // Clear stores
    this.clearAllStores();

    for (const entry of entries) {
      try {
        dispatchLauncherEvent(entry.event);
        yield entry;
      } catch (error) {
        console.error(`[EventReplay] Failed to replay event:`, entry.event, error);
      }
    }
  }

  /**
   * Verify state consistency by replaying and checking invariants
   */
  static verifyStateConsistency(sessionId: string): { isValid: boolean; violations: string[] } {
    const eventLog = useEventLog.getState();
    const entries = eventLog.getEntriesBySession(sessionId).sort((a, b) => a.appliedAt - b.appliedAt);

    if (entries.length === 0) {
      return { isValid: true, violations: [] };
    }

    // Clear and replay
    this.clearAllStores();
    entries.forEach((entry) => {
      dispatchLauncherEvent(entry.event);
    });

    // Check invariants
    const violations: string[] = [];
    const sessionStore = useSessionStore.getState();
    const downloadsStore = useDownloadsStore.getState();
    const traceStore = useTraceStore.getState();

    // Invariant 1: Current session should match a download session if set
    if (sessionStore.currentSessionId) {
      const downloadSession = downloadsStore.getSession(sessionStore.currentSessionId);
      if (!downloadSession) {
        violations.push(`Current session ${sessionStore.currentSessionId} not found in downloads store`);
      }
    }

    // Invariant 2: Active trace should exist if set
    if (traceStore.activeTrace) {
      const traces = traceStore.getTraceForSession(sessionId);
      const activeExists = traces.some((t) => t.id === traceStore.activeTrace);
      if (!activeExists) {
        violations.push(`Active trace ${traceStore.activeTrace} not found in trace log`);
      }
    }

    // Invariant 3: Launch state should be valid
    const validStates = ['idle', 'resolving', 'downloading', 'installing', 'verifying', 'materializing', 'launching', 'running', 'complete', 'error'];
    if (!validStates.includes(sessionStore.launchState)) {
      violations.push(`Invalid launch state: ${sessionStore.launchState}`);
    }

    return {
      isValid: violations.length === 0,
      violations,
    };
  }

  /**
   * Internal: Clear all stores for fresh replay
   */
  private static clearAllStores(): void {
    const downloadsStore = useDownloadsStore.getState();
    const traceStore = useTraceStore.getState();
    const sessionStore = useSessionStore.getState();
    const serversStore = useServersStore.getState();

    // Clear each store properly
    downloadsStore.clearCompleted();
    sessionStore.resetSession();
    serversStore.setJoining(false);
  }
}
