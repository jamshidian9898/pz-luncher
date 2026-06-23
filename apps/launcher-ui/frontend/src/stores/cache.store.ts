import { create } from 'zustand';

export interface CacheEntry {
  type: 'version' | 'mod';
  key: string;
  platform?: string;
  gameVersion?: string;
  modId?: string;
  sizeBytes: number;
  downloadedAt: string;
  lastUsedAt: string;
  usedByProfiles: string[];
}

export interface CacheStats {
  totalBytes: number;
  versionBytes: number;
  modBytes: number;
  entries: CacheEntry[];
  deletableBytes: number;
}

interface CacheStore {
  stats?: CacheStats;
  loading: boolean;
  deleting: Record<string, boolean>;

  load: () => Promise<void>;
  delete: (type: 'version' | 'mod', key: string) => Promise<void>;
}

export const useCacheStore = create<CacheStore>((set, get) => ({
  stats: undefined,
  loading: false,
  deleting: {},

  load: async () => {
    set({ loading: true });
    try {
      const stats = await window.go.main.App.GetCacheStats();
      set({ stats, loading: false });
    } catch (e) {
      console.error('Failed to load cache stats:', e);
      set({ loading: false });
    }
  },

  delete: async (type, key) => {
    set((s) => ({ deleting: { ...s.deleting, [key]: true } }));
    try {
      await window.go.main.App.DeleteCacheEntry(type, key);
      get().load();
    } catch (e) {
      console.error('Failed to delete cache entry:', e);
    } finally {
      set((s) => {
        const d = { ...s.deleting };
        delete d[key];
        return { deleting: d };
      });
    }
  },
}));
