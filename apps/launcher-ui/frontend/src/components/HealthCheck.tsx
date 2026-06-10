import { useState } from 'react';
import { useSettingsStore } from '../stores/settings.store';
import { CheckCircle, XCircle, Loader2, RefreshCw, AlertCircle } from 'lucide-react';

type CheckStatus = 'pending' | 'running' | 'ok' | 'warn' | 'fail';

interface CheckResult {
  status: 'ok' | 'warn' | 'fail';
  message: string;
  action?: string;
}

interface CheckState {
  status: CheckStatus;
  result: CheckResult | null;
}

const isWails = (): boolean =>
  typeof (window as any).go !== 'undefined' &&
  typeof (window as any).go?.main?.App?.CheckBackend === 'function';

function getBackendUrl(s: any): string {
  if (s?.backendUrl) return s.backendUrl.replace(/\/$/, '');
  if (typeof (window as any).__BACKEND_URL__ === 'string') return (window as any).__BACKEND_URL__;
  return 'http://localhost:8080';
}

const CHECK_IDS = ['backend', 'servers', 'agents', 'gamepath', 'join_api', 'wails'] as const;
type CheckId = typeof CHECK_IDS[number];

const CHECK_META: Record<CheckId, { label: string; description: string }> = {
  backend:  { label: 'Backend reachable',    description: 'PZ Backend API responds to /api/v1/servers' },
  servers:  { label: 'Servers available',    description: 'At least one server registered (via agent)' },
  agents:   { label: 'Agent connected',      description: 'At least one agent is online and sending heartbeats' },
  gamepath: { label: 'Game path configured', description: 'PZ installation path is set in settings' },
  join_api: { label: 'Join API works',       description: 'POST /join responds correctly' },
  wails:    { label: 'Launcher runtime',     description: 'Wails bindings available' },
};

async function runChecksViaGo(): Promise<Record<string, CheckState>> {
  const h = await (window as any).go.main.App.CheckBackend();
  const states: Record<string, CheckState> = {};

  const toState = (s: string, msg: string, action?: string): CheckState => ({
    status: s as CheckStatus,
    result: { status: s as 'ok' | 'warn' | 'fail', message: msg, action },
  });

  states.backend  = toState(h.backend,  h.backendMsg,  h.backend  === 'fail' ? `Check Docker stack on Windows` : undefined);
  states.servers  = toState(h.servers,  h.serversMsg,  h.servers  === 'warn' ? 'Run: make fake-agents-up' : undefined);
  states.agents   = toState(h.agents,   h.agentsMsg,   h.agents   === 'warn' ? 'Run: make fake-agents-up' : undefined);
  states.gamepath = typeof (window as any).go?.main?.App?.GetSettings === 'function'
    ? await (async () => {
        try {
          const s = await (window as any).go.main.App.GetSettings();
          if (!s?.gamePath) return toState('fail', 'Game path is empty', 'Go to Settings → set game path');
          return toState('ok', s.gamePath);
        } catch { return toState('warn', 'Could not read settings'); }
      })()
    : toState('warn', 'Cannot check game path');
  states.join_api = h.backend === 'ok'
    ? toState('ok', 'Join endpoint active (backend reachable)')
    : toState('fail', 'Backend unreachable');
  states.wails = toState('ok', `Wails runtime active — backend: ${h.backendUrl}`);

  return states;
}

