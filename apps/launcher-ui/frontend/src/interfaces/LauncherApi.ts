import { ServerInfo, ServerDetails, SessionStatus } from '../types';

export interface LauncherApi {
  getServerList(): Promise<ServerInfo[]>;
  getServerDetails(serverId: string): Promise<ServerDetails>;
  joinServer(serverId: string): Promise<void>;
  launchServer(serverId: string): Promise<void>;
  stopGame(): Promise<void>;
  isGameRunning(): Promise<boolean>;
  getSessionStatus(sessionId: string): Promise<SessionStatus>;
}
