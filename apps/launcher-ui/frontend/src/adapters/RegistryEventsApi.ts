import { EventsApi } from '../interfaces/EventsApi';
import { LauncherEvent } from '../interfaces/LauncherEvent';
import { launcherEventBus } from './eventBus';

/** Uses shared bus fed by RegistryLauncherApi (and Wails when bridged). */
export class RegistryEventsApi implements EventsApi {
  onLauncherEvent(cb: (event: LauncherEvent) => void): () => void {
    return launcherEventBus.on(cb);
  }
}
