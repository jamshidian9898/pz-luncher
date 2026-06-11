import { useState } from 'react';
import { useEventLog } from '../stores/eventLog.store';
import { usePatchFailureLog } from '../stores/patchFailureLog.store';
import { ClipboardCopy, CheckCircle, Trash2, ChevronDown, ChevronRight } from 'lucide-react';

export function EventLogPanel() {
  const { entries, clear, getStats } = useEventLog();
  const { failures, clear: clearFailures } = usePatchFailureLog();
  const [copied, setCopied] = useState(false);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [filter, setFilter] = useState<'all' | 'applied' | 'rejected'>('all');

  const stats = getStats();

  const visible = entries.filter(e =>
    filter === 'all' ? true : e.status === filter
  );

  function copyAll() {
    const payload = {
      stats,
      entries: entries.map(e => ({
        t: new Date(e.appliedAt).toISOString(),
        type: e.event.type,
        session: e.event.sessionId,
        status: e.status,
        errors: e.validationErrors.length ? e.validationErrors : undefined,
        payload: e.event.payload,
      })),
      failures: failures.map(f => ({
        t: new Date(f.timestamp).toISOString(),
        type: f.eventType,
        session: f.sessionId,
        reason: f.reason,
      })),
    };
    const text = JSON.stringify(payload, null, 2);
    navigator.clipboard.writeText(text).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2500);
    }).catch(() => {
      const w = window.open('', '_blank');
      if (w) {
        w.document.write(`<pre style="font:13px monospace;padding:16px">${text}</pre>`);
        w.document.title = 'PZ Launcher — Event Log';
      }
    });
  }

  function clearAll() {
    clear();
    clearFailures();
  }

  return (
    <div className="space-y-3">
      {/* Toolbar */}
      <div className="flex items-center justify-between gap-2 flex-wrap">
        <div className="flex items-center gap-1 text-xs">
          <StatBadge label="total" value={stats.totalEvents} color="slate" />
          <StatBadge label="ok" value={stats.applied} color="emerald" />
          <StatBadge label="rejected" value={stats.rejected} color="red" />
          {failures.length > 0 && <StatBadge label="failures" value={failures.length} color="amber" />}
        </div>
        <div className="flex items-center gap-2">
          <select
            value={filter}
            onChange={e => setFilter(e.target.value as any)}
            className="bg-slate-700 border border-slate-600 rounded px-2 py-1 text-xs text-slate-300"
          >
            <option value="all">All</option>
            <option value="applied">Applied</option>
            <option value="rejected">Rejected</option>
          </select>
          <button
            onClick={clearAll}
            className="flex items-center gap-1 px-2 py-1 bg-slate-700 hover:bg-slate-600 rounded text-xs text-slate-400"
          >
            <Trash2 size={12} /> Clear
          </button>
          <button
            onClick={copyAll}
            className={`flex items-center gap-1 px-3 py-1 rounded text-xs font-medium transition-all ${
              copied
                ? 'bg-emerald-600/20 border border-emerald-500/40 text-emerald-400'
                : 'bg-slate-700 hover:bg-slate-600 border border-slate-600 text-slate-300'
            }`}
          >
            {copied ? <><CheckCircle size={12} /> Copied!</> : <><ClipboardCopy size={12} /> Copy Log</>}
          </button>
        </div>
      </div>

      {/* Entries */}
      {visible.length === 0 ? (
        <div className="text-center py-8 text-slate-500 text-sm">
          {entries.length === 0
            ? 'No events yet — click Join on a server to start'
            : 'No events match filter'}
        </div>
      ) : (
        <div className="space-y-1 max-h-[480px] overflow-y-auto pr-1">
          {visible.map(entry => {
            const expanded = expandedId === entry.id;
            const isErr = entry.status === 'rejected';
            return (
              <div
                key={entry.id}
                className={`rounded border text-xs font-mono ${
                  isErr
                    ? 'border-red-500/30 bg-red-900/10'
                    : 'border-slate-700 bg-slate-800/60'
                }`}
              >
                <button
                  className="w-full flex items-center gap-2 px-3 py-2 text-left"
                  onClick={() => setExpandedId(expanded ? null : entry.id)}
                >
                  {expanded ? <ChevronDown size={11} className="shrink-0 text-slate-500" /> : <ChevronRight size={11} className="shrink-0 text-slate-500" />}
                  <span className="text-slate-500 shrink-0">
                    {new Date(entry.appliedAt).toISOString().slice(11, 23)}
                  </span>
                  <span className={`shrink-0 px-1 rounded ${isErr ? 'text-red-400' : 'text-emerald-400'}`}>
                    {isErr ? '✗' : '✓'}
                  </span>
                  <span className="text-slate-200 truncate">{entry.event.type}</span>
                  <span className="text-slate-500 truncate ml-auto">{entry.event.sessionId?.slice(-12)}</span>
                </button>
                {expanded && (
                  <div className="px-3 pb-3 space-y-1 border-t border-slate-700/50 pt-2">
                    {entry.validationErrors.length > 0 && (
                      <div className="text-red-400">
                        {entry.validationErrors.map((e, i) => <div key={i}>⚠ {e}</div>)}
                      </div>
                    )}
                    {entry.event.payload && (
                      <pre className="text-slate-400 text-[11px] whitespace-pre-wrap break-all">
                        {JSON.stringify(entry.event.payload, null, 2)}
                      </pre>
                    )}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

function StatBadge({ label, value, color }: { label: string; value: number; color: string }) {
  const colors: Record<string, string> = {
    slate: 'bg-slate-700 text-slate-300',
    emerald: 'bg-emerald-900/40 text-emerald-400',
    red: 'bg-red-900/40 text-red-400',
    amber: 'bg-amber-900/40 text-amber-400',
  };
  return (
    <span className={`px-2 py-0.5 rounded text-xs font-medium ${colors[color] ?? colors.slate}`}>
      {value} {label}
    </span>
  );
}
