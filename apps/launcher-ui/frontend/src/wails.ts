import { createLauncherApi, createEventsApi, createSettingsApi } from './adapters';

export const isWails = (): boolean =>
  typeof window !== 'undefined' &&
  typeof (window as any).go !== 'undefined' &&
  (window as any).go?.main?.App != null;

export const launcherApi = createLauncherApi(true);
export const eventsApi = createEventsApi(true);
export const settingsApi = createSettingsApi(true);
