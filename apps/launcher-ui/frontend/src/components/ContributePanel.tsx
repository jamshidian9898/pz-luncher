import { useEffect } from 'react';
import { Upload, CheckCircle, Clock, AlertTriangle, HelpCircle, RefreshCw } from 'lucide-react';
import { useContributeStore, ContributeEntry, TrustLevel, EntryProgress } from '../stores/contribute.store';

export function ContributePanel() {
  const { entries, progress, loading, load, contribute } = useContributeStore();

  useEffect(() => {
    load();

    // Listen for hashing/upload progress events from Go
    const off = window.runtime.EventsOn('contribute:event', (data: unknown) => {
      const d = data as { type: string; percent: number; version?: string };
      // Progress events update the currently-active entry
      // The store setProgress is called indirectly via contribute()
      // We re-use the event to update UI progress
      useContributeStore.getState().setProgress(
        // find the entry currently in hashing/uploading state
        Object.keys(useContributeStore.getState().progress).find((v) => {
          const p = useContributeStore.getState().progress[v];
          return p.state === 'hashing' || p.state === 'uploading';
        }) ?? '',
        { percent: d.percent }
      );
    });
    return () => off?.();
  }, [load]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-48 text-slate-400 gap-2">
        <RefreshCw size={16} className="animate-spin" />
        در حال بررسی نسخه‌های محلی…
      </div>
    );
  }

  if (entries.length === 0) {
    return (
      <div className="max-w-xl space-y-4">
        <SectionHeader />
        <div className="bg-slate-800 border border-slate-700 rounded-lg p-8 text-center text-slate-400">
          <HelpCircle size={32} className="mx-auto mb-3 opacity-40" />
          <p className="text-sm">هیچ نسخه‌ای روی این دستگاه پیدا نشد.</p>
          <p className="text-xs mt-1 text-slate-500">
            مسیر نصب بازی را در تنظیمات وارد کنید.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-2xl space-y-6">
      <SectionHeader />

      <div className="bg-slate-800 border border-slate-700 rounded-lg p-4 text-sm text-slate-400 space-y-1">
        <p>
          با ارسال hash نسخه بازی به تأیید جمعی کمک می‌کنی. نیازی به ثبت‌نام نیست.
        </p>
        <p className="text-xs text-slate-500">
          ۳ کاربر مستقل با hash یکسان → <span className="text-emerald-400">Verified ✅</span>
        </p>
      </div>

      <div className="space-y-3">
        {entries.map((entry) => (
          <EntryCard
            key={entry.gameVersion}
            entry={entry}
            progress={progress[entry.gameVersion]}
            onContribute={() => contribute(entry)}
          />
        ))}
      </div>

      <button
        onClick={load}
        className="flex items-center gap-2 text-xs text-slate-500 hover:text-slate-300 transition-colors"
      >
        <RefreshCw size={12} />
        بروزرسانی وضعیت
      </button>
    </div>
  );
}

function SectionHeader() {
  return (
    <div>
      <h2 className="text-lg font-semibold text-slate-100 flex items-center gap-2">
        <Upload size={18} className="text-emerald-400" />
        کمک به پایگاه نسخه‌ها
      </h2>
      <p className="text-sm text-slate-400 mt-1">
        نسخه‌های بازی که روی این دستگاه داری را برای تأیید جمعی ارسال کن.
      </p>
    </div>
  );
}

interface EntryCardProps {
  entry: ContributeEntry;
  progress?: EntryProgress;
  onContribute: () => void;
}

