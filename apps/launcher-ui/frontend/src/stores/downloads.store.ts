import { create } from 'zustand';
import { devtools } from 'zustand/middleware';
import { SessionStatus } from '../types';

export interface DownloadsState {
  // State
  sessions: Map<string, SessionStatus>;

  // Actions
  updateSession: (session: SessionStatus) => void;
  updateSessionProgress: (sessionId: string, progress: number) => void;
  updateSessionMod: (sessionId: string, modName: string) => void;
  completeSession: (sessionId: string) => void;
  failSession: (sessionId: string, error: string) => void;
  removeSession: (sessionId: string) => void;
  clearCompleted: () => void;

  // Getters
  getActiveDownloads: () => SessionStatus[];
  getCompletedDownloads: () => SessionStatus[];
  getSession: (sessionId: string) => SessionStatus | undefined;
}

export const useDownloadsStore = create<DownloadsState>()(
  devtools(
    (set: (fn: (state: DownloadsState) => Partial<DownloadsState>) => void, get: () => DownloadsState) => ({
      // Initial state
      sessions: new Map<string, SessionStatus>(),

      // Actions
      updateSession: (session: SessionStatus) => {
        set((state) => ({
          ...state,
          sessions: new Map(state.sessions).set(session.sessionId, session),
        }));
      },

      updateSessionProgress: (sessionId: string, progress: number) => {
        set((state) => {
          const session = state.sessions.get(sessionId);
          if (!session) return state;
          
          return {
            ...state,
            sessions: new Map(state.sessions).set(sessionId, {
              ...session,
              progress,
              state: progress >= 100 ? 'complete' : 'downloading',
            }),
          };
        });
      },

      updateSessionMod: (sessionId: string, modName: string) => {
        set((state) => {
          const session = state.sessions.get(sessionId);
          if (!session) return state;
          
          return {
            ...state,
            sessions: new Map(state.sessions).set(sessionId, {
              ...session,
              currentMod: modName,
              state: 'downloading',
            }),
          };
        });
      },

      completeSession: (sessionId: string) => {
        set((state) => {
          const session = state.sessions.get(sessionId);
          if (!session) return state;
          
          return {
            ...state,
            sessions: new Map(state.sessions).set(sessionId, {
              ...session,
              state: 'complete',
              progress: 100,
            }),
          };
        });
      },

      failSession: (sessionId: string, error: string) => {
        set((state) => {
          const session = state.sessions.get(sessionId);
          if (!session) return state;
          
          return {
            ...state,
            sessions: new Map(state.sessions).set(sessionId, {
              ...session,
              state: 'error',
              errors: [...(session.errors || []), error],
            }),
          };
        });
      },

      removeSession: (sessionId: string) => {
        set((state) => {
          const newSessions = new Map(state.sessions);
          newSessions.delete(sessionId);
          return { ...state, sessions: newSessions };
        });
      },

      clearCompleted: () => {
        set((state) => {
          const newSessions = new Map<string, SessionStatus>();
          for (const [id, session] of state.sessions) {
            if (session.state !== 'complete') {
              newSessions.set(id, session);
            }
          }
          return { ...state, sessions: newSessions };
        });
      },

      // Getters
      getActiveDownloads: () => {
        return Array.from(get().sessions.values()).filter(
          (s: SessionStatus) => s.state === 'downloading' || s.state === 'resolving' || s.state === 'installing'
        );
      },

      getCompletedDownloads: () => {
        return Array.from(get().sessions.values()).filter((s: SessionStatus) => s.state === 'complete');
      },

      getSession: (sessionId: string) => {
        return get().sessions.get(sessionId);
      },
    }),
    { name: 'downloads-store' }
  )
);
