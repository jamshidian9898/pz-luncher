import { SessionStatus } from '../types';
import { Download, CheckCircle, Loader2 } from 'lucide-react';

interface DownloadPanelProps {
  sessions: SessionStatus[];
}

export function DownloadPanel({ sessions }: DownloadPanelProps) {
  const activeSessions = sessions.filter(
    s => s.state === 'downloading' || s.state === 'resolving' || s.state === 'installing'
  );

  const completedSessions = sessions.filter(s => s.state === 'complete');

  if (sessions.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-slate-400">
        <Download size={48} className="mb-4 opacity-50" />
        <p>No active downloads</p>
        <p className="text-sm mt-2">Join a server to start downloading mods</p>
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

      {/* Completed */}
      {completedSessions.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-slate-300 mb-3 uppercase tracking-wide">
            Completed ({completedSessions.length})
          </h3>
          <div className="space-y-2">
            {completedSessions.map(session => (
              <CompletedCard key={session.sessionId} session={session} />
            ))}
          </div>
        </div>
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

function CompletedCard({ session }: { session: SessionStatus }) {
  return (
    <div className="flex items-center gap-3 p-3 bg-slate-800/50 rounded-lg">
      <CheckCircle size={16} className="text-emerald-400" />
      <span className="text-sm text-slate-400 flex-1">
        Session {session.sessionId.slice(-8)}
      </span>
      <span className="text-xs text-emerald-400">Ready</span>
    </div>
  );
}
