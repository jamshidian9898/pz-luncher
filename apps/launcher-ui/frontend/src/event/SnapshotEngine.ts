/**
 * SnapshotEngine
 * Creates and restores state snapshots for efficient replay.
 * Enables reconstruction of state without replaying all historical events.
 */

import { useDownloadsStore } from '../stores/downloads.store';
import { useTraceStore } from '../stores/trace.store';
import { useSessionStore } from '../stores/session.store';
import { useServersStore } from '../stores/servers.store';
import { PerformanceBoundaries } from './PerformanceBoundaries';

export interface StateSnapshot {
  id: string;
  sessionId: string;
  eventNumber: number; // The event index this snapshot was created after
  timestamp: number;
  downloads: any;
  trace: any;
  session: any;
  servers: any;
  memorySize: number; // Approximate size in bytes
}

export interface SnapshotMetadata {
  id: string;
  sessionId: string;
  eventNumber: number;
  timestamp: number;
  memorySize: number;
  createdAt: number;
}

export class SnapshotEngine {
  /**
   * Create a snapshot of current state
   */
  static createSnapshot(sessionId: string, eventNumber: number): StateSnapshot {
    const timestamp = Date.now();
    const id = `snapshot-${sessionId}-${eventNumber}-${timestamp}`;

    const downloads = JSON.parse(JSON.stringify(useDownloadsStore.getState()));
    const trace = JSON.parse(JSON.stringify(useTraceStore.getState()));
    const session = JSON.parse(JSON.stringify(useSessionStore.getState()));
    const servers = JSON.parse(JSON.stringify(useServersStore.getState()));

    // Estimate memory size
    const memorySize = this.estimateMemorySize(downloads, trace, session, servers);

    return {
      id,
      sessionId,
      eventNumber,
      timestamp,
      downloads,
      trace,
      session,
      servers,
      memorySize,
    };
  }

  /**
   * Restore state from snapshot
   */
  static restoreSnapshot(snapshot: StateSnapshot): void {
    useDownloadsStore.setState(snapshot.downloads);
    useTraceStore.setState(snapshot.trace);
    useSessionStore.setState(snapshot.session);
    useServersStore.setState(snapshot.servers);
  }

  /**
   * Get metadata for a snapshot without loading full state
   */
  static getSnapshotMetadata(snapshot: StateSnapshot): SnapshotMetadata {
    return {
      id: snapshot.id,
      sessionId: snapshot.sessionId,
      eventNumber: snapshot.eventNumber,
      timestamp: snapshot.timestamp,
      memorySize: snapshot.memorySize,
      createdAt: Date.now(),
    };
  }

  /**
   * Estimate memory size of state object (rough approximation)
   */
  private static estimateMemorySize(...objects: any[]): number {
    const json = JSON.stringify(objects);
    // Each character is roughly 2 bytes in UTF-16, but compress to estimate real memory
    return Math.ceil(json.length * 1.5);
  }

  /**
   * Find best snapshot for replay starting point
   * Returns snapshot that minimizes replay time while staying within time budget
   */
  static findBestSnapshotForReconstruction(
    snapshots: StateSnapshot[],
    targetEventNumber: number
  ): StateSnapshot | null {
    if (snapshots.length === 0) return null;

    // Find latest snapshot before target event
    let bestSnapshot = null;
    let smallestDelta = Infinity;

    for (const snapshot of snapshots) {
      if (snapshot.eventNumber <= targetEventNumber) {
        const delta = targetEventNumber - snapshot.eventNumber;
        if (delta < smallestDelta) {
          smallestDelta = delta;
          bestSnapshot = snapshot;
        }
      }
    }

    // Check if replay from this snapshot would exceed time budget
    if (bestSnapshot) {
      const estimatedReplayTime = smallestDelta * 10; // Rough: 10ms per event
      if (estimatedReplayTime > PerformanceBoundaries.MAX_RECONSTRUCTION_TIME_MS) {
        return bestSnapshot; // Use it anyway as it reduces time significantly
      }
    }

    return bestSnapshot;
  }

