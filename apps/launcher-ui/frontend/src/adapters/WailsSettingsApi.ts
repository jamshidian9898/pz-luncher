import { SettingsApi } from '../interfaces/SettingsApi';
import { Settings } from '../types';

export class WailsSettingsApi implements SettingsApi {
  async getSettings(): Promise<Settings> {
    return window.go.main.App.GetSettings();
  }

  async saveSettings(settings: Settings): Promise<void> {
    return window.go.main.App.SaveSettings(settings);
  }
}
