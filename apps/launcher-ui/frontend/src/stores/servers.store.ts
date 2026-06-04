import { create } from 'zustand';
import { devtools } from 'zustand/middleware';
import { ServerInfo, ServerDetails } from '../types';
import { launcherApi } from '../wails';

export interface ServersState {
  // State
  servers: ServerInfo[];
  selectedServer: ServerInfo | null;
  serverDetails: Map<string, ServerDetails>;
  loading: boolean;
  error: string | null;
  joining: boolean;

  // Actions
  fetchServers: () => Promise<void>;
  selectServer: (server: ServerInfo | null) => void;
  joinServer: (serverId: string) => Promise<void>;
  getServerDetails: (serverId: string) => Promise<ServerDetails | null>;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
  setJoining: (joining: boolean) => void;
}

export const useServersStore = create<ServersState>()(
  devtools(
    (set: (fn: (state: ServersState) => Partial<ServersState>) => void, get: () => ServersState) => ({
      // Initial state
      servers: [],
      selectedServer: null,
      serverDetails: new Map<string, ServerDetails>(),
      loading: false,
      error: null,
      joining: false,

      // Actions
      fetchServers: async () => {
        set((state) => ({ ...state, loading: true, error: null }));
        try {
          const servers = await launcherApi.getServerList();
          set((state) => ({ ...state, servers, loading: false }));
        } catch (err) {
          set((state) => ({ ...state, error: 'Failed to fetch servers', loading: false }));
          console.error('Failed to fetch servers:', err);
        }
      },

      selectServer: (server: ServerInfo | null) => {
        set((state) => ({ ...state, selectedServer: server }));
      },

      joinServer: async (serverId: string) => {
        set((state) => ({ ...state, joining: true, error: null }));
        try {
          await launcherApi.joinServer(serverId);
        } catch (err) {
          set((state) => ({ ...state, joining: false, error: 'Failed to join server' }));
          console.error('Failed to join server:', err);
        }
      },

      getServerDetails: async (serverId: string) => {
        const cached = get().serverDetails.get(serverId);
        if (cached) return cached;

        try {
          const details = await launcherApi.getServerDetails(serverId);
          if (details) {
            set((state) => ({
              ...state,
              serverDetails: new Map(state.serverDetails).set(serverId, details),
            }));
          }
          return details;
        } catch (err) {
          console.error('Failed to get server details:', err);
          return null;
        }
      },

      setLoading: (loading: boolean) => set((state) => ({ ...state, loading })),
      setError: (error: string | null) => set((state) => ({ ...state, error })),
      setJoining: (joining: boolean) => set((state) => ({ ...state, joining })),
    }),
    { name: 'servers-store' }
  )
);
