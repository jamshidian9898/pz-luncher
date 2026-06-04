import { LauncherEvent } from './LauncherEvent';

export interface EventsApi {
  onLauncherEvent(cb: (event: LauncherEvent) => void): () => void;
}
