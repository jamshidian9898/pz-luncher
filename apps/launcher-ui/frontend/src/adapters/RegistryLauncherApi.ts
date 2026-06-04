import { LauncherApi } from '../interfaces/LauncherApi';
import { ServerManifest, ServerRegistry } from '../contracts/generated';
import { ServerInfo, ServerDetails, SessionStatus, ModInfo } from '../types';
import { LauncherEvent, LauncherEventType } from '../interfaces/LauncherEvent';
import { launcherEventBus } from './eventBus';

const REGISTRY_BASE = '/registry';
const API_BASE = '/api';

async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, init);
  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || res.statusText);
  }
  return res.json() as Promise<T>;
}

export class RegistryLauncherApi implements LauncherApi {
  async getServerList(): Promise<ServerInfo[]> {
    const reg = await fetchJSON<ServerRegistry>(`${REGISTRY_BASE}/servers.json`);
    const servers: ServerInfo[] = [];
    for (const d of reg.servers) {
      let modCount = 0;
      try {
        const m = await this.loadManifest(d.manifestPath);
        modCount = m.mods.length;
      } catch {
        /* ignore */
      }
      servers.push({
        id: d.id,
        name: d.name,
        description: d.description ?? '',
        playerCount: d.playerCount ?? 0,
        maxPlayers: d.maxPlayers ?? 0,
        ping: d.ping ?? 0,
        modCount,
        installed: false,
        upToDate: false,
      });
    }
    return servers;
  }

  async getServerDetails(serverId: string): Promise<ServerDetails> {
    const reg = await fetchJSON<ServerRegistry>(`${REGISTRY_BASE}/servers.json`);
    const desc = reg.servers.find((s) => s.id === serverId);
    if (!desc) throw new Error(`server not found: ${serverId}`);

    const manifest = await this.loadManifest(desc.manifestPath);
    const mods: ModInfo[] = manifest.mods.map((m) => ({
      id: m.id,
      name: m.name,
      workshopId: m.workshopId ?? '',
      size: m.sizeBytes ?? 0,
      installed: false,
      upToDate: false,
      required: !m.optional,
    }));
    const totalSize = mods.reduce((a, m) => a + m.size, 0);

    return {
      id: desc.id,
      name: desc.name,
      description: desc.description ?? '',
      playerCount: desc.playerCount ?? 0,
      maxPlayers: desc.maxPlayers ?? 0,
      ping: desc.ping ?? 0,
      modCount: mods.length,
      installed: false,
      upToDate: false,
      mods,
      totalSize,
      installedSize: 0,
      missingSize: totalSize,
    };
  }

  async joinServer(serverId: string): Promise<void> {
    const sessionId = `session-${Date.now()}`;
    const now = () => Math.floor(Date.now() / 1000);
    const emit = (e: LauncherEvent) => launcherEventBus.emit(e);

    emit({
      type: LauncherEventType.SessionStart,
      timestamp: now(),
      sessionId,
      payload: { metadata: { serverId } },
    });
    emit({
      type: LauncherEventType.ModResolveStart,
      timestamp: now(),
      sessionId,
    });

    try {
      const result = await fetchJSON<{ sessionId: string; ready: boolean }>(
        `${API_BASE}/join/${encodeURIComponent(serverId)}`,
        { method: 'POST' }
      );

      emit({
        type: LauncherEventType.ModResolveComplete,
        timestamp: now(),
        sessionId: result.sessionId || sessionId,
        payload: { metadata: { serverId } },
      });
      emit({
        type: LauncherEventType.SessionComplete,
        timestamp: now(),
        sessionId: result.sessionId || sessionId,
        payload: { metadata: { ready: result.ready } },
      });
    } catch (err) {
      emit({
        type: LauncherEventType.Error,
        timestamp: now(),
        sessionId,
        payload: { error: err instanceof Error ? err.message : String(err) },
      });
      throw err;
    }
  }

  async launchServer(serverId: string): Promise<void> {
    await fetchJSON(`${API_BASE}/launch/${encodeURIComponent(serverId)}`, { method: 'POST' });
  }

  async getSessionStatus(sessionId: string): Promise<SessionStatus> {
    return { sessionId, state: 'complete', progress: 100 };
  }

  private async loadManifest(manifestPath: string): Promise<ServerManifest> {
    const path = manifestPath.startsWith('/') ? manifestPath : `${REGISTRY_BASE}/${manifestPath}`;
    return fetchJSON<ServerManifest>(path);
  }
}
