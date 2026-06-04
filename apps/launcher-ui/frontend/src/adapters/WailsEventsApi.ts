import { EventsApi } from '../interfaces/EventsApi';
import { LauncherEvent, normalizeLauncherEvent } from '../interfaces/LauncherEvent';

export class WailsEventsApi implements EventsApi {
  onLauncherEvent(cb: (event: LauncherEvent) => void): () => void {
    if (typeof window !== 'undefined' && window.runtime?.EventsOn) {
      return window.runtime.EventsOn('launcher:event', (data: unknown) => {
        const event = normalizeLauncherEvent(data);
        cb(event);
      });
    }

    return () => undefined;
  }
}