  /**
   * Compact snapshots by removing old ones and keeping only strategic ones
   */
  static compactSnapshots(
    snapshots: StateSnapshot[]
  ): {
    kept: StateSnapshot[];
    removed: StateSnapshot[];
    savedMemory: number;
  } {
    if (snapshots.length <= PerformanceBoundaries.MAX_SNAPSHOTS_PER_SESSION) {
      return {
        kept: snapshots,
        removed: [],
        savedMemory: 0,
      };
    }

    // Keep only the most recent snapshots
    const sortedByTime = [...snapshots].sort((a, b) => b.timestamp - a.timestamp);
    const kept = sortedByTime.slice(0, PerformanceBoundaries.MAX_SNAPSHOTS_PER_SESSION);
    const removed = sortedByTime.slice(PerformanceBoundaries.MAX_SNAPSHOTS_PER_SESSION);

    const savedMemory = removed.reduce((sum, s) => sum + s.memorySize, 0);

    return {
      kept,
      removed,
      savedMemory,
    };
  }

  /**
   * Calculate how much memory snapshots use
   */
  static calculateSnapshotMemory(snapshots: StateSnapshot[]): number {
    return snapshots.reduce((sum, s) => sum + s.memorySize, 0);
  }

  /**
   * Analyze snapshot efficiency
   */
  static analyzeSnapshotEfficiency(
    snapshots: StateSnapshot[],
    totalEvents: number
  ): {
    snapshotCount: number;
    totalMemory: number;
    averageMemoryPerSnapshot: number;
    averageEventsPerSnapshot: number;
    compressionRatio: number;
    recommendation: string;
  } {
    if (snapshots.length === 0) {
      return {
        snapshotCount: 0,
        totalMemory: 0,
        averageMemoryPerSnapshot: 0,
        averageEventsPerSnapshot: 0,
        compressionRatio: 0,
        recommendation: 'No snapshots yet. Consider creating one.',
      };
    }

    const totalMemory = this.calculateSnapshotMemory(snapshots);
    const avgMemory = totalMemory / snapshots.length;
    const avgEvents = totalEvents / snapshots.length;
    const compressionRatio = avgEvents * 0.01; // Rough: 1% overhead per event to store

    let recommendation = '';
    if (snapshots.length >= PerformanceBoundaries.MAX_SNAPSHOTS_PER_SESSION) {
      recommendation = `Too many snapshots (${snapshots.length}). Compact to ${PerformanceBoundaries.MAX_SNAPSHOTS_PER_SESSION}.`;
    } else if (totalMemory > PerformanceBoundaries.TARGET_SNAPSHOT_MEMORY_MB * 1024 * 1024) {
      recommendation = `Snapshot memory high (${(totalMemory / 1024 / 1024).toFixed(2)}MB). Consider interval increase.`;
    } else if (avgEvents > PerformanceBoundaries.SNAPSHOT_INTERVAL * 2) {
      recommendation = 'Snapshots are sparse. Consider more frequent snapshots.';
    } else {
      recommendation = 'Snapshot efficiency is good.';
    }

    return {
      snapshotCount: snapshots.length,
      totalMemory,
      averageMemoryPerSnapshot: avgMemory,
      averageEventsPerSnapshot: avgEvents,
      compressionRatio,
      recommendation,
    };
  }

  /**
   * Create minimal snapshot by only storing deltas
   * (Advanced: not fully implemented, placeholder for future optimization)
   */
  static createDeltaSnapshot(previousSnapshot: StateSnapshot, currentSnapshot: StateSnapshot): any {
    // TODO: Implement delta compression
    // Compare previous and current, store only changed fields
    return {
      id: currentSnapshot.id,
      baseSnapshotId: previousSnapshot.id,
      deltas: {
        // Only changed state branches
      },
    };
  }
}
