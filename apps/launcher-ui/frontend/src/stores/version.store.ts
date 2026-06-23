import { create } from 'zustand';

export interface VersionSource {
  type: 'registry' | 'agent' | 'hoster';
  url: string;
  trustLevel: string;
  description: string;
}

export interface VersionCandidate {
  gameVersion: string;
  platform: string;
  sizeBytes: number;
  trustLevel: string;
  availableSources: VersionSource[];
  isLocal: boolean;
  localPath?: string;
}

export interface VersionSelector {
  required: string;
  localVersion?: string;
  candidates: VersionCandidate[];
  needDownload: boolean;
  autoSelected?: VersionSource;
}

interface VersionStore {
  selector?: VersionSelector;
  selectedSource?: VersionSource;
  loading: boolean;

  load: (requiredVersion: string) => Promise<void>;
  selectSource: (source: VersionSource) => void;
  confirm: (source: VersionSource) => Promise<void>;
}

export const useVersionStore = create<VersionStore>((set, get) => ({
  selector: undefined,
  selectedSource: undefined,
  loading: false,

  load: async (requiredVersion) => {
    set({ loading: true });
    try {
      const sel = await window.go.main.App.GetVersionSelector(requiredVersion);
      set({ selector: sel, selectedSource: sel.autoSelected, loading: false });
    } catch (e) {
      console.error('Failed to load version selector:', e);
      set({ loading: false });
    }
  },

  selectSource: (source) => {
    set({ selectedSource: source });
  },

  confirm: async (source) => {
    const sel = get().selector;
    if (!sel) return;

    try {
      // Find the candidate for this source
      const candidate = sel.candidates.find((c) =>
        c.availableSources.some((s) => s.url === source.url)
      );
      if (!candidate) return;

      await window.go.main.App.ConfirmVersionDownload(
        candidate.gameVersion,
        candidate.platform,
        source.url
      );
    } catch (e) {
      console.error('Failed to confirm version download:', e);
    }
  },
}));
