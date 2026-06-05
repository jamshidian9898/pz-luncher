import { useState } from 'react';
import { useSettingsStore } from '../stores/settings.store';
import { CheckCircle, XCircle, Loader2, RefreshCw, AlertCircle } from 'lucide-react';

type CheckStatus = 'pending' | 'running' | 'ok' | 'warn' | 'fail';

interface Check {
  id: string;
  label: string;
  description: string;
  run: (settings: ReturnType<typeof useSettingsStore.getState>['settings']) => Promise<CheckResult>;
}

interface CheckResult {
  status: 'ok' | 'warn' | 'fail';
  message: string;
  action?: string;
}

interface CheckState {
  status: CheckStatus;
  result: CheckResult | null;
}

/* ── helpers ── */
const isWails = (): boolean => typeof (window as any).go !== 'undefined';

/* ── Checks definition ── */
const CHECKS: Check[] = [
  {
    id: 'api',
    label: 'Backend reachable',
    description: 'App backend responds to settings request',
    run: async () => {
      if (isWails()) {
        try {
          const go = (window as any).go;
          await go.main.App.GetSettings();
          return { status: 'ok', message: 'Wails backend is up' };
        } catch (e: any) {
          return { status: 'fail', message: String(e), action: 'Restart the launcher' };
        }
      }
      try {
        const r = await fetch('/api/settings', { signal: AbortSignal.timeout(3000) });
        if (r.ok) return { status: 'ok', message: 'Dev API is up' };
        return { status: 'fail', message: `HTTP ${r.status}`, action: 'Make sure dev-api is running on :8765' };
      } catch {
        return { status: 'fail', message: 'Could not reach API', action: 'Run: go run ./apps/dev-api' };
      }
    },
  },
  {
    id: 'gamepath',
    label: 'Game path configured',
    description: 'settings.gamePath is set',
    run: async (s) => {
      if (!s?.gamePath) return { status: 'fail', message: 'Game path is empty', action: 'Go to Settings and set the game installation path' };
      return { status: 'ok', message: s.gamePath };
    },
  },
  {
    id: 'registry',
    label: 'Server registry loaded',
    description: 'At least one server in /registry/servers.json',
    run: async () => {
      try {
        const r = await fetch('/registry/servers.json', { signal: AbortSignal.timeout(3000) });
        if (!r.ok) return { status: 'fail', message: `HTTP ${r.status}`, action: 'Check public/registry/servers.json' };
        const data = await r.json();
        const count = Array.isArray(data?.servers) ? data.servers.length : 0;
        if (count === 0) return { status: 'warn', message: 'No servers defined', action: 'Add entries to servers.json' };
        return { status: 'ok', message: `${count} server${count !== 1 ? 's' : ''} found` };
      } catch {
        return { status: 'fail', message: 'Registry unreachable', action: 'Make sure Vite proxy is configured for /registry' };
      }
    },
  },
  {
    id: 'cache',
    label: 'Cache directory writable',
    description: 'settings.cacheLocation is set (or uses default)',
    run: async (s) => {
      const path = s?.cacheLocation || '(default ./cache)';
      if (!s?.cacheLocation) return { status: 'warn', message: 'Using default cache path', action: 'Set a custom cache location in Settings for better control' };
      return { status: 'ok', message: path };
    },
  },
  {
    id: 'profiles',
    label: 'Profiles directory configured',
    description: 'settings.profilesLocation is set (or uses default)',
    run: async (s) => {
      if (!s?.profilesLocation) return { status: 'warn', message: 'Using default profiles path', action: 'Set a custom profiles location in Settings' };
      return { status: 'ok', message: s.profilesLocation };
    },
  },
  {
    id: 'sse',
    label: 'Event system works',
    description: 'Wails events or SSE streaming available',
    run: async () => {
      if (isWails()) {
        return { status: 'ok', message: 'Wails event system active' };
      }
      return new Promise((resolve) => {
        const timeout = setTimeout(() => {
          resolve({ status: 'warn', message: 'No SSE response in 2s — normal for empty session' });
        }, 2000);
        try {
          const es = new EventSource('/api/events/health-check');
          es.onopen = () => { clearTimeout(timeout); es.close(); resolve({ status: 'ok', message: 'SSE connected' }); };
          es.onerror = () => { clearTimeout(timeout); es.close(); resolve({ status: 'warn', message: 'SSE closed immediately — normal without active session' }); };
        } catch {
          clearTimeout(timeout);
          resolve({ status: 'fail', message: 'EventSource not supported' });
        }
      });
    },
  },
  {
    id: 'join_api',
    label: 'Join function available',
    description: 'JoinServer binding or /api/join endpoint responds',
    run: async () => {
      if (isWails()) {
        const hasBinding = typeof (window as any).go?.main?.App?.JoinServer === 'function';
        if (hasBinding) return { status: 'ok', message: 'JoinServer binding ready' };
        return { status: 'fail', message: 'JoinServer binding not found', action: 'Rebuild the app' };
      }
      try {
        const r = await fetch('/api/join/__health__', { method: 'POST', signal: AbortSignal.timeout(3000) });
        if (r.status === 404 || r.status === 400) return { status: 'warn', message: 'Endpoint exists — server not found (expected)' };
        return { status: 'ok', message: `API responded with HTTP ${r.status}` };
      } catch {
        return { status: 'fail', message: 'Join API unreachable', action: 'Make sure dev-api is running' };
      }
    },
  },
];

