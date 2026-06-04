import { create } from 'zustand';
import { devtools } from 'zustand/middleware';

export interface TraceNode {
  id: string;
  timestamp: number;
  type: 'resolve' | 'download' | 'verify' | 'install' | 'complete' | 'error';
  modId: string;
  modName: string;
  provider?: string;
  providerReason?: string;
  progress?: {
    current: number;
    total: number;
    percent: number;
    speed?: number;
  };
  state?: string;
  error?: string;
  duration?: number;
}

export interface TraceState {
  // State
  traces: Map<string, TraceNode[]>;
  activeTrace: string | null;

  // Actions
  addTraceEvent: (sessionId: string, event: TraceNode) => void;
  updateTraceProgress: (sessionId: string, nodeId: string, progress: number, speed?: number) => void;
  completeTraceNode: (sessionId: string, nodeId: string) => void;
  setActiveTrace: (sessionId: string | null) => void;
  getTraceForSession: (sessionId: string) => TraceNode[];
  clearTrace: (sessionId: string) => void;
  exportTrace: (sessionId: string) => string;
}

export const useTraceStore = create<TraceState>()(
  devtools(
    (set: (fn: (state: TraceState) => Partial<TraceState>) => void, get: () => TraceState) => ({
      // Initial state
      traces: new Map<string, TraceNode[]>(),
      activeTrace: null,

      // Actions
      addTraceEvent: (sessionId: string, event: TraceNode) => {
        set((state) => {
          const existing = state.traces.get(sessionId) || [];
          return {
            ...state,
            traces: new Map(state.traces).set(sessionId, [...existing, event]),
          };
        });
      },

      updateTraceProgress: (sessionId: string, nodeId: string, progress: number, speed?: number) => {
        set((state) => {
          const existing = state.traces.get(sessionId) || [];
          const updated = existing.map((node: TraceNode) =>
            node.id === nodeId
              ? {
                  ...node,
                  progress: {
                    current: progress,
                    total: 100,
                    percent: progress,
                    speed,
                  },
                }
              : node
          );
          return {
            ...state,
            traces: new Map(state.traces).set(sessionId, updated),
          };
        });
      },

      completeTraceNode: (sessionId: string, nodeId: string) => {
        set((state) => {
          const existing = state.traces.get(sessionId) || [];
          const updated = existing.map((node: TraceNode) =>
            node.id === nodeId
              ? {
                  ...node,
                  type: 'complete' as const,
                  progress: { current: 100, total: 100, percent: 100 },
                }
              : node
          );
          return {
            ...state,
            traces: new Map(state.traces).set(sessionId, updated),
          };
        });
      },

      setActiveTrace: (sessionId: string | null) => {
        set((state) => ({ ...state, activeTrace: sessionId }));
      },

      getTraceForSession: (sessionId: string) => {
        return get().traces.get(sessionId) || [];
      },

      clearTrace: (sessionId: string) => {
        set((state) => {
          const newTraces = new Map(state.traces);
          newTraces.delete(sessionId);
          return { ...state, traces: newTraces };
        });
      },

      exportTrace: (sessionId: string) => {
        const trace = get().traces.get(sessionId) || [];
        return JSON.stringify(
          {
            sessionId,
            exportedAt: new Date().toISOString(),
            nodes: trace,
            summary: {
              total: trace.length,
              complete: trace.filter((n: TraceNode) => n.type === 'complete').length,
              errors: trace.filter((n: TraceNode) => n.type === 'error').length,
            },
          },
          null,
          2
        );
      },
    }),
    { name: 'trace-store' }
  )
);
