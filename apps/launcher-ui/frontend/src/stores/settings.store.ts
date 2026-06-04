import { create } from 'zustand';
import { devtools } from 'zustand/middleware';
import { Settings } from '../types';
import { settingsApi } from '../wails';

export interface SettingsState {
  settings: Settings | null;
  loading: boolean;
  error: string | null;

  fetchSettings: () => Promise<void>;
  saveSettings: (settings: Settings) => Promise<void>;
  setSettings: (settings: Settings | null) => void;
}

export const useSettingsStore = create<SettingsState>()(
  devtools(
    (set: (fn: (state: SettingsState) => Partial<SettingsState>) => void) => ({
      settings: null,
      loading: false,
      error: null,

      fetchSettings: async () => {
        set(() => ({ loading: true, error: null }));
        try {
          const data = await settingsApi.getSettings();
          set(() => ({ settings: data, loading: false }));
        } catch (err) {
          set(() => ({ error: 'Failed to load settings', loading: false }));
          console.error('Failed to load settings:', err);
        }
      },

      saveSettings: async (settings: Settings) => {
        set(() => ({ loading: true, error: null }));
        try {
          await settingsApi.saveSettings(settings);
          set(() => ({ settings, loading: false }));
        } catch (err) {
          set(() => ({ error: 'Failed to save settings', loading: false }));
          console.error('Failed to save settings:', err);
        }
      },

      setSettings: (settings: Settings | null) => set(() => ({ settings })),
    }),
    { name: 'settings-store' }
  )
);
