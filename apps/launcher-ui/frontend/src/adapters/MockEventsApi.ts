import { EventsApi } from '../interfaces/EventsApi';
import { LauncherEvent } from '../interfaces/LauncherEvent';
import { launcherEventBus } from './eventBus';

/** @deprecated Use launcherEventBus — kept for MockLauncherApi simulations */
export const mockEventBus = launcherEventBus;

export class MockEventsApi implements EventsApi {
  onLauncherEvent(cb: (event: LauncherEvent) => void): () => void {
    return launcherEventBus.on(cb);
  }
}
