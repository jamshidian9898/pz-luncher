import { create } from 'zustand';

export type TrustLevel = 'unknown' | 'pending' | 'verified' | 'rejected';

export interface ContributeEntry {
  gameVersion: string;
  platform: string;
  localPath: string;
  sizeBytes: number;
  sha256?: string;
  trustLevel: TrustLevel;
  uploadCount: number;
  source: 'cache' | 'gamePath';
}

export type EntryState =
  | 'idle'
  | 'hashing'
  | 'submitting'
  | 'upload_required'
  | 'uploading'
  | 'done'
  | 'error';

export interface EntryProgress {
  state: EntryState;
  percent: number;
  error?: string;
  submitResult?: {
    status: string;
    trustLevel: TrustLevel;
    uploadCount: number;
  };
}

interface ContributeStore {
  entries: ContributeEntry[];
  progress: Record<string, EntryProgress>; // keyed by gameVersion
  loading: boolean;
  backendUrl: string;

  load: () => Promise<void>;
  setProgress: (version: string, p: Partial<EntryProgress>) => void;
  contribute: (entry: ContributeEntry) => Promise<void>;
}

function key(e: ContributeEntry) {
  return e.gameVersion;
}

export const useContributeStore = create<ContributeStore>((set, get) => ({
  entries: [],
  progress: {},
  loading: false,
  backendUrl: '',

  load: async () => {
    set({ loading: true });
    try {
      const result = await window.go.main.App.GetContributeStatus();
      const entries = (result.entries ?? []).map((e) => ({
        ...e,
        trustLevel: (e.trustLevel as TrustLevel) ?? 'unknown',
      }));
      set({ entries, backendUrl: result.backendUrl ?? '', loading: false });
    } catch (e) {
      set({ loading: false });
    }
  },

  setProgress: (version, p) => {
    set((s) => ({
      progress: {
        ...s.progress,
        [version]: { ...s.progress[version], ...p },
      },
    }));
  },

  contribute: async (entry) => {
    const { setProgress } = get();
    const version = key(entry);

    // Step 1: Hash the binary if we don't already have a SHA256
    let sha256 = entry.sha256 ?? '';
    let sizeBytes = entry.sizeBytes;
    if (!sha256) {
      setProgress(version, { state: 'hashing', percent: 0 });
      try {
        const hashResult = await window.go.main.App.HashGameVersion(entry.localPath);
        sha256 = hashResult.sha256;
        sizeBytes = hashResult.sizeBytes;
      } catch (e) {
        setProgress(version, { state: 'error', error: String(e) });
        return;
      }
    }

    // Step 2: Submit hash to registry
    setProgress(version, { state: 'submitting', percent: 0 });
    let submitResult;
    try {
      submitResult = await window.go.main.App.SubmitVersionHash(
        'pz', entry.gameVersion, entry.platform, sha256, sizeBytes
      );
    } catch (e) {
      setProgress(version, { state: 'error', error: String(e) });
      return;
    }

    if (submitResult.error) {
      setProgress(version, { state: 'error', error: submitResult.error });
      return;
    }

    // Step 3: Upload binary if backend needs it
    if (submitResult.status === 'upload_required') {
      setProgress(version, { state: 'upload_required', percent: 0 });
      try {
        await window.go.main.App.UploadVersionBinary(
          'pz', entry.gameVersion, entry.platform, entry.localPath, sha256, sizeBytes
        );
      } catch (e) {
        setProgress(version, { state: 'error', error: String(e) });
        return;
      }
    }

    setProgress(version, {
      state: 'done',
      percent: 100,
      submitResult: {
        status: submitResult.status,
        trustLevel: submitResult.trustLevel as TrustLevel,
        uploadCount: submitResult.uploadCount,
      },
    });

    // Refresh entries to show updated trust level
    get().load();
  },
}));
