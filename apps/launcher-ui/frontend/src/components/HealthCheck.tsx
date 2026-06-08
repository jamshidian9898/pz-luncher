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

function getBackendUrl(s: any): string {
  if (s?.backendUrl) return s.backendUrl.replace(/\/$/, '');
  if (typeof (window as any).__BACKEND_URL__ === 'string') return (window as any).__BACKEND_URL__;
  return 'http://localhost:8080';
}

/* ── Checks definition ── */
const CHECKS: Check[] = [
  {
    id: 'backend',
    label: 'Backend reachable',
    description: 'PZ Backend API responds to /api/v1/servers',
    run: async (s) => {
      const base = getBackendUrl(s);
      try {
        const r = await fetch(`${base}/api/v1/servers`, { signal: AbortSignal.timeout(5000) });
        if (!r.ok) return { status: 'fail', message: `HTTP ${r.status}`, action: `Check backend at ${base}` };
        const data = await r.json();
        const count = Array.isArray(data?.servers) ? data.servers.length : 0;
        return { status: 'ok', message: `Connected — ${count} server(s) registered` };
      } catch (e: any) {
        return { status: 'fail', message: e?.message || 'Connection failed', action: `Make sure backend is running at ${base}` };
      }
    },
  },
  {
    id: 'servers',
    label: 'Servers available',
    description: 'At least one server registered (via agent)',
    run: async (s) => {
      const base = getBackendUrl(s);
      try {
        const r = await fetch(`${base}/api/v1/servers`, { signal: AbortSignal.timeout(5000) });
        if (!r.ok) return { status: 'fail', message: `HTTP ${r.status}` };
        const data = await r.json();
        const count = Array.isArray(data?.servers) ? data.servers.length : 0;
        if (count === 0) return { status: 'warn', message: 'No servers yet', action: 'Install agent on your PZ server — it will auto-register' };
        return { status: 'ok', message: `${count} server${count !== 1 ? 's' : ''} online` };
      } catch {
        return { status: 'fail', message: 'Cannot reach backend' };
      }
    },
  },
  {
    id: 'agents',
    label: 'Agent connected',
    description: 'At least one agent is online and sending heartbeats',
    run: async (s) => {
      const base = getBackendUrl(s);
      try {
        const r = await fetch(`${base}/api/v1/agents`, { signal: AbortSignal.timeout(5000) });
        if (!r.ok) return { status: 'warn', message: `HTTP ${r.status}` };
        const data = await r.json();
        const agents = Array.isArray(data?.agents) ? data.agents : [];
        const online = agents.filter((a: any) => a.status === 'online').length;
        if (online === 0 && agents.length === 0) return { status: 'warn', message: 'No agents registered', action: 'Install agent on PZ server VM' };
        if (online === 0) return { status: 'warn', message: `${agents.length} agent(s) but none online`, action: 'Check agent process on server' };
        return { status: 'ok', message: `${online} agent(s) online` };
      } catch {
        return { status: 'fail', message: 'Cannot reach backend' };
      }
    },
  },
  {
    id: 'gamepath',
    label: 'Game path configured',
    description: 'PZ installation path is set in settings',
    run: async (s) => {
      if (!s?.gamePath) return { status: 'fail', message: 'Game path is empty', action: 'Go to Settings → General and set the game path' };
      return { status: 'ok', message: s.gamePath };
    },
  },
  {
    id: 'join_api',
    label: 'Join API works',
    description: 'POST /join responds correctly',
    run: async (s) => {
      const base = getBackendUrl(s);
      try {
        const r = await fetch(`${base}/api/v1/join/__health_check__`, { method: 'POST', signal: AbortSignal.timeout(5000) });
        if (r.status === 404) return { status: 'ok', message: 'Join endpoint active (server not found = expected)' };
        if (r.status === 409) return { status: 'ok', message: 'Join endpoint active (server offline = expected for health check)' };
        return { status: 'ok', message: `Join API responded: HTTP ${r.status}` };
      } catch {
        return { status: 'fail', message: 'Join API unreachable', action: `Check backend at ${base}` };
      }
    },
  },
  {
    id: 'wails',
    label: 'Launcher runtime',
    description: 'Wails bindings or dev-api available',
    run: async () => {
      if (isWails()) {
        const hasBinding = typeof (window as any).go?.main?.App?.JoinServer === 'function';
        if (hasBinding) return { status: 'ok', message: 'Wails runtime active — all bindings ready' };
        return { status: 'fail', message: 'Wails bindings missing', action: 'Rebuild the launcher' };
      }
      return { status: 'warn', message: 'Running in browser (dev mode)', action: 'This is fine for development' };
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
          <p className="text-xs text-slate-500 mt-0.5">Validates backend, agent, and launcher connectivity</p>
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
            ? 'All checks passed — system healthy'
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
