import { create } from 'zustand';
import { devtools } from 'zustand/middleware';
import { StateSnapshot } from '../event/SnapshotEngine';

interface SnapshotStoreState {
  snapshots: Map<string, StateSnapshot[]>;

  addSnapshot: (snapshot: StateSnapshot) => void;
  getSnapshotsForSession: (sessionId: string) => StateSnapshot[];
  removeOldSnapshots: (sessionId: string, keepCount: number) => void;
  getLatestSnapshot: (sessionId: string) => StateSnapshot | null;
  getSnapshotAt: (sessionId: string, eventNumber: number) => StateSnapshot | null;
  clearSessionSnapshots: (sessionId: string) => void;
  getTotalSnapshotMemory: () => number;
  getSnapshotStats: () => {
    totalSnapshots: number;
    totalMemory: number;
    sessionCount: number;
  };
}

export const useSnapshotStore = create<SnapshotStoreState>()(
  devtools(
    (set, get) => ({
      snapshots: new Map<string, StateSnapshot[]>(),

      addSnapshot: (snapshot: StateSnapshot) => {
        set((state) => {
          const sessionSnapshots = state.snapshots.get(snapshot.sessionId) || [];
          const updated = new Map(state.snapshots);
          updated.set(snapshot.sessionId, [...sessionSnapshots, snapshot].sort((a, b) => a.eventNumber - b.eventNumber));
          return { snapshots: updated };
        });
      },

      getSnapshotsForSession: (sessionId: string) => {
        return get().snapshots.get(sessionId) || [];
      },

      removeOldSnapshots: (sessionId: string, keepCount: number) => {
        set((state) => {
          const sessionSnapshots = state.snapshots.get(sessionId) || [];
          if (sessionSnapshots.length <= keepCount) return state;

          const updated = new Map(state.snapshots);
          const kept = sessionSnapshots.slice(-keepCount);
          updated.set(sessionId, kept);
          return { snapshots: updated };
        });
      },

      getLatestSnapshot: (sessionId: string) => {
        const snapshots = get().snapshots.get(sessionId) || [];
        return snapshots.length > 0 ? snapshots[snapshots.length - 1] : null;
      },

      getSnapshotAt: (sessionId: string, eventNumber: number) => {
        const snapshots = get().snapshots.get(sessionId) || [];
        for (let i = snapshots.length - 1; i >= 0; i--) {
          if (snapshots[i].eventNumber <= eventNumber) {
            return snapshots[i];
          }
        }
        return null;
      },

      clearSessionSnapshots: (sessionId: string) => {
        set((state) => {
          const updated = new Map(state.snapshots);
          updated.delete(sessionId);
          return { snapshots: updated };
        });
      },

      getTotalSnapshotMemory: () => {
        let total = 0;
        get().snapshots.forEach((snapshots) => {
          snapshots.forEach((s) => {
            total += s.memorySize;
          });
        });
        return total;
      },

      getSnapshotStats: () => {
        const state = get();
        let totalSnapshots = 0;
        let totalMemory = 0;

        state.snapshots.forEach((snapshots) => {
          totalSnapshots += snapshots.length;
          snapshots.forEach((s) => {
            totalMemory += s.memorySize;
          });
        });

        return {
          totalSnapshots,
          totalMemory,
          sessionCount: state.snapshots.size,
        };
      },
    }),
    { name: 'snapshot-store' }
  )
);
