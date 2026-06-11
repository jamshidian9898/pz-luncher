// Re-export shared contracts; extend with UI-only fields where needed.
export type {
  ServerManifest,
  ModEntry,
  ServerDescriptor,
  ServerRegistry,
  LauncherSettings,
  LauncherEvent,
  LauncherEventType,
} from './contracts/generated';

export { settingsFromLauncher, settingsToLauncher } from './contracts/generated';

export interface ServerInfo {
  id: string;
  name: string;
  description: string;
  playerCount: number;
  maxPlayers: number;
  ping: number;
  modCount: number;
  installed: boolean;
  upToDate: boolean;
}

export interface ServerDetails extends ServerInfo {
  mods: ModInfo[];
  totalSize: number;
  installedSize: number;
  missingSize: number;
}

export interface ModInfo {
  id: string;
  name: string;
  workshopId: string;
  size: number;
  installed: boolean;
  upToDate: boolean;
  required: boolean;
}

export interface SessionStatus {
  sessionId: string;
  state: 'idle' | 'resolving' | 'downloading' | 'installing' | 'complete' | 'error';
  progress: number;
  currentMod?: string;
  downloadSpeed?: number;
  eta?: number;
  errors?: string[];
  serverName?: string; // Associated server name for display
  serverId?: string;   // Associated server ID
}

/** UI settings — maps to LauncherSettings via settingsToLauncher */
export interface Settings {
  gamePath: string;
  backendUrl: string;
  cacheLocation: string;
  profilesLocation: string;
  maxConcurrent: number;
  bandwidthLimit: number;
  verifyChecksum: boolean;
  launchOptions?: string; // Additional launch arguments (e.g., "-debug -nosound")
}

export interface Progress {
  current: number;
  total: number;
  percent: number;
  speed?: number;
  eta?: number;
}

declare global {
  interface Window {
    go: {
      main: {
        App: {
          JoinServer(serverId: string): Promise<void>;
          LaunchServer(serverId: string): Promise<void>;
          GetServerList(): Promise<ServerInfo[]>;
          GetServerDetails(serverId: string): Promise<ServerDetails>;
          GetSessionStatus(sessionId: string): Promise<SessionStatus>;
          RepairCache(): Promise<void>;
          CheckBackend(): Promise<{
            backendUrl: string; backend: string; backendMsg: string;
            agents: string; agentsMsg: string;
            servers: string; serversMsg: string;
            workspaceRoot: string; settingsPath: string;
          }>;
          GetSettings(): Promise<Settings>;
          SaveSettings(settings: Settings): Promise<void>;
        };
      };
    };
    runtime: {
      EventsOn(event: string, callback: (data: unknown) => void): () => void;
      EventsEmit(event: string, data: unknown): void;
    };
  }
}

export {};
