import { describe, it, expect, beforeEach, vi } from 'vitest';
import { useVersionStore, VersionSelector } from './version.store';

const mockApp = {
  GetVersionSelector: vi.fn(),
  ConfirmVersionDownload: vi.fn(),
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

describe('VersionStore', () => {
  beforeEach(() => {
    useVersionStore.setState({
      selector: undefined,
      selectedSource: undefined,
      loading: false,
    });
    vi.clearAllMocks();
  });

  it('should load version selector', async () => {
    const mockSelector: VersionSelector = {
      required: '42.16',
      localVersion: '',
      candidates: [
        {
          gameVersion: '42.16',
          platform: 'windows-x64',
          sizeBytes: 3200000000,
          trustLevel: 'verified',
          availableSources: [
            {
              type: 'registry',
              url: 'http://localhost:8080/api/v1/registry/versions/pz/42.16/windows-x64',
              trustLevel: 'verified',
              description: 'PZ Registry (verified)',
            },
          ],
          isLocal: false,
        },
      ],
      needDownload: true,
      autoSelected: {
        type: 'registry',
        url: 'http://localhost:8080/api/v1/registry/versions/pz/42.16/windows-x64',
        trustLevel: 'verified',
        description: 'PZ Registry (verified)',
      },
    };

    mockApp.GetVersionSelector.mockResolvedValue(mockSelector);

    const store = useVersionStore.getState();
    await store.load('42.16');

    const state = useVersionStore.getState();
    expect(state.selector).toEqual(mockSelector);
    expect(state.selectedSource).toEqual(mockSelector.autoSelected);
    expect(state.loading).toBe(false);
  });

  it('should auto-select first available source', async () => {
    const source1 = {
      type: 'registry' as const,
      url: 'http://registry.example.com/v42.16',
      trustLevel: 'verified',
      description: 'Registry',
    };

    const source2 = {
      type: 'agent' as const,
      url: 'http://agent.example.com/content/v42.16',
      trustLevel: 'verified',
      description: 'Agent',
    };

    const mockSelector: VersionSelector = {
      required: '42.16',
      candidates: [
        {
          gameVersion: '42.16',
          platform: 'windows-x64',
          sizeBytes: 3200000000,
          trustLevel: 'verified',
          availableSources: [source1, source2],
          isLocal: false,
        },
      ],
      needDownload: true,
      autoSelected: source1,
    };

    mockApp.GetVersionSelector.mockResolvedValue(mockSelector);

    const store = useVersionStore.getState();
    await store.load('42.16');

    const state = useVersionStore.getState();
    expect(state.selectedSource).toEqual(source1);
  });

  it('should allow manual source selection', async () => {
    const mockSelector: VersionSelector = {
      required: '42.16',
      candidates: [
        {
          gameVersion: '42.16',
          platform: 'windows-x64',
          sizeBytes: 3200000000,
          trustLevel: 'verified',
          availableSources: [
            {
              type: 'registry',
              url: 'http://registry.example.com/v42.16',
              trustLevel: 'verified',
              description: 'Registry',
            },
            {
              type: 'agent',
              url: 'http://agent.example.com/content/v42.16',
              trustLevel: 'verified',
              description: 'Agent',
            },
          ],
          isLocal: false,
        },
      ],
      needDownload: true,
    };

    mockApp.GetVersionSelector.mockResolvedValue(mockSelector);

    const store = useVersionStore.getState();
    await store.load('42.16');

    const agentSource = mockSelector.candidates[0].availableSources[1];
    store.selectSource(agentSource);

    const state = useVersionStore.getState();
    expect(state.selectedSource).toEqual(agentSource);
  });

  it('should confirm version download', async () => {
    const mockSelector: VersionSelector = {
      required: '42.16',
      candidates: [
        {
          gameVersion: '42.16',
          platform: 'windows-x64',
          sizeBytes: 3200000000,
          trustLevel: 'verified',
          availableSources: [
            {
              type: 'registry',
              url: 'http://registry.example.com/v42.16',
              trustLevel: 'verified',
              description: 'Registry',
            },
          ],
          isLocal: false,
        },
      ],
      needDownload: true,
      autoSelected: {
        type: 'registry',
        url: 'http://registry.example.com/v42.16',
        trustLevel: 'verified',
        description: 'Registry',
      },
    };

    mockApp.GetVersionSelector.mockResolvedValue(mockSelector);
    mockApp.ConfirmVersionDownload.mockResolvedValue(undefined);

    const store = useVersionStore.getState();
    await store.load('42.16');

    const source = mockSelector.autoSelected!;
    await store.confirm(source);

    expect(mockApp.ConfirmVersionDownload).toHaveBeenCalledWith(
      '42.16',
      'windows-x64',
      'http://registry.example.com/v42.16'
    );
  });

  it('should handle load error gracefully', async () => {
    mockApp.GetVersionSelector.mockRejectedValue(new Error('Network error'));

    const store = useVersionStore.getState();
    await store.load('42.16');

    const state = useVersionStore.getState();
    expect(state.loading).toBe(false);
    expect(state.selector).toBeUndefined();
  });
});
