/**
 * EventCompaction
 * Compresses event logs by removing redundancy and aggregating related events.
 * Reduces memory footprint while maintaining state reconstruction fidelity.
 */

import { EventLogEntry, useEventLog } from '../stores/eventLog.store';
import { PerformanceBoundaries } from './PerformanceBoundaries';
import { LauncherEventType } from '../interfaces/LauncherEvent';

export interface CompactionReport {
  originalCount: number;
  compactedCount: number;
  removedCount: number;
  savingsPercent: number;
  savingsBytes: number;
  compactionStrategies: string[];
  timestamp: number;
}

export interface AggregatedEvent {
  events: EventLogEntry[];
  aggregatedType: string;
  count: number;
  firstTimestamp: number;
  lastTimestamp: number;
}

export class EventCompaction {
  /**
   * Analyze event log for compaction opportunities
   */
  static analyzeCompactionOpportunities(
    entries: EventLogEntry[]
  ): {
    redundantPatches: number;
    aggregatableEvents: number;
    duplicateEvents: number;
    potentialSavings: number;
  } {
    let redundantPatches = 0;
    let aggregatableEvents = 0;
    let duplicateEvents = 0;
    let potentialSavings = 0;

    // Find redundant patches (same patch applied multiple times in a row)
    for (let i = 1; i < entries.length; i++) {
      const prev = entries[i - 1];
      const curr = entries[i];

      const prevJson = JSON.stringify(prev.patch);
      const currJson = JSON.stringify(curr.patch);

      if (prevJson === currJson && prev.status === 'applied' && curr.status === 'applied') {
        redundantPatches++;
        potentialSavings += prevJson.length;
      }
    }

    // Find aggregatable progress events
    let progressCount = 0;
    for (let i = 0; i < entries.length; i++) {
      if (entries[i].event.type === LauncherEventType.DownloadProgress) {
        progressCount++;
      }
    }

    // If we have more than REDUNDANCY_THRESHOLD consecutive progress events, they're aggregatable
    if (progressCount > PerformanceBoundaries.REDUNDANCY_THRESHOLD) {
      aggregatableEvents = Math.floor(progressCount / 2); // Can aggregate roughly half
      potentialSavings += aggregatableEvents * 200; // Rough size per progress event
    }

    // Find duplicate events (exact same event logged twice)
    const seenEvents = new Set<string>();
    for (const entry of entries) {
      const hash = EventCompaction.hashEvent(entry.event);
      if (seenEvents.has(hash)) {
        duplicateEvents++;
        potentialSavings += JSON.stringify(entry).length;
      }
      seenEvents.add(hash);
    }

    return {
      redundantPatches,
      aggregatableEvents,
      duplicateEvents,
      potentialSavings,
    };
  }

  /**
   * Compact event log by removing redundant entries
   */
  static compactEventLog(entries: EventLogEntry[]): {
    compacted: EventLogEntry[];
    removed: EventLogEntry[];
    report: CompactionReport;
  } {
    const removed: EventLogEntry[] = [];
    const compacted: EventLogEntry[] = [];
    const strategies: string[] = [];

    // Strategy 1: Remove duplicate events (same event type, timestamp, payload within 100ms)
    const deduped = EventCompaction.deduplicateEvents(entries);
    if (deduped.removed.length > 0) {
      strategies.push(`Deduplicated ${deduped.removed.length} duplicate events`);
      removed.push(...deduped.removed);
    }

    // Strategy 2: Remove consecutive rejected events for same domain
    const cleaned = EventCompaction.removeConsecutiveRejections(deduped.kept);
    if (cleaned.removed.length > 0) {
      strategies.push(`Removed ${cleaned.removed.length} consecutive rejections`);
      removed.push(...cleaned.removed);
    }

    // Strategy 3: Aggregate progress events
    const aggregated = EventCompaction.aggregateProgressEvents(cleaned.kept);
    if (aggregated.removed.length > 0) {
      strategies.push(
        `Aggregated ${aggregated.removed.length} progress events into ${aggregated.aggregated.length} summary events`
      );
      removed.push(...aggregated.removed);
    }

    // Combine all compacted events
    compacted.push(...cleaned.kept, ...aggregated.aggregated);
    compacted.sort((a, b) => a.appliedAt - b.appliedAt);

    const originalSize = JSON.stringify(entries).length;
    const compactedSize = JSON.stringify(compacted).length;
    const savingsBytes = originalSize - compactedSize;
    const savingsPercent = (savingsBytes / originalSize) * 100;

    const report: CompactionReport = {
      originalCount: entries.length,
      compactedCount: compacted.length,
      removedCount: removed.length,
      savingsPercent,
      savingsBytes,
      compactionStrategies: strategies,
      timestamp: Date.now(),
    };

    return { compacted, removed, report };
  }

  /**
   * Remove exact duplicate events
   */
  private static deduplicateEvents(
    entries: EventLogEntry[]
  ): {
    kept: EventLogEntry[];
    removed: EventLogEntry[];
  } {
    const kept: EventLogEntry[] = [];
    const removed: EventLogEntry[] = [];
    const seenHashes = new Map<string, EventLogEntry>();

    for (const entry of entries) {
      const hash = EventCompaction.hashEvent(entry.event);

      if (seenHashes.has(hash)) {
        const prev = seenHashes.get(hash)!;
        // If timestamps are within 100ms, consider it a duplicate
        if (entry.appliedAt - prev.appliedAt < 100) {
          removed.push(entry);
          continue;
        }
      }

      seenHashes.set(hash, entry);
      kept.push(entry);
    }

    return { kept, removed };
  }

