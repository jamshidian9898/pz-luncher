import { useEffect, useState } from 'react';
import { Download, AlertTriangle, CheckCircle, RefreshCw } from 'lucide-react';
import { useVersionStore, VersionSource } from '../stores/version.store';
import { formatBytes } from '../utils/format';

interface VersionSelectorProps {
  requiredVersion: string;
  onDownloadStart?: () => void;
  onClose?: () => void;
}

export function VersionSelector({ requiredVersion, onDownloadStart, onClose }: VersionSelectorProps) {
  const { selector, selectedSource, loading, load, selectSource, confirm } = useVersionStore();
  const [confirming, setConfirming] = useState(false);

  useEffect(() => {
    load(requiredVersion);
  }, [requiredVersion, load]);

  if (loading || !selector) {
    return (
      <div className="flex items-center justify-center h-64 text-slate-400 gap-2">
        <RefreshCw size={16} className="animate-spin" />
        بارگذاری نسخه‌های دسترس...
      </div>
    );
  }

  // If no download needed, just show confirmation
  if (!selector.needDownload) {
    return (
      <div className="space-y-4">
        <div className="flex items-center gap-3 text-emerald-400">
          <CheckCircle size={20} />
          <span>v{selector.localVersion} روی این دستگاه موجود است</span>
        </div>
        <button
          onClick={onClose}
          className="px-4 py-2 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg"
        >
          ادامه
        </button>
      </div>
    );
  }

  // No sources available
  if (selector.candidates.length === 0) {
    return (
      <div className="space-y-4 bg-red-900/20 border border-red-700 rounded-lg p-4">
        <div className="flex items-start gap-3">
          <AlertTriangle size={20} className="text-red-400 mt-0.5" />
          <div className="space-y-1">
            <p className="font-medium text-red-300">نسخه v{requiredVersion} در دسترس نیست</p>
            <p className="text-sm text-red-200">
              این سرور نسخه {requiredVersion} را می‌خواهد، اما هیچ منبع برای دانلود پیدا نشد.
            </p>
            <p className="text-xs text-red-300 mt-2">
              با مدیر سرور تماس بگیر تا این نسخه را در Registry ثبت کند.
            </p>
          </div>
        </div>
      </div>
    );
  }

  // Show available sources for the candidate
  const candidate = selector.candidates[0];

  return (
    <div className="space-y-4">
      <div className="bg-slate-800 border border-slate-700 rounded-lg p-4">
        <h3 className="font-semibold text-slate-100 mb-2">دانلود v{candidate.gameVersion}</h3>
        <p className="text-sm text-slate-400 mb-4">
          {formatBytes(candidate.sizeBytes)} • وضعیت: {trustBadgeText(candidate.trustLevel)}
        </p>

        <div className="space-y-2">
          {candidate.availableSources.map((source, idx) => (
            <SourceOption
              key={idx}
              source={source}
              selected={selectedSource?.url === source.url}
              isAuto={selectedSource === undefined && idx === 0}
              onSelect={() => selectSource(source)}
            />
          ))}
        </div>
      </div>

      <div className="flex gap-3">
        <button
          onClick={async () => {
            if (!selectedSource) return;
            setConfirming(true);
            await confirm(selectedSource);
            setConfirming(false);
            onDownloadStart?.();
          }}
          disabled={!selectedSource || confirming}
          className="flex-1 flex items-center justify-center gap-2 px-4 py-2 bg-emerald-600 hover:bg-emerald-500 disabled:bg-slate-700 disabled:text-slate-500 text-white rounded-lg transition-colors"
        >
          {confirming ? (
            <RefreshCw size={16} className="animate-spin" />
          ) : (
            <Download size={16} />
          )}
          {confirming ? 'در حال شروع...' : 'دانلود و ادامه'}
        </button>
        {onClose && (
          <button
            onClick={onClose}
            className="px-4 py-2 bg-slate-700 hover:bg-slate-600 text-slate-300 rounded-lg"
          >
            لغو
          </button>
        )}
      </div>
    </div>
  );
}

interface SourceOptionProps {
  source: VersionSource;
  selected: boolean;
  isAuto: boolean;
  onSelect: () => void;
}

function SourceOption({ source, selected, isAuto, onSelect }: SourceOptionProps) {
  return (
    <button
      onClick={onSelect}
      className={`w-full text-left p-3 rounded-lg border transition-all ${
        selected
          ? 'bg-emerald-900/30 border-emerald-600'
          : 'bg-slate-700/50 border-slate-600 hover:border-slate-500'
      }`}
    >
      <div className="flex items-start justify-between">
        <div className="space-y-0.5">
          <div className="flex items-center gap-2">
            <input
              type="radio"
              checked={selected}
              onChange={onSelect}
              className="w-4 h-4"
            />
            <span className="font-medium text-slate-100">{source.description}</span>
            {isAuto && (
              <span className="text-xs text-emerald-400 bg-emerald-900/30 px-2 py-0.5 rounded">
                بهترین
              </span>
            )}
          </div>
          <p className="text-xs text-slate-500 ml-6">
            {source.type === 'registry' && 'از پایگاه داده PZ'}
            {source.type === 'agent' && 'از سرور مستقیم'}
            {source.type === 'hoster' && 'از آپلود hoster (تأیید نشده)'}
          </p>
        </div>
        <TrustBadge trustLevel={source.trustLevel} />
      </div>
    </button>
  );
}

function TrustBadge({ trustLevel }: { trustLevel: string }) {
  switch (trustLevel) {
    case 'verified':
      return (
        <span className="text-xs text-emerald-400 flex items-center gap-1">
          <CheckCircle size={12} /> تأیید شده
        </span>
      );
    case 'pending':
      return (
        <span className="text-xs text-amber-400">
          در انتظار
        </span>
      );
    default:
      return null;
  }
}

function trustBadgeText(trustLevel: string): string {
  switch (trustLevel) {
    case 'verified':
      return '✅ تأیید شده';
    case 'pending':
      return '⏳ در انتظار';
    default:
      return '❓ نامشخص';
  }
}
