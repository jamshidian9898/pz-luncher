import { LauncherEvent } from '../interfaces/LauncherEvent';

type EventCallback = (event: LauncherEvent) => void;

export const launcherEventBus = {
  listeners: [] as EventCallback[],

  on(cb: EventCallback): () => void {
    this.listeners.push(cb);
    return () => {
      const i = this.listeners.indexOf(cb);
      if (i > -1) this.listeners.splice(i, 1);
    };
  },

  emit(event: LauncherEvent) {
    this.listeners.forEach((cb) => cb(event));
  },
};
