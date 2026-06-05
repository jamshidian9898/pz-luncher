import { useSessionStore } from '../stores/session.store';
import { useDownloadsStore } from '../stores/downloads.store';
import { CheckCircle, Loader2, AlertCircle, Play, RefreshCw, Wrench } from 'lucide-react';

interface Step {
  id: string;
  label: string;
  states: string[];
}

const STEPS: Step[] = [
  { id: 'resolving',   label: 'Resolving mods',    states: ['resolving'] },
  { id: 'downloading', label: 'Downloading',        states: ['downloading'] },
  { id: 'installing',  label: 'Installing',         states: ['installing', 'verifying', 'materializing'] },
  { id: 'ready',       label: 'Ready to launch',    states: ['complete'] },
];

function stepStatus(step: Step, currentState: string): 'done' | 'active' | 'pending' | 'error' {
  if (currentState === 'error') return 'pending';
  const currentIdx = STEPS.findIndex(s => s.states.includes(currentState));
  const stepIdx    = STEPS.indexOf(step);
  if (stepIdx < currentIdx) return 'done';
  if (step.states.includes(currentState)) return 'active';
  return 'pending';
}

interface SessionProgressCardProps {
  onLaunch?: () => void;
  onRetry?: () => void;
  onRepairCache?: () => void;
}

