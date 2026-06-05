import { SettingsApi } from '../interfaces/SettingsApi';
import { Settings } from '../types';

const MOCK_SETTINGS: Settings = {
  gamePath: '',
  backendUrl: 'http://localhost:8080',
  cacheLocation: '~/PZLauncher/cache',
  profilesLocation: '~/PZLauncher/profiles',
  maxConcurrent: 3,
  bandwidthLimit: 0,
  verifyChecksum: true,
};

export class MockSettingsApi implements SettingsApi {
  async getSettings(): Promise<Settings> {
    return new Promise((resolve) => setTimeout(() => resolve({ ...MOCK_SETTINGS }), 150));
  }

  async saveSettings(settings: Settings): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, 200));
  }
}
