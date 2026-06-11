import { SessionStatus } from '../types';
import { useSessionStore } from '../stores/session.store';
import { Download, CheckCircle, Loader2, AlertCircle, Play } from 'lucide-react';

interface DownloadPanelProps {
  sessions: SessionStatus[];
  onLaunch?: () => void;
}

export function DownloadPanel({ sessions, onLaunch }: DownloadPanelProps) {
  const launchState   = useSessionStore(s => s.launchState);
  const currentServer = useSessionStore(s => s.currentServer);
  const isComplete    = launchState === 'complete';
  const activeSessions = sessions.filter(
    s => s.state === 'downloading' || s.state === 'resolving' || s.state === 'installing'
  );
  const completedSessions = sessions.filter(s => s.state === 'complete');
  const failedSessions = sessions.filter(s => s.state === 'error');

  if (sessions.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-slate-400">
        <Download size={48} className="mb-4 opacity-50" />
        <p>No active downloads</p>
        <p className="text-sm mt-2">Join a server to start downloading mods</p>
      </div>
    );
  }

  // If session completed instantly (0 mods), show a dedicated ready card
  if (isComplete && activeSessions.length === 0 && completedSessions.length > 0 && currentServer) {
    return (
      <div className="space-y-4">
        <div className="bg-emerald-900/20 border border-emerald-500/40 rounded-xl p-6 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <CheckCircle size={24} className="text-emerald-400 shrink-0" />
            <div>
              <p className="font-semibold text-emerald-300">{currentServer.name}</p>
              <p className="text-sm text-slate-400">No mods required — server is ready to join</p>
            </div>
          </div>
          {onLaunch && (
            <button
              onClick={onLaunch}
              className="flex items-center gap-2 px-5 py-2.5 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg text-sm font-medium transition-colors"
            >
              <Play size={16} /> Launch
            </button>
          )}
        </div>
        <CompletedSessionsList sessions={completedSessions} />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Active Downloads */}
      {activeSessions.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-slate-300 mb-3 uppercase tracking-wide">
            Active ({activeSessions.length})
          </h3>
          <div className="space-y-3">
            {activeSessions.map(session => (
              <DownloadCard key={session.sessionId} session={session} />
            ))}
          </div>
        </div>
      )}

      {/* Failed */}
      {failedSessions.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-red-400 mb-3 uppercase tracking-wide">
            Failed ({failedSessions.length})
          </h3>
          <div className="space-y-2">
            {failedSessions.map(session => (
              <ErrorCard key={session.sessionId} session={session} />
            ))}
          </div>
        </div>
      )}

      {/* Completed */}
      {completedSessions.length > 0 && (
        <CompletedSessionsList sessions={completedSessions} />
      )}
    </div>
  );
}

function DownloadCard({ session }: { session: SessionStatus }) {
  const formatSpeed = (speed?: number) => {
    if (!speed) return '';
    if (speed > 1024 * 1024) return `${(speed / 1024 / 1024).toFixed(1)} MB/s`;
    if (speed > 1024) return `${(speed / 1024).toFixed(1)} KB/s`;
    return `${speed} B/s`;
  };

  const formatETA = (eta?: number) => {
    if (!eta) return '';
    if (eta > 60) return `${Math.ceil(eta / 60)}m`;
    return `${eta}s`;
  };

  return (
    <div className="bg-slate-800 border border-slate-700 rounded-lg p-4">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          {session.state === 'downloading' ? (
            <Loader2 size={18} className="text-blue-400 animate-spin" />
          ) : (
            <div className="w-4 h-4 rounded-full border-2 border-slate-600 border-t-blue-400 animate-spin" />
          )}
          <span className="font-medium text-slate-200">
            {session.currentMod || 'Resolving mods...'}
          </span>
        </div>
        <span className="text-sm text-slate-400">{session.progress}%</span>
      </div>

      {/* Progress Bar */}
      <div className="w-full bg-slate-700 rounded-full h-2 mb-3">
        <div
          className="bg-blue-500 h-2 rounded-full transition-all duration-300"
          style={{ width: `${session.progress}%` }}
        />
      </div>

      {/* Stats */}
      <div className="flex items-center justify-between text-xs text-slate-400">
        <span>{formatSpeed(session.downloadSpeed)}</span>
        {session.eta !== undefined && session.eta > 0 && (
          <span>ETA: {formatETA(session.eta)}</span>
        )}
      </div>
    </div>
  );
}

function CompletedSessionsList({ sessions }: { sessions: SessionStatus[] }) {
  return (
    <div>
      <h3 className="text-sm font-semibold text-slate-300 mb-3 uppercase tracking-wide">
        Completed ({sessions.length})
      </h3>
      <div className="space-y-2">
        {sessions.map(session => (
          <CompletedCard key={session.sessionId} session={session} />
        ))}
      </div>
    </div>
  );
}

function CompletedCard({ session }: { session: SessionStatus }) {
  const currentServer = useSessionStore(s => s.currentServer);
  const label = currentServer?.name ?? `Session ${session.sessionId.slice(-8)}`;
  return (
    <div className="flex items-center gap-3 p-3 bg-slate-800/50 rounded-lg">
      <CheckCircle size={16} className="text-emerald-400 shrink-0" />
      <span className="text-sm text-slate-300 flex-1">{label}</span>
      <span className="text-xs text-emerald-400">Ready</span>
    </div>
  );
}

function ErrorCard({ session }: { session: SessionStatus }) {
  const lastError = session.errors?.[session.errors.length - 1] ?? 'Unknown error';
  return (
    <div className="bg-red-900/20 border border-red-500/30 rounded-lg p-4">
      <div className="flex items-center gap-2 mb-2">
        <AlertCircle size={16} className="text-red-400 shrink-0" />
        <span className="text-sm font-medium text-red-300">
          Session {session.sessionId.slice(-8)} failed
        </span>
      </div>
      <p className="text-xs text-red-400 font-mono break-all">{lastError}</p>
    </div>
  );
}
