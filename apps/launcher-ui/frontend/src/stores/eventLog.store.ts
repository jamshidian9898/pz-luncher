import { create } from 'zustand';
import { devtools } from 'zustand/middleware';
import { LauncherEvent } from '../interfaces/LauncherEvent';
import { LauncherEventPatch } from '../event/LauncherStateReducer';

export interface EventLogEntry {
  id: string;
  event: LauncherEvent;
  patch: LauncherEventPatch;
  validationErrors: string[];
  appliedAt: number;
  status: 'applied' | 'rejected' | 'skipped';
  sessionId: string;
}

interface EventLogState {
  entries: EventLogEntry[];
  addEntry: (entry: Omit<EventLogEntry, 'id'>) => void;
  getEntriesBySession: (sessionId: string) => EventLogEntry[];
  getEntriesAfter: (timestamp: number) => EventLogEntry[];
  getEntriesBetween: (startTime: number, endTime: number) => EventLogEntry[];
  clear: () => void;
  clearSession: (sessionId: string) => void;
  getStats: () => {
    totalEvents: number;
    applied: number;
    rejected: number;
    byDomain: Record<string, number>;
  };
}

export const useEventLog = create<EventLogState>()(
  devtools(
    (set, get) => ({
      entries: [],

      addEntry: (entry) => {
        set((state) => {
          const id = `log-${Date.now()}-${Math.random()}`;
          const newEntry: EventLogEntry = {
            ...entry,
            id,
          };

          // Keep last 1000 entries for audit trail
          const entries = [newEntry, ...state.entries].slice(0, 1000);
          return { entries };
        });
      },

      getEntriesBySession: (sessionId) => {
        return get().entries.filter((e) => e.sessionId === sessionId);
      },

      getEntriesAfter: (timestamp) => {
        return get().entries.filter((e) => e.appliedAt >= timestamp);
      },

      getEntriesBetween: (startTime, endTime) => {
        return get().entries.filter((e) => e.appliedAt >= startTime && e.appliedAt <= endTime);
      },

      clear: () => {
        set({ entries: [] });
      },

      clearSession: (sessionId) => {
        set((state) => ({
          entries: state.entries.filter((e) => e.sessionId !== sessionId),
        }));
      },

      getStats: () => {
        const entries = get().entries;
        const stats = {
          totalEvents: entries.length,
          applied: entries.filter((e) => e.status === 'applied').length,
          rejected: entries.filter((e) => e.status === 'rejected').length,
          byDomain: {} as Record<string, number>,
        };

        entries.forEach((entry) => {
          const eventType = entry.event.type;
          stats.byDomain[eventType] = (stats.byDomain[eventType] || 0) + 1;
        });

        return stats;
      },
    }),
    { name: 'event-log' }
  )
);
