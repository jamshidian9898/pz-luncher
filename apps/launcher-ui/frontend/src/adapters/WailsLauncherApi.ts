import { LauncherApi } from '../interfaces/LauncherApi';
import { ServerInfo, ServerDetails, SessionStatus } from '../types';

export class WailsLauncherApi implements LauncherApi {
  async getServerList(): Promise<ServerInfo[]> {
    return window.go.main.App.GetServerList();
  }

  async getServerDetails(serverId: string): Promise<ServerDetails> {
    return window.go.main.App.GetServerDetails(serverId);
  }

  async joinServer(serverId: string): Promise<void> {
    return window.go.main.App.JoinServer(serverId);
  }

  async launchServer(serverId: string): Promise<void> {
    return window.go.main.App.LaunchServer(serverId);
  }

  async getSessionStatus(sessionId: string): Promise<SessionStatus> {
    return window.go.main.App.GetSessionStatus(sessionId);
  }
}
