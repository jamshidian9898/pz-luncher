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

export interface ContributeEntry {
  gameVersion: string;
  platform: string;
  localPath: string;
  sizeBytes: number;
  sha256?: string;
  trustLevel: string;
  uploadCount: number;
  source: 'cache' | 'gamePath';
}

export interface ContributeStatus {
  entries: ContributeEntry[];
  backendUrl: string;
}

export interface SubmitHashResult {
  status: string;
  trustLevel: string;
  uploadCount: number;
  uploadUrl?: string;
  error?: string;
}

export interface CacheEntry {
  type: 'version' | 'mod';
  key: string;
  platform?: string;
  gameVersion?: string;
  modId?: string;
  sizeBytes: number;
  downloadedAt: string;
  lastUsedAt: string;
  usedByProfiles: string[];
}

export interface CacheStats {
  totalBytes: number;
  versionBytes: number;
  modBytes: number;
  entries: CacheEntry[];
  deletableBytes: number;
}

export interface VersionSource {
  type: 'registry' | 'agent' | 'hoster';
  url: string;
  trustLevel: string;
  description: string;
}

export interface VersionCandidate {
  gameVersion: string;
  platform: string;
  sizeBytes: number;
  trustLevel: string;
  availableSources: VersionSource[];
  isLocal: boolean;
  localPath?: string;
}

export interface VersionSelector {
  required: string;
  localVersion?: string;
  candidates: VersionCandidate[];
  needDownload: boolean;
  autoSelected?: VersionSource;
}

declare global {
  interface Window {
    go: {
      main: {
        App: {
          JoinServer(serverId: string): Promise<void>;
          LaunchServer(serverId: string): Promise<void>;
          StopGame(): Promise<void>;
          IsGameRunning(): Promise<boolean>;
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
          // RFC-0060: Community Upload
          GetContributeStatus(): Promise<ContributeStatus>;
          HashGameVersion(localPath: string): Promise<string>;
          SubmitVersionHash(gameId: string, version: string, platform: string, sha256: string, sizeBytes: number): Promise<SubmitHashResult>;
          UploadVersionBinary(gameId: string, version: string, platform: string, localPath: string, sha256: string): Promise<void>;
          // RFC-0061: Cache Manager
          GetCacheStats(): Promise<CacheStats>;
          DeleteCacheEntry(type: 'version' | 'mod', key: string): Promise<void>;
          // RFC-0062: Game Version Selection
          GetVersionSelector(requiredVersion: string): Promise<VersionSelector>;
          ConfirmVersionDownload(gameVersion: string, platform: string, sourceUrl: string): Promise<void>;
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