async function runChecksViaFetch(settings: any): Promise<Record<string, CheckState>> {
  const base = getBackendUrl(settings);
  const states: Record<string, CheckState> = {};

  const ok   = (msg: string): CheckState => ({ status: 'ok',   result: { status: 'ok',   message: msg } });
  const warn = (msg: string, action?: string): CheckState => ({ status: 'warn', result: { status: 'warn', message: msg, action } });
  const fail = (msg: string, action?: string): CheckState => ({ status: 'fail', result: { status: 'fail', message: msg, action } });

  try {
    const r = await fetch(`${base}/api/v1/servers`, { signal: AbortSignal.timeout(5000) });
    if (!r.ok) {
      states.backend = fail(`HTTP ${r.status}`, `Check backend at ${base}`);
      states.servers = fail('Backend error'); states.agents = fail('Backend error'); states.join_api = fail('Backend error');
    } else {
      const data = await r.json();
      const count = Array.isArray(data?.servers) ? data.servers.length : 0;
      states.backend = ok(`Connected — ${count} server(s)`);
      states.servers = count > 0 ? ok(`${count} server(s) online`) : warn('No servers yet', 'Run: make fake-agents-up');
    }
  } catch (e: any) {
    const msg = e?.message || 'Connection failed';
    states.backend = fail(msg, `Make sure backend is running at ${base}`);
    states.servers = fail('Backend unreachable'); states.agents = fail('Backend unreachable'); states.join_api = fail('Backend unreachable');
  }

  if (!states.agents) {
    try {
      const r = await fetch(`${base}/api/v1/agents`, { signal: AbortSignal.timeout(5000) });
      const data = await r.json();
      const agents = Array.isArray(data?.agents) ? data.agents : [];
      const online = agents.filter((a: any) => a.status === 'online').length;
      states.agents = online > 0 ? ok(`${online} agent(s) online`) : warn(`${agents.length} agent(s), none online`, 'Run: make fake-agents-up');
    } catch { states.agents = fail('Cannot reach agents endpoint'); }
  }

  if (!states.join_api) {
    try {
      const r = await fetch(`${base}/api/v1/join/__health_check__`, { method: 'POST', signal: AbortSignal.timeout(5000) });
      states.join_api = r.status === 404 || r.status === 409
        ? ok('Join endpoint active')
        : ok(`Join API responded: HTTP ${r.status}`);
    } catch { states.join_api = fail('Join API unreachable', `Check backend at ${base}`); }
  }

  states.gamepath = settings?.gamePath ? ok(settings.gamePath) : fail('Game path empty', 'Go to Settings → set game path');
  states.wails = warn('Running in browser (dev mode)', 'This is fine for development');

  return states;
}

/* ── Component ── */
export function HealthCheck() {
  const { settings, fetchSettings } = useSettingsStore(s => ({ settings: s.settings, fetchSettings: s.fetchSettings }));
  const [states, setStates] = useState<Record<string, CheckState>>({});
  const [running, setRunning] = useState(false);

  const resolvedBackend = getBackendUrl(settings);

  async function runAll() {
    if (!settings) await fetchSettings();
    const currentSettings = useSettingsStore.getState().settings;

    setRunning(true);
    const initial: Record<string, CheckState> = {};
    for (const id of CHECK_IDS) initial[id] = { status: 'running', result: null };
    setStates(initial);

    try {
      const results = isWails()
        ? await runChecksViaGo()
        : await runChecksViaFetch(currentSettings);
      setStates(results);
    } catch (e: any) {
      const errState: Record<string, CheckState> = {};
      for (const id of CHECK_IDS) errState[id] = { status: 'fail', result: { status: 'fail', message: e?.message || 'Unexpected error' } };
      setStates(errState);
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
          <p className="text-xs text-slate-500 mt-0.5">
            Target: <span className="text-emerald-400 font-mono">{resolvedBackend}</span>
          </p>
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
        {CHECK_IDS.map(id => (
          <CheckRow
            key={id}
            id={id}
            meta={CHECK_META[id]}
            state={states[id] ?? { status: 'pending', result: null }}
          />
        ))}
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
function CheckRow({ id: _id, meta, state }: { id: string; meta: { label: string; description: string }; state: CheckState }) {
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
            <span className="text-sm font-medium text-slate-200">{meta.label}</span>
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
          <p className="text-xs text-slate-500 mt-0.5">{meta.description}</p>
          {state.result?.action && state.status !== 'ok' && (
            <p className="text-xs text-slate-400 mt-1.5 italic">{state.result.action}</p>
          )}
        </div>
      </div>
    </div>
  );
}
