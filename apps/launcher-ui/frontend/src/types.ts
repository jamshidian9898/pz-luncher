// Types matching Wails bindings (main.go)

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
}

export interface Settings {
  steamcmdPath: string;
  cacheLocation: string;
  profilesLocation: string;
  maxConcurrent: number;
  bandwidthLimit: number;
}

export interface Progress {
  current: number;
  total: number;
  percent: number;
  speed?: number;
  eta?: number;
}

// Wails runtime types
declare global {
  interface Window {
    go: {
      main: {
        App: {
          JoinServer(serverId: string): Promise<void>;
          GetServerList(): Promise<ServerInfo[]>;
          GetServerDetails(serverId: string): Promise<ServerDetails>;
          GetSessionStatus(sessionId: string): Promise<SessionStatus>;
          RepairCache(): Promise<void>;
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
