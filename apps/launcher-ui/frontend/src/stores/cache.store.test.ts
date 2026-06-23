import { describe, it, expect, beforeEach, vi } from 'vitest';
import { useCacheStore, CacheStats } from './cache.store';

// Mock window.go.main.App
const mockApp = {
  GetCacheStats: vi.fn(),
  DeleteCacheEntry: vi.fn(),
};

declare global {
  interface Window {
    go: {
      main: {
        App: typeof mockApp;
      };
    };
  }
}

window.go = {
  main: {
    App: mockApp,
  },
} as any;

describe('CacheStore', () => {
  beforeEach(() => {
    useCacheStore.setState({
      stats: undefined,
      loading: false,
      deleting: {},
    });
    vi.clearAllMocks();
  });

  it('should load cache stats', async () => {
    const mockStats: CacheStats = {
      totalBytes: 1000,
      versionBytes: 500,
      modBytes: 500,
      entries: [
        {
          type: 'version',
          key: '42.16',
          gameVersion: '42.16',
          sizeBytes: 500,
          downloadedAt: '2026-06-23T10:00:00Z',
          lastUsedAt: '2026-06-23T10:00:00Z',
          usedByProfiles: [],
        },
      ],
      deletableBytes: 0,
    };

    mockApp.GetCacheStats.mockResolvedValue(mockStats);

    const store = useCacheStore.getState();
    await store.load();

    const state = useCacheStore.getState();
    expect(state.stats).toEqual(mockStats);
    expect(state.loading).toBe(false);
  });

  it('should handle load error gracefully', async () => {
    mockApp.GetCacheStats.mockRejectedValue(new Error('Network error'));

    const store = useCacheStore.getState();
    await store.load();

    const state = useCacheStore.getState();
    expect(state.loading).toBe(false);
    expect(state.stats).toBeUndefined();
  });

  it('should delete cache entry and reload', async () => {
    const mockStats: CacheStats = {
      totalBytes: 500,
      versionBytes: 500,
      modBytes: 0,
      entries: [],
      deletableBytes: 0,
    };

    mockApp.DeleteCacheEntry.mockResolvedValue(undefined);
    mockApp.GetCacheStats.mockResolvedValue(mockStats);

    const store = useCacheStore.getState();
    await store.delete('version', '42.16');

    expect(mockApp.DeleteCacheEntry).toHaveBeenCalledWith('version', '42.16');
    expect(mockApp.GetCacheStats).toHaveBeenCalled();
  });

  it('should mark entry as deleting during operation', async () => {
    mockApp.DeleteCacheEntry.mockImplementation(
      () => new Promise((resolve) => setTimeout(resolve, 100))
    );
    mockApp.GetCacheStats.mockResolvedValue({
      totalBytes: 0,
      versionBytes: 0,
      modBytes: 0,
      entries: [],
      deletableBytes: 0,
    });

    const store = useCacheStore.getState();
    const deletePromise = store.delete('version', '42.16');

    // Check deleting state immediately
    let state = useCacheStore.getState();
    expect(state.deleting['42.16']).toBe(true);

    await deletePromise;

    // Check deleting state after completion
    state = useCacheStore.getState();
    expect(state.deleting['42.16']).toBeUndefined();
  });

  it('should handle delete error gracefully', async () => {
    mockApp.DeleteCacheEntry.mockRejectedValue(new Error('In use'));

    const store = useCacheStore.getState();
    await store.delete('version', '42.16');

    const state = useCacheStore.getState();
    expect(state.deleting['42.16']).toBeUndefined();
  });
});