export function SessionProgressCard({ onLaunch, onRetry, onRepairCache }: SessionProgressCardProps) {
  const launchState   = useSessionStore(s => s.launchState);
  const currentServer = useSessionStore(s => s.currentServer);
  const sessionId     = useSessionStore(s => s.currentSessionId);

  const session = useDownloadsStore(s =>
    sessionId ? s.sessions.get(sessionId) : undefined
  );

  if (launchState === 'idle' || !currentServer) return null;

  const progress    = session?.progress ?? 0;
  const currentMod  = session?.currentMod;
  const speed       = session?.downloadSpeed;
  const eta         = session?.eta;
  const errors      = session?.errors ?? [];
  const hasError    = launchState === 'error';
  const isComplete  = launchState === 'complete';

  return (
    <div className={`rounded-xl border p-5 space-y-4 transition-colors ${
      hasError
        ? 'bg-red-900/20 border-red-500/40'
        : isComplete
        ? 'bg-emerald-900/20 border-emerald-500/40'
        : 'bg-slate-800 border-slate-700'
    }`}>
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <p className="text-xs text-slate-400 uppercase tracking-wide mb-0.5">Joining</p>
          <h3 className="text-base font-semibold text-slate-100">{currentServer.name}</h3>
        </div>
        {isComplete && onLaunch && (
          <button
            onClick={onLaunch}
            className="flex items-center gap-2 px-4 py-2 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg text-sm font-medium transition-colors"
          >
            <Play size={16} />
            Launch
          </button>
        )}
        {hasError && (
          <AlertCircle size={20} className="text-red-400 shrink-0" />
        )}
      </div>

      {/* Step Tracker */}
      <div className="flex items-center gap-1">
        {STEPS.map((step, i) => {
          const status = stepStatus(step, launchState);
          return (
            <div key={step.id} className="flex items-center flex-1 min-w-0">
              <div className={`flex items-center gap-1.5 px-2 py-1 rounded-md text-xs font-medium transition-colors flex-1 min-w-0 ${
                status === 'active'  ? 'bg-blue-500/20 text-blue-300' :
                status === 'done'    ? 'bg-emerald-500/10 text-emerald-400' :
                status === 'error'   ? 'bg-red-500/10 text-red-400' :
                                       'text-slate-600'
              }`}>
                {status === 'done'   && <CheckCircle size={12} className="shrink-0" />}
                {status === 'active' && <Loader2 size={12} className="animate-spin shrink-0" />}
                <span className="truncate">{step.label}</span>
              </div>
              {i < STEPS.length - 1 && (
                <div className={`h-px w-3 shrink-0 mx-0.5 ${
                  status === 'done' ? 'bg-emerald-500/40' : 'bg-slate-700'
                }`} />
              )}
            </div>
          );
        })}
      </div>

      {/* Progress Bar (only while downloading) */}
      {launchState === 'downloading' && (
        <div className="space-y-1.5">
          <div className="flex justify-between text-xs text-slate-400">
            <span className="truncate max-w-xs">
              {currentMod ? `Downloading: ${currentMod}` : 'Preparing…'}
            </span>
            <span className="shrink-0 ml-2">{Math.round(progress)}%</span>
          </div>
          <div className="w-full bg-slate-700 rounded-full h-2 overflow-hidden">
            <div
              className="h-2 rounded-full bg-blue-500 transition-all duration-300"
              style={{ width: `${Math.min(progress, 100)}%` }}
            />
          </div>
          <div className="flex justify-between text-xs text-slate-500">
            <span>{speed ? formatSpeed(speed) : ''}</span>
            <span>{eta && eta > 0 ? `ETA ${formatETA(eta)}` : ''}</span>
          </div>
        </div>
      )}

      {/* Resolving / Installing indicator */}
      {(launchState === 'resolving' || launchState === 'installing') && (
        <div className="flex items-center gap-2 text-sm text-slate-400">
          <Loader2 size={14} className="animate-spin" />
          <span>{launchState === 'resolving' ? 'Resolving mod dependencies…' : 'Installing mods into profile…'}</span>
        </div>
      )}

      {/* Ready state */}
      {isComplete && (
        <div className="flex items-center gap-2 text-sm text-emerald-400">
          <CheckCircle size={14} />
          <span>All mods ready — click Launch to start the game</span>
        </div>
      )}

      {/* Error state */}
      {hasError && (
        <div className="space-y-3">
          {errors.length > 0 && (
            <div className="space-y-1">
              {errors.map((e, i) => (
                <p key={i} className="text-xs text-red-400 break-all">{mapErrorCode(e)}</p>
              ))}
            </div>
          )}
          <div className="flex gap-2">
            {onRetry && (
              <button
                onClick={onRetry}
                className="flex items-center gap-1.5 px-3 py-1.5 bg-slate-700 hover:bg-slate-600 text-slate-200 rounded-lg text-xs font-medium transition-colors"
              >
                <RefreshCw size={12} /> Retry
              </button>
            )}
            {onRepairCache && (
              <button
                onClick={onRepairCache}
                className="flex items-center gap-1.5 px-3 py-1.5 bg-slate-700 hover:bg-slate-600 text-slate-200 rounded-lg text-xs font-medium transition-colors"
              >
                <Wrench size={12} /> Repair Cache
              </button>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

function formatSpeed(bps: number): string {
  if (bps >= 1024 * 1024) return `${(bps / 1024 / 1024).toFixed(1)} MB/s`;
  if (bps >= 1024)         return `${(bps / 1024).toFixed(1)} KB/s`;
  return `${bps} B/s`;
}

function formatETA(secs: number): string {
  if (secs >= 60) return `${Math.ceil(secs / 60)}m`;
  return `${secs}s`;
}

/** RFC-0038 minimal error mapper — translates pipeline codes to human messages */
function mapErrorCode(raw: string): string {
  if (raw.includes('PIPELINE_MANIFEST'))  return 'Could not load server manifest. The server may be offline or misconfigured.';
  if (raw.includes('PIPELINE_RESOLVER'))  return 'Mod dependency could not be resolved. A required mod may be missing from the manifest.';
  if (raw.includes('PIPELINE_PLAN'))      return 'Could not build a download plan. Check your internet connection.';
  if (raw.includes('PIPELINE_PROFILE'))   return 'Failed to create the game profile. Check your Profiles directory in Settings.';
  if (raw.includes('PIPELINE_DOWNLOAD'))  return 'Download failed. Check your internet connection or cache settings.';
  if (raw.includes('LAUNCH_PROFILE_NOT_READY')) return 'Game profile is not ready. Please join the server first.';
  if (raw.includes('join required first')) return 'You must join a server before launching.';
  return raw;
}
