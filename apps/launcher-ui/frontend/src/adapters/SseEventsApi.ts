import { EventsApi } from '../interfaces/EventsApi';
import { LauncherEvent, normalizeLauncherEvent } from '../interfaces/LauncherEvent';

const API_BASE = '/api';

/**
 * Connects to the dev-api SSE stream for a given sessionId.
 * Used by RegistryLauncherApi after POST /api/join returns a sessionId.
 */
export class SseEventsApi implements EventsApi {
  onLauncherEvent(cb: (event: LauncherEvent) => void): () => void {
    return () => undefined;
  }

  /**
   * Subscribe to events for a specific session via SSE.
   * Returns an unsubscribe function.
   */
  subscribeSession(sessionId: string, cb: (event: LauncherEvent) => void): () => void {
    const url = `${API_BASE}/events/${encodeURIComponent(sessionId)}`;
    const es = new EventSource(url);

    es.onmessage = (e) => {
      try {
        const raw = JSON.parse(e.data);
        cb(normalizeLauncherEvent(raw));
      } catch {
        // ignore parse errors
      }
    };

    es.onerror = () => {
      es.close();
    };

    return () => es.close();
  }
}

export const sseEventsApi = new SseEventsApi();
