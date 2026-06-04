import { SettingsApi } from '../interfaces/SettingsApi';
import { LauncherSettings, settingsFromLauncher, settingsToLauncher } from '../contracts/generated';
import { Settings } from '../types';

const API_BASE = '/api';

export class RegistrySettingsApi implements SettingsApi {
  async getSettings(): Promise<Settings> {
    const raw = await fetch(`${API_BASE}/settings`).then((r) => {
      if (!r.ok) throw new Error(r.statusText);
      return r.json() as Promise<LauncherSettings>;
    });
    return settingsFromLauncher(raw);
  }

  async saveSettings(settings: Settings): Promise<void> {
    const body = settingsToLauncher(settings);
    const res = await fetch(`${API_BASE}/settings`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!res.ok) throw new Error(await res.text());
  }
}
