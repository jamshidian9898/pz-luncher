import { Settings } from '../types';

export interface SettingsApi {
  getSettings(): Promise<Settings>;
  saveSettings(settings: Settings): Promise<void>;
}
