import { WailsLauncherApi } from './WailsLauncherApi';
import { RegistryLauncherApi } from './RegistryLauncherApi';
import { WailsEventsApi } from './WailsEventsApi';
import { RegistryEventsApi } from './RegistryEventsApi';
import { WailsSettingsApi } from './WailsSettingsApi';
import { RegistrySettingsApi } from './RegistrySettingsApi';

/** Wails desktop bindings, or registry + dev-api (default for Vite dev). */
export const createLauncherApi = (isWails: boolean) =>
  isWails ? new WailsLauncherApi() : new RegistryLauncherApi();

export const createEventsApi = (isWails: boolean) =>
  isWails ? new WailsEventsApi() : new RegistryEventsApi();

export const createSettingsApi = (isWails: boolean) =>
  isWails ? new WailsSettingsApi() : new RegistrySettingsApi();
