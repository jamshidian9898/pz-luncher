/**
 * PerformanceBoundaries
 * Defines system constraints and performance budgets for the event-sourced runtime.
 * Ensures predictable behavior under load and guides optimization decisions.
 */

export interface PerformanceMetrics {
  eventLogSize: number;
  snapshotCount: number;
  replayDuration: number;
  validationOverhead: number;
  memoryUsage: number;
}

export interface PerformanceBoundary {
  name: string;
  limit: number;
  unit: string;
  warning: number;
  severity: 'info' | 'warning' | 'critical';
}

export class PerformanceBoundaries {
  /**
   * Maximum events in single session's EventLog
   * Beyond this, older events should be compacted
   */
  static readonly MAX_EVENTS_PER_SESSION = 1000;

  /**
   * Warning threshold for event log size (80% of max)
   */
  static readonly EVENTS_WARNING_THRESHOLD = 800;

  /**
   * Maximum patch failures to keep in memory
   */
  static readonly MAX_FAILURES_PER_SESSION = 500;

  /**
   * Maximum snapshots per session
   * Older snapshots are compacted
   */
  static readonly MAX_SNAPSHOTS_PER_SESSION = 20;

  /**
   * Snapshot interval (create snapshot every N events)
   * Balances reconstruction speed vs. memory usage
   */
  static readonly SNAPSHOT_INTERVAL = 100;

  /**
   * Maximum time (ms) allowed for reconstruction without snapshot
   * If replay would exceed this, force snapshot creation
   */
  static readonly MAX_RECONSTRUCTION_TIME_MS = 500;

  /**
   * Maximum time (ms) allowed for event replay per event
   * If an event takes longer, log warning
   */
  static readonly MAX_EVENT_REPLAY_TIME_MS = 10;

  /**
   * Maximum time (ms) allowed for patch validation per patch
   */
  static readonly MAX_VALIDATION_TIME_MS = 5;

  /**
   * Target memory usage for event log (MB)
   * If exceeded, trigger compaction
   */
  static readonly TARGET_EVENTLOG_MEMORY_MB = 2;

  /**
   * Target memory usage for all snapshots (MB)
   */
  static readonly TARGET_SNAPSHOT_MEMORY_MB = 3;

  /**
   * Maximum total overhead (event log + snapshots + failures)
   */
  static readonly MAX_TOTAL_MEMORY_MB = 10;

  /**
   * Event compression: aggregate events after this many redundant operations
   */
  static readonly REDUNDANCY_THRESHOLD = 5;

  /**
   * Minimum compaction savings (%) before triggering
   * Only compact if we can save at least this much space
   */
  static readonly MIN_COMPACTION_SAVINGS_PERCENT = 20;

  /**
   * Check if event log size is acceptable
   */
  static isEventLogSizeAcceptable(eventCount: number, sessionId?: string): {
    ok: boolean;
    status: 'ok' | 'warning' | 'critical';
    message: string;
  } {
    if (eventCount >= this.MAX_EVENTS_PER_SESSION) {
      return {
        ok: false,
        status: 'critical',
        message: `Event log for ${sessionId} exceeded max (${eventCount}/${this.MAX_EVENTS_PER_SESSION}). Compaction required.`,
      };
    }

    if (eventCount >= this.EVENTS_WARNING_THRESHOLD) {
      return {
        ok: true,
        status: 'warning',
        message: `Event log for ${sessionId} at ${((eventCount / this.MAX_EVENTS_PER_SESSION) * 100).toFixed(1)}% capacity. Consider compaction.`,
      };
    }

    return {
      ok: true,
      status: 'ok',
      message: `Event log for ${sessionId} size acceptable (${eventCount} events).`,
    };
  }

  /**
   * Check if reconstruction time is acceptable
   */
  static isReconstructionTimeAcceptable(
    durationMs: number,
    eventCount: number
  ): {
    ok: boolean;
    status: 'ok' | 'warning' | 'critical';
    message: string;
    shouldSnapshot: boolean;
  } {
    const shouldSnapshot = durationMs > this.MAX_RECONSTRUCTION_TIME_MS || eventCount > this.SNAPSHOT_INTERVAL;

    if (durationMs > this.MAX_RECONSTRUCTION_TIME_MS * 2) {
      return {
        ok: false,
        status: 'critical',
        message: `Reconstruction took ${durationMs}ms (${(durationMs / eventCount).toFixed(2)}ms per event). Performance degraded.`,
        shouldSnapshot: true,
      };
    }

    if (durationMs > this.MAX_RECONSTRUCTION_TIME_MS) {
      return {
        ok: true,
        status: 'warning',
        message: `Reconstruction took ${durationMs}ms. Consider snapshots.`,
        shouldSnapshot: true,
      };
    }

    return {
      ok: true,
      status: 'ok',
      message: `Reconstruction took ${durationMs}ms for ${eventCount} events.`,
      shouldSnapshot: false,
    };
  }