  /**
   * Remove consecutive rejections for the same domain
   */
  private static removeConsecutiveRejections(
    entries: EventLogEntry[]
  ): {
    kept: EventLogEntry[];
    removed: EventLogEntry[];
  } {
    const kept: EventLogEntry[] = [];
    const removed: EventLogEntry[] = [];

    for (let i = 0; i < entries.length; i++) {
      const curr = entries[i];

      // Keep all applied events
      if (curr.status === 'applied') {
        kept.push(curr);
        continue;
      }

      // For rejected events, check if previous is also rejected for same reason
      if (i > 0) {
        const prev = entries[i - 1];
        if (
          prev.status === 'rejected' &&
          JSON.stringify(prev.validationErrors) === JSON.stringify(curr.validationErrors)
        ) {
          // Same rejection pattern, remove current
          removed.push(curr);
          continue;
        }
      }

      kept.push(curr);
    }

    return { kept, removed };
  }

  /**
   * Aggregate consecutive progress events into summaries
   */
  private static aggregateProgressEvents(
    entries: EventLogEntry[]
  ): {
    kept: EventLogEntry[];
    aggregated: EventLogEntry[];
    removed: EventLogEntry[];
  } {
    const kept: EventLogEntry[] = [];
    const aggregated: EventLogEntry[] = [];
    const removed: EventLogEntry[] = [];

    let progressBuffer: EventLogEntry[] = [];

    for (const entry of entries) {
      if (entry.event.type === LauncherEventType.DownloadProgress) {
        progressBuffer.push(entry);
      } else {
        // Flush progress buffer
        if (progressBuffer.length > PerformanceBoundaries.REDUNDANCY_THRESHOLD) {
          // Create aggregate
          const firstProgress = progressBuffer[0];
          const lastProgress = progressBuffer[progressBuffer.length - 1];

          aggregated.push({
            ...lastProgress,
            id: `aggregated-progress-${firstProgress.id}`,
            event: {
              ...lastProgress.event,
              type: LauncherEventType.DownloadComplete, // Represent as completion
            },
          });

          removed.push(...progressBuffer);
        } else {
          kept.push(...progressBuffer);
        }

        progressBuffer = [];
        kept.push(entry);
      }
    }

    // Handle remaining progress buffer
    if (progressBuffer.length > PerformanceBoundaries.REDUNDANCY_THRESHOLD) {
      const firstProgress = progressBuffer[0];
      const lastProgress = progressBuffer[progressBuffer.length - 1];

      aggregated.push({
        ...lastProgress,
        id: `aggregated-progress-${firstProgress.id}`,
        event: {
          ...lastProgress.event,
          type: LauncherEventType.DownloadComplete,
        },
      });

      removed.push(...progressBuffer);
    } else {
      kept.push(...progressBuffer);
    }

    return { kept, aggregated, removed };
  }

  /**
   * Hash an event for deduplication
   */
  private static hashEvent(event: any): string {
    return `${event.type}-${event.sessionId}-${event.payload?.packageId || 'global'}`;
  }

  /**
   * Compact event log for a specific session
   */
  static compactSessionEventLog(sessionId: string): CompactionReport | null {
    const eventLog = useEventLog.getState();
    const entries = eventLog.getEntriesBySession(sessionId);

    if (entries.length < PerformanceBoundaries.SNAPSHOT_INTERVAL) {
      return null; // Not enough events to justify compaction
    }

    const opportunities = this.analyzeCompactionOpportunities(entries);
    const shouldCompact = EventCompaction.calculateCompactionWorth(entries, opportunities);

    if (!shouldCompact) {
      return null;
    }

    const { compacted, removed, report } = this.compactEventLog(entries);

    // In real implementation, would update store here
    // For now, just return the report
    return report;
  }

  /**
   * Calculate if compaction is worth doing
   */
  private static calculateCompactionWorth(
    entries: EventLogEntry[],
    opportunities: ReturnType<typeof this.analyzeCompactionOpportunities>
  ): boolean {
    const potentialSavingsPercent = (opportunities.potentialSavings / (entries.length * 200)) * 100; // Rough estimate
    return potentialSavingsPercent >= PerformanceBoundaries.MIN_COMPACTION_SAVINGS_PERCENT;
  }

  /**
   * Get compaction recommendation
   */
  static getCompactionRecommendation(
    entries: EventLogEntry[]
  ): {
    shouldCompact: boolean;
    reason: string;
    estimatedSavings: number;
  } {
    if (entries.length < PerformanceBoundaries.SNAPSHOT_INTERVAL) {
      return {
        shouldCompact: false,
        reason: `Too few events (${entries.length}/${PerformanceBoundaries.SNAPSHOT_INTERVAL})`,
        estimatedSavings: 0,
      };
    }

    const opportunities = this.analyzeCompactionOpportunities(entries);

    if (opportunities.potentialSavings === 0) {
      return {
        shouldCompact: false,
        reason: 'No compaction opportunities detected',
        estimatedSavings: 0,
      };
    }

    const potentialSavingsPercent = (opportunities.potentialSavings / (entries.length * 200)) * 100;

    if (potentialSavingsPercent < PerformanceBoundaries.MIN_COMPACTION_SAVINGS_PERCENT) {
      return {
        shouldCompact: false,
        reason: `Savings too low (${potentialSavingsPercent.toFixed(1)}%)`,
        estimatedSavings: opportunities.potentialSavings,
      };
    }

    return {
      shouldCompact: true,
      reason: `Can save ~${potentialSavingsPercent.toFixed(1)}% (${opportunities.potentialSavings} bytes)`,
      estimatedSavings: opportunities.potentialSavings,
    };
  }
}
