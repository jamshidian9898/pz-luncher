import { createLauncherApi, createEventsApi, createSettingsApi } from './adapters';

export const isWails = (): boolean =>
  typeof window !== 'undefined' &&
  typeof window.go !== 'undefined' &&
  window.go?.main?.App != null;

export const launcherApi = createLauncherApi(isWails());
export const eventsApi = createEventsApi(isWails());
export const settingsApi = createSettingsApi(isWails());