function EntryCard({ entry, progress, onContribute }: EntryCardProps) {
  const state = progress?.state ?? 'idle';
  const percent = progress?.percent ?? 0;
  const busy = ['hashing', 'submitting', 'uploading'].includes(state);
  const done = state === 'done';

  return (
    <div className="bg-slate-800 border border-slate-700 rounded-lg p-4 space-y-3">
      {/* Header row */}
      <div className="flex items-start justify-between gap-4">
        <div className="space-y-0.5">
          <div className="flex items-center gap-2">
            <span className="font-mono text-sm font-semibold text-slate-100">
              v{entry.gameVersion}
            </span>
            <span className="text-xs text-slate-500">{entry.platform}</span>
            <SourceBadge source={entry.source} />
          </div>
          <div className="flex items-center gap-3 text-xs text-slate-500">
            <span>{formatBytes(entry.sizeBytes)}</span>
            <TrustBadge
              trustLevel={done && progress?.submitResult ? progress.submitResult.trustLevel : entry.trustLevel}
              uploadCount={done && progress?.submitResult ? progress.submitResult.uploadCount : entry.uploadCount}
            />
          </div>
        </div>

        <ContributeButton
          state={state}
          done={done}
          busy={busy}
          onClick={onContribute}
        />
      </div>

      {/* Progress bar */}
      {busy && (
        <div className="space-y-1">
          <div className="flex justify-between text-xs text-slate-500">
            <span>{stateLabel(state)}</span>
            <span>{percent}%</span>
          </div>
          <div className="h-1.5 bg-slate-700 rounded-full overflow-hidden">
            <div
              className="h-full bg-emerald-500 rounded-full transition-all duration-300"
              style={{ width: `${percent}%` }}
            />
          </div>
        </div>
      )}

      {/* Error */}
      {state === 'error' && progress?.error && (
        <div className="flex items-start gap-2 text-xs text-red-400 bg-red-900/20 rounded-lg p-2">
          <AlertTriangle size={12} className="mt-0.5 shrink-0" />
          <span>{progress.error}</span>
        </div>
      )}

      {/* Done */}
      {done && (
        <div className="flex items-center gap-2 text-xs text-emerald-400">
          <CheckCircle size={12} />
          <span>
            {progress?.submitResult?.status === 'upload_required'
              ? 'آپلود شد — hash ثبت شد'
              : 'hash ثبت شد — ممنون از کمکت'}
          </span>
        </div>
      )}
    </div>
  );
}

function ContributeButton({ state, done, busy, onClick }: {
  state: string; done: boolean; busy: boolean; onClick: () => void;
}) {
  if (done) {
    return (
      <div className="flex items-center gap-1 text-xs text-emerald-400 shrink-0">
        <CheckCircle size={14} />
        ارسال شد
      </div>
    );
  }
  return (
    <button
      onClick={onClick}
      disabled={busy}
      className={`shrink-0 flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors ${
        busy
          ? 'bg-slate-700 text-slate-500 cursor-not-allowed'
          : 'bg-emerald-600 hover:bg-emerald-500 text-white'
      }`}
    >
      {busy ? (
        <RefreshCw size={12} className="animate-spin" />
      ) : (
        <Upload size={12} />
      )}
      {busy ? stateLabel(state) : 'ارسال'}
    </button>
  );
}

function TrustBadge({ trustLevel, uploadCount }: { trustLevel: TrustLevel; uploadCount: number }) {
  switch (trustLevel) {
    case 'verified':
      return (
        <span className="flex items-center gap-1 text-emerald-400">
          <CheckCircle size={11} /> تأییدشده
        </span>
      );
    case 'pending':
      return (
        <span className="flex items-center gap-1 text-amber-400">
          <Clock size={11} /> در انتظار ({uploadCount}/3)
        </span>
      );
    case 'rejected':
      return (
        <span className="flex items-center gap-1 text-red-400">
          <AlertTriangle size={11} /> تعارض hash
        </span>
      );
    default:
      return (
        <span className="flex items-center gap-1 text-slate-500">
          <HelpCircle size={11} /> نامشخص
        </span>
      );
  }
}

function SourceBadge({ source }: { source: 'cache' | 'gamePath' }) {
  return (
    <span className={`text-xs px-1.5 py-0.5 rounded ${
      source === 'cache'
        ? 'bg-slate-700 text-slate-400'
        : 'bg-blue-900/40 text-blue-400'
    }`}>
      {source === 'cache' ? 'cache' : 'نصب اصلی'}
    </span>
  );
}

function stateLabel(state: string): string {
  switch (state) {
    case 'hashing': return 'محاسبه hash…';
    case 'submitting': return 'ارسال hash…';
    case 'upload_required': return 'آماده آپلود…';
    case 'uploading': return 'در حال آپلود…';
    default: return state;
  }
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '—';
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}
