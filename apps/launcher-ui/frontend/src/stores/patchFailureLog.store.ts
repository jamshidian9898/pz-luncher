import { create } from 'zustand';
import { devtools } from 'zustand/middleware';

export interface PatchFailure {
  id: string;
  eventId: string;
  eventType: string;
  domain: 'downloads' | 'trace' | 'session' | 'servers';
  reason: string[];
  payload: unknown;
  timestamp: number;
  sessionId: string;
}

interface PatchFailureLogState {
  failures: PatchFailure[];
  addFailure: (failure: Omit<PatchFailure, 'id'>) => void;
  clear: () => void;
  getFailuresBySession: (sessionId: string) => PatchFailure[];
  getFailuresByDomain: (domain: PatchFailure['domain']) => PatchFailure[];
  getRecentFailures: (count: number) => PatchFailure[];
}

export const usePatchFailureLog = create<PatchFailureLogState>()(
  devtools(
    (set, get) => ({
      failures: [],

      addFailure: (failure) => {
        set((state) => {
          const id = `failure-${Date.now()}-${Math.random()}`;
          const newFailure: PatchFailure = {
            ...failure,
            id,
          };

          // Keep only last 500 failures to avoid memory bloat
          const failures = [newFailure, ...state.failures].slice(0, 500);
          return { failures };
        });
      },

      clear: () => {
        set({ failures: [] });
      },

      getFailuresBySession: (sessionId) => {
        return get().failures.filter((f) => f.sessionId === sessionId);
      },

      getFailuresByDomain: (domain) => {
        return get().failures.filter((f) => f.domain === domain);
      },

      getRecentFailures: (count) => {
        return get().failures.slice(0, count);
      },
    }),
    { name: 'patch-failure-log' }
  )
);
