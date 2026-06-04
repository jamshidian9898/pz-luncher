import { WailsLauncherApi } from './WailsLauncherApi';
import { MockLauncherApi } from './MockLauncherApi';
import { WailsEventsApi } from './WailsEventsApi';
import { MockEventsApi } from './MockEventsApi';
import { WailsSettingsApi } from './WailsSettingsApi';
import { MockSettingsApi } from './MockSettingsApi';

export const createLauncherApi = (isWails: boolean) =>
  isWails ? new WailsLauncherApi() : new MockLauncherApi();

export const createEventsApi = (isWails: boolean) =>
  isWails ? new WailsEventsApi() : new MockEventsApi();

export const createSettingsApi = (isWails: boolean) =>
  isWails ? new WailsSettingsApi() : new MockSettingsApi();
