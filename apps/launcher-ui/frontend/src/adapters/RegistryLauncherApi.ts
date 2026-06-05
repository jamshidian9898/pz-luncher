import { LauncherApi } from '../interfaces/LauncherApi';
import { ServerInfo, ServerDetails, SessionStatus } from '../types';
import { launcherEventBus } from './eventBus';
import { sseEventsApi } from './SseEventsApi';

const DEV_API_BASE = '/api';

function getBackendBase(): string {
  if (typeof window !== 'undefined' && (window as unknown as Record<string, unknown>).__BACKEND_URL__) {
    return (window as unknown as Record<string, string>).__BACKEND_URL__;
  }
  return 'http://localhost:8080';
}

async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, init);
  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || res.statusText);
  }
  return res.json() as Promise<T>;
}

interface BackendServer {
  id: string;
  name: string;
  description?: string;
  playerCount: number;
  maxPlayers: number;
  status?: string;
  tags?: string[];
}

export class RegistryLauncherApi implements LauncherApi {
  async getServerList(): Promise<ServerInfo[]> {
    const base = getBackendBase();
    const resp = await fetchJSON<{ servers: BackendServer[] }>(`${base}/api/v1/servers`);
    return resp.servers.map((d) => ({
      id: d.id,
      name: d.name,
      description: d.description ?? '',
      playerCount: d.playerCount ?? 0,
      maxPlayers: d.maxPlayers ?? 0,
      ping: 0,
      modCount: 0,
      installed: false,
      upToDate: false,
    }));
  }

  async getServerDetails(serverId: string): Promise<ServerDetails> {
    const base = getBackendBase();
    const d = await fetchJSON<BackendServer>(`${base}/api/v1/servers/${encodeURIComponent(serverId)}`);
    return {
      id: d.id,
      name: d.name,
      description: d.description ?? '',
      playerCount: d.playerCount ?? 0,
      maxPlayers: d.maxPlayers ?? 0,
      ping: 0,
      modCount: 0,
      installed: false,
      upToDate: false,
      mods: [],
      totalSize: 0,
      installedSize: 0,
      missingSize: 0,
    };
  }

  async joinServer(serverId: string): Promise<void> {
    const base = getBackendBase();
    const result = await fetchJSON<{ sessionId: string }>(
      `${base}/api/v1/join/${encodeURIComponent(serverId)}`,
      { method: 'POST' }
    );
    const { sessionId } = result;
    // Subscribe to SSE events from dev-api for download progress (A3: dev-api still owns SSE)
    sseEventsApi.subscribeSession(sessionId, (event) => {
      launcherEventBus.emit(event);
    });
  }

  async launchServer(serverId: string): Promise<void> {
    await fetchJSON(`${DEV_API_BASE}/launch/${encodeURIComponent(serverId)}`, { method: 'POST' });
  }

  async getSessionStatus(sessionId: string): Promise<SessionStatus> {
    return { sessionId, state: 'complete', progress: 100 };
  }
}
