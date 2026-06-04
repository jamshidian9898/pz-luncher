import { create } from 'zustand';
import { devtools } from 'zustand/middleware';
import { ServerInfo } from '../types';

export type LaunchState =
  | 'idle'
  | 'resolving'
  | 'downloading'
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

  setCurrentSession: (sessionId: string | null) => void;
  setLaunchState: (state: LaunchState) => void;
  setCurrentServer: (server: ServerInfo | null) => void;
  resetSession: () => void;
}

export const useSessionStore = create<SessionState>()(
  devtools(
    (set: (fn: (state: SessionState) => Partial<SessionState>) => void) => ({
      currentSessionId: null,
      launchState: 'idle',
      currentServer: null,

      setCurrentSession: (sessionId: string | null) =>
        set((state) => ({ ...state, currentSessionId: sessionId })),

      setLaunchState: (launchState: LaunchState) =>
        set((state) => ({ ...state, launchState })),

      setCurrentServer: (currentServer: ServerInfo | null) =>
        set((state) => ({ ...state, currentServer })),

      resetSession: () =>
        set(() => ({ currentSessionId: null, launchState: 'idle', currentServer: null })),
    }),
    { name: 'session-store' }
  )
);
