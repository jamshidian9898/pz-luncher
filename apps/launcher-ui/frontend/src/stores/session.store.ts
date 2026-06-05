import { create } from 'zustand';
import { devtools } from 'zustand/middleware';
import { ServerInfo } from '../types';

export type LaunchState =
  | 'idle'
  | 'resolving'
  | 'downloading'
  | 'installing'
  | 'verifying'
  | 'materializing'
  | 'launching'
  | 'running'
  | 'complete'
  | 'error';

export interface SessionState {
  currentSessionId: string | null;
  launchState: LaunchState;
  currentServer: ServerInfo | null;
  lastError: string | null;
  joinStartedAt: number | null;

  setCurrentSession: (sessionId: string | null) => void;
  setLaunchState: (state: LaunchState) => void;
  setCurrentServer: (server: ServerInfo | null) => void;
  setLastError: (error: string | null) => void;
  resetSession: () => void;
}

export const useSessionStore = create<SessionState>()(
  devtools(
    (set: (fn: (state: SessionState) => Partial<SessionState>) => void) => ({
      currentSessionId: null,
      launchState: 'idle',
      currentServer: null,
      lastError: null,
      joinStartedAt: null,

      setCurrentSession: (sessionId: string | null) =>
        set((state) => ({ ...state, currentSessionId: sessionId })),

      setLaunchState: (launchState: LaunchState) =>
        set((state) => ({ ...state, launchState })),

      setCurrentServer: (currentServer: ServerInfo | null) =>
        set((state) => ({ ...state, currentServer, joinStartedAt: Date.now() })),

      setLastError: (lastError: string | null) =>
        set((state) => ({ ...state, lastError })),

      resetSession: () =>
        set(() => ({ currentSessionId: null, launchState: 'idle', currentServer: null, lastError: null, joinStartedAt: null })),
    }),
    { name: 'session-store' }
  )
);
