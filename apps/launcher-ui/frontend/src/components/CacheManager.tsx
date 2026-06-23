import { useEffect } from 'react';
import { Trash2, RefreshCw, HardDrive, AlertTriangle, Check } from 'lucide-react';
import { useCacheStore, CacheEntry } from '../stores/cache.store';

export function CacheManager() {
  const { stats, loading, deleting, load } = useCacheStore();

  useEffect(() => {
    load();
  }, [load]);

  if (loading || !stats) {
    return (
      <div className="flex items-center justify-center h-48 text-slate-400 gap-2">
        <RefreshCw size={16} className="animate-spin" />
        بارگذاری فضا…
      </div>
    );
  }

  return (
    <div className="max-w-3xl space-y-6">
      {/* Header */}
      <div>
        <h2 className="text-lg font-semibold text-slate-100 flex items-center gap-2 mb-2">
          <HardDrive size={18} className="text-emerald-400" />
          مدیریت فضای ذخیره‌سازی
        </h2>
        <p className="text-sm text-slate-400">
          نسخه‌ها و مود‌هایی که روی این دستگاه ذخیره شده‌اند.
        </p>
      </div>

      {/* Summary stats */}
      <div className="grid grid-cols-3 gap-3">
        <StatCard label="کل استفاده" bytes={stats.totalBytes} />
        <StatCard label="نسخه‌های بازی" bytes={stats.versionBytes} />
        <StatCard label="مود‌ها" bytes={stats.modBytes} />
      </div>

      {/* Deletable suggestion */}
      {stats.deletableBytes > 0 && (
        <div className="bg-amber-900/20 border border-amber-700 rounded-lg p-4 flex items-start gap-3">
          <AlertTriangle size={16} className="text-amber-400 mt-0.5 shrink-0" />
          <div className="space-y-1">
            <p className="text-sm font-medium text-amber-300">
              {formatBytes(stats.deletableBytes)} قابل‌حذف
            </p>
            <p className="text-xs text-amber-200">
              نسخه‌ها و مود‌های ۳۰+ روز استفاده‌نشده و فاقد ارتباط با سرور فعال.
            </p>
          </div>
        </div>
      )}

      {/* Entries list */}
      <div className="space-y-2">
        <div className="text-sm text-slate-400 font-medium">نسخه‌های بازی</div>
        {stats.entries
          .filter((e) => e.type === 'version')
          .map((entry) => (
            <EntryRow
              key={entry.key}
              entry={entry}
              deleting={deleting[entry.key] ?? false}
              onDelete={() => useCacheStore.getState().delete('version', entry.key)}
            />
          ))}
      </div>

      <div className="space-y-2">
        <div className="text-sm text-slate-400 font-medium">مود‌ها</div>
        {stats.entries.length === 0 ? (
          <div className="text-xs text-slate-500 p-4 bg-slate-800 rounded-lg">
            هیچ مود پیدا نشد.
          </div>
        ) : (
          stats.entries
            .filter((e) => e.type === 'mod')
            .map((entry) => (
              <EntryRow
                key={entry.key}
                entry={entry}
                deleting={deleting[entry.key] ?? false}
                onDelete={() => useCacheStore.getState().delete('mod', entry.key)}
              />
            ))
        )}
      </div>

      <button
        onClick={load}
        className="flex items-center gap-2 text-xs text-slate-500 hover:text-slate-300 transition-colors"
      >
        <RefreshCw size={12} />
        بروزرسانی
      </button>
    </div>
  );
}

function StatCard({ label, bytes }: { label: string; bytes: number }) {
  return (
    <div className="bg-slate-800 border border-slate-700 rounded-lg p-4 text-center">
      <div className="text-2xl font-bold text-slate-100">{formatBytes(bytes)}</div>
      <div className="text-xs text-slate-500 mt-1">{label}</div>
    </div>
  );
}

interface EntryRowProps {
  entry: CacheEntry;
  deleting: boolean;
  onDelete: () => void;
}

function EntryRow({ entry, deleting, onDelete }: EntryRowProps) {
  const usedByCount = entry.usedByProfiles.length;
  const canDelete = usedByCount === 0;
  const lastUsedDaysAgo = daysSince(entry.lastUsedAt);
  const shouldSuggestDelete = canDelete && lastUsedDaysAgo > 30;

  return (
    <div
      className={`flex items-center justify-between p-3 rounded-lg border transition-all ${
        shouldSuggestDelete
          ? 'bg-red-900/10 border-red-700/30'
          : 'bg-slate-800 border-slate-700'
      }`}
    >
      <div className="space-y-0.5">
        <div className="flex items-center gap-2">
          <span className="font-mono text-sm text-slate-200">
            {entry.type === 'version'
              ? `v${entry.gameVersion} (${entry.platform})`
              : entry.modId}
          </span>
          {usedByCount > 0 && (
            <span className="text-xs text-emerald-400 bg-emerald-900/30 px-2 py-0.5 rounded">
              ✓ استفاده: {usedByCount}
            </span>
          )}
        </div>
        <div className="flex items-center gap-3 text-xs text-slate-500">
          <span>{formatBytes(entry.sizeBytes)}</span>
          {lastUsedDaysAgo > 0 && (
            <span>
              آخرین استفاده: {lastUsedDaysAgo} روز پیش
            </span>
          )}
        </div>
      </div>

      <button
        onClick={onDelete}
        disabled={deleting || !canDelete}
        title={
          !canDelete
            ? `در حال استفاده توسط: ${entry.usedByProfiles.join(', ')}`
            : `حذف ${formatBytes(entry.sizeBytes)}`
        }
        className={`p-2 rounded transition-all ${
          deleting
            ? 'bg-slate-700 text-slate-500 cursor-not-allowed'
            : canDelete
              ? 'bg-red-600 hover:bg-red-500 text-white'
              : 'bg-slate-700 text-slate-500 cursor-not-allowed'
        }`}
      >
        {deleting ? (
          <RefreshCw size={14} className="animate-spin" />
        ) : (
          <Trash2 size={14} />
        )}
      </button>
    </div>
  );
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '—';
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

function daysSince(dateStr: string): number {
  try {
    const date = new Date(dateStr);
    const now = new Date();
    const ms = now.getTime() - date.getTime();
    return Math.floor(ms / (1000 * 60 * 60 * 24));
  } catch {
    return 0;
  }
}