  /**
   * Check if memory usage is acceptable
   */
  static isMemoryUsageAcceptable(
    eventLogMB: number,
    snapshotsMB: number,
    failuresMB: number
  ): {
    ok: boolean;
    status: 'ok' | 'warning' | 'critical';
    message: string;
    breakdown: string;
  } {
    const totalMB = eventLogMB + snapshotsMB + failuresMB;
    const breakdown = `EventLog: ${eventLogMB.toFixed(2)}MB | Snapshots: ${snapshotsMB.toFixed(2)}MB | Failures: ${failuresMB.toFixed(2)}MB | Total: ${totalMB.toFixed(2)}MB`;

    if (totalMB > this.MAX_TOTAL_MEMORY_MB) {
      return {
        ok: false,
        status: 'critical',
        message: `Total memory usage ${totalMB.toFixed(2)}MB exceeded limit (${this.MAX_TOTAL_MEMORY_MB}MB). Compaction urgent.`,
        breakdown,
      };
    }

    if (eventLogMB > this.TARGET_EVENTLOG_MEMORY_MB || snapshotsMB > this.TARGET_SNAPSHOT_MEMORY_MB) {
      return {
        ok: true,
        status: 'warning',
        message: `Memory usage approaching limits. Consider compaction.`,
        breakdown,
      };
    }

    return {
      ok: true,
      status: 'ok',
      message: `Memory usage within limits.`,
      breakdown,
    };
  }

  /**
   * Should create a snapshot based on event count
   */
  static shouldCreateSnapshot(eventsSinceLastSnapshot: number, forceIfExceeds: boolean = false): boolean {
    if (forceIfExceeds && eventsSinceLastSnapshot > this.SNAPSHOT_INTERVAL * 2) {
      return true;
    }
    return eventsSinceLastSnapshot >= this.SNAPSHOT_INTERVAL;
  }

  /**
   * Should compact event log based on size
   */
  static shouldCompactEventLog(eventCount: number, potentialSavingsPercent: number): boolean {
    return eventCount > this.SNAPSHOT_INTERVAL && potentialSavingsPercent >= this.MIN_COMPACTION_SAVINGS_PERCENT;
  }

  /**
   * Get all performance boundaries as array
   */
  static getAllBoundaries(): PerformanceBoundary[] {
    return [
      {
        name: 'Max Events Per Session',
        limit: this.MAX_EVENTS_PER_SESSION,
        unit: 'events',
        warning: this.EVENTS_WARNING_THRESHOLD,
        severity: 'critical',
      },
      {
        name: 'Max Snapshots Per Session',
        limit: this.MAX_SNAPSHOTS_PER_SESSION,
        unit: 'snapshots',
        warning: this.MAX_SNAPSHOTS_PER_SESSION * 0.8,
        severity: 'warning',
      },
      {
        name: 'Snapshot Interval',
        limit: this.SNAPSHOT_INTERVAL,
        unit: 'events',
        warning: this.SNAPSHOT_INTERVAL * 0.9,
        severity: 'info',
      },
      {
        name: 'Max Reconstruction Time',
        limit: this.MAX_RECONSTRUCTION_TIME_MS,
        unit: 'ms',
        warning: this.MAX_RECONSTRUCTION_TIME_MS * 0.8,
        severity: 'warning',
      },
      {
        name: 'Max Event Log Memory',
        limit: this.TARGET_EVENTLOG_MEMORY_MB,
        unit: 'MB',
        warning: this.TARGET_EVENTLOG_MEMORY_MB * 0.8,
        severity: 'warning',
      },
      {
        name: 'Max Total Memory',
        limit: this.MAX_TOTAL_MEMORY_MB,
        unit: 'MB',
        warning: this.MAX_TOTAL_MEMORY_MB * 0.8,
        severity: 'critical',
      },
    ];
  }

  /**
   * Format metrics for human-readable output
   */
  static formatMetrics(metrics: PerformanceMetrics): string {
    return `
Performance Metrics:
  Events: ${metrics.eventLogSize}
  Snapshots: ${metrics.snapshotCount}
  Replay Duration: ${metrics.replayDuration.toFixed(2)}ms
  Validation Overhead: ${metrics.validationOverhead.toFixed(2)}ms
  Memory Usage: ${metrics.memoryUsage.toFixed(2)}MB
    `.trim();
  }
}
