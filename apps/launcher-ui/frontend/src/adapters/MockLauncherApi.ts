import { LauncherApi } from '../interfaces/LauncherApi';
import { ServerInfo, ServerDetails, SessionStatus } from '../types';
import { LauncherEvent, LauncherEventType } from '../interfaces/LauncherEvent';
import { mockEventBus } from './MockEventsApi';

const MOCK_SERVERS: ServerInfo[] = [
  {
    id: 'server-1',
    name: 'One Life',
    description: 'Hardcore survival server',
    playerCount: 42,
    maxPlayers: 64,
    ping: 45,
    modCount: 15,
    installed: false,
    upToDate: false,
  },
  {
    id: 'server-2',
    name: 'Casual RP',
    description: 'Relaxed roleplay server',
    playerCount: 28,
    maxPlayers: 128,
    ping: 30,
    modCount: 8,
    installed: true,
    upToDate: true,
  },
  {
    id: 'server-3',
    name: 'PvP Arena',
    description: 'Player vs player combat',
    playerCount: 15,
    maxPlayers: 32,
    ping: 60,
    modCount: 5,
    installed: true,
    upToDate: false,
  },
];

export class MockLauncherApi implements LauncherApi {
  async getServerList(): Promise<ServerInfo[]> {
    return new Promise((resolve) => setTimeout(() => resolve(MOCK_SERVERS), 300));
  }

  async getServerDetails(serverId: string): Promise<ServerDetails> {
    const server = MOCK_SERVERS.find((s) => s.id === serverId) ?? MOCK_SERVERS[0];
    return new Promise((resolve) =>
      setTimeout(
        () =>
          resolve({
            ...server,
            mods: [
              { id: 'm1', name: 'Brita Weapons', workshopId: '2200148440', size: 100 * 1024 * 1024, installed: server.installed, upToDate: server.upToDate, required: true },
              { id: 'm2', name: 'Common Sense', workshopId: '2875848298', size: 50 * 1024 * 1024, installed: server.installed, upToDate: server.upToDate, required: true },
              { id: 'm3', name: 'True Music', workshopId: '2529746525', size: 20 * 1024 * 1024, installed: server.installed, upToDate: server.upToDate, required: false },
            ],
            totalSize: 170 * 1024 * 1024,
            installedSize: server.installed ? 170 * 1024 * 1024 : 0,
            missingSize: server.installed ? 0 : 170 * 1024 * 1024,
          }),
        200
      )
    );
  }

  async joinServer(serverId: string): Promise<void> {
    this.simulateMockSession(serverId);
  }

  async launchServer(_serverId: string): Promise<void> {
    return Promise.resolve();
  }

  async getSessionStatus(sessionId: string): Promise<SessionStatus> {
    return { sessionId, state: 'complete', progress: 100 };
  }

  private simulateMockSession(serverId: string) {
    const sessionId = `session-${Date.now()}`;
    const emit = (event: LauncherEvent) => mockEventBus.emit(event);
    const now = () => Math.floor(Date.now() / 1000);
    const mods = ['Brita Weapons', 'Common Sense', 'True Music'];

    (async () => {
      emit({
        type: LauncherEventType.SessionStart,
        timestamp: now(),
        sessionId,
        payload: { metadata: { serverId } },
      });
      await this.delay(200);

      emit({
        type: LauncherEventType.ModResolveStart,
        timestamp: now(),
        sessionId,
        payload: { packageId: 'resolve' },
      });
      await this.delay(400);
      emit({
        type: LauncherEventType.ModResolveComplete,
        timestamp: now(),
        sessionId,
        payload: { packageId: 'resolve', metadata: { modCount: mods.length } },
      });
      await this.delay(100);

      for (const mod of mods) {
        emit({
          type: LauncherEventType.DownloadStart,
          timestamp: now(),
          sessionId,
          payload: { packageId: mod },
        });

        for (let pct = 0; pct <= 100; pct += 20) {
          await this.delay(120);
          emit({
            type: LauncherEventType.DownloadProgress,
            timestamp: now(),
            sessionId,
            payload: {
              packageId: mod,
              progress: { current: pct, total: 100, percent: pct, speed: 2 * 1024 * 1024, eta: Math.ceil((100 - pct) / 20) },
            },
          });
        }

        emit({
          type: LauncherEventType.DownloadComplete,
          timestamp: now(),
          sessionId,
          payload: { packageId: mod },
        });
        await this.delay(80);
      }

      emit({
        type: LauncherEventType.SessionComplete,
        timestamp: now(),
        sessionId,
        payload: { metadata: { ready: true } },
      });
    })();
  }

  private delay(ms: number) {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }
}
