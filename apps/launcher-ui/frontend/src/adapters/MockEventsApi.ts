import { EventsApi } from '../interfaces/EventsApi';
import { LauncherEvent } from '../interfaces/LauncherEvent';

type EventCallback = (event: LauncherEvent) => void;

export const mockEventBus = {
  listeners: [] as EventCallback[],

  on(cb: EventCallback): () => void {
    this.listeners.push(cb);
    return () => {
      const index = this.listeners.indexOf(cb);
      if (index > -1) {
        this.listeners.splice(index, 1);
      }
    };
  },

  emit(event: LauncherEvent) {
    this.listeners.forEach((cb) => cb(event));
  },
};

export class MockEventsApi implements EventsApi {
  onLauncherEvent(cb: (event: LauncherEvent) => void): () => void {
    return mockEventBus.on(cb);
  }
}