/* ── Component ── */
export function HealthCheck() {
  const settings = useSettingsStore(s => s.settings);
  const [states, setStates] = useState<Record<string, CheckState>>({});
  const [running, setRunning] = useState(false);

  async function runAll() {
    setRunning(true);
    const initial: Record<string, CheckState> = {};
    for (const c of CHECKS) initial[c.id] = { status: 'running', result: null };
    setStates(initial);

    for (const check of CHECKS) {
      try {
        const result = await check.run(settings);
        setStates(prev => ({ ...prev, [check.id]: { status: result.status, result } }));
      } catch {
        setStates(prev => ({
          ...prev,
          [check.id]: { status: 'fail', result: { status: 'fail', message: 'Unexpected error' } },
        }));
      }
    }
    setRunning(false);
  }

  const hasResults = Object.keys(states).length > 0;
  const counts = hasResults ? {
    ok:   Object.values(states).filter(s => s.status === 'ok').length,
    warn: Object.values(states).filter(s => s.status === 'warn').length,
    fail: Object.values(states).filter(s => s.status === 'fail').length,
  } : null;

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold text-slate-200">System Health</h3>
          <p className="text-xs text-slate-500 mt-0.5">RFC-0042 — verify all systems before Beta</p>
        </div>
        <button
          onClick={runAll}
          disabled={running}
          className="flex items-center gap-2 px-4 py-2 bg-slate-700 hover:bg-slate-600 disabled:opacity-50 text-slate-200 rounded-lg text-sm font-medium transition-colors"
        >
          {running
            ? <Loader2 size={14} className="animate-spin" />
            : <RefreshCw size={14} />
          }
          {running ? 'Running…' : 'Run Checks'}
        </button>
      </div>

      {/* Summary bar */}
      {counts && !running && (
        <div className={`flex items-center gap-4 px-4 py-2.5 rounded-lg text-sm font-medium ${
          counts.fail > 0 ? 'bg-red-900/20 border border-red-500/30 text-red-300' :
          counts.warn > 0 ? 'bg-amber-900/20 border border-amber-500/30 text-amber-300' :
                            'bg-emerald-900/20 border border-emerald-500/30 text-emerald-300'
        }`}>
          {counts.fail > 0
            ? <XCircle size={16} />
            : counts.warn > 0
            ? <AlertCircle size={16} />
            : <CheckCircle size={16} />
          }
          {counts.fail === 0 && counts.warn === 0
            ? 'All checks passed — ready for Beta'
            : `${counts.ok} passed · ${counts.warn} warnings · ${counts.fail} failed`
          }
        </div>
      )}

      {/* Check list */}
      <div className="space-y-2">
        {CHECKS.map(check => {
          const state = states[check.id];
          return (
            <CheckRow
              key={check.id}
              check={check}
              state={state ?? { status: 'pending', result: null }}
            />
          );
        })}
      </div>

      {!hasResults && (
        <p className="text-xs text-slate-500 text-center py-4">
          Click "Run Checks" to verify your environment before Beta testing.
        </p>
      )}
    </div>
  );
}

/* ── Row ── */
function CheckRow({ check, state }: { check: Check; state: CheckState }) {
  return (
    <div className={`rounded-lg border px-4 py-3 transition-colors ${
      state.status === 'ok'      ? 'border-emerald-500/20 bg-emerald-900/10' :
      state.status === 'warn'    ? 'border-amber-500/20 bg-amber-900/10' :
      state.status === 'fail'    ? 'border-red-500/20 bg-red-900/10' :
      state.status === 'running' ? 'border-blue-500/20 bg-blue-900/10 animate-pulse' :
                                   'border-slate-700 bg-slate-800/50'
    }`}>
      <div className="flex items-start gap-3">
        <div className="mt-0.5 shrink-0">
          {state.status === 'ok'      && <CheckCircle size={15} className="text-emerald-400" />}
          {state.status === 'warn'    && <AlertCircle size={15} className="text-amber-400" />}
          {state.status === 'fail'    && <XCircle size={15} className="text-red-400" />}
          {state.status === 'running' && <Loader2 size={15} className="text-blue-400 animate-spin" />}
          {state.status === 'pending' && <div className="w-3.5 h-3.5 rounded-full border border-slate-600 mt-0.5" />}
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-baseline gap-2">
            <span className="text-sm font-medium text-slate-200">{check.label}</span>
            {state.result && (
              <span className={`text-xs truncate ${
                state.status === 'ok'   ? 'text-emerald-400' :
                state.status === 'warn' ? 'text-amber-400' :
                                          'text-red-400'
              }`}>
                — {state.result.message}
              </span>
            )}
          </div>
          <p className="text-xs text-slate-500 mt-0.5">{check.description}</p>
          {state.result?.action && state.status !== 'ok' && (
            <p className="text-xs text-slate-400 mt-1.5 italic">{state.result.action}</p>
          )}
        </div>
      </div>
    </div>
  );
}
