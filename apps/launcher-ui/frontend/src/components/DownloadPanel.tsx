import { SessionStatus } from '../types';
import { useSessionStore } from '../stores/session.store';
import { Download, CheckCircle, Loader2, AlertCircle, Play, Clock, HardDrive, Gamepad2, File, Server } from 'lucide-react';

interface DownloadPanelProps {
  sessions: SessionStatus[];
  onLaunch?: () => void;
}

export function DownloadPanel({ sessions, onLaunch }: DownloadPanelProps) {
  const launchState = useSessionStore(s => s.launchState);
  const currentServer = useSessionStore(s => s.currentServer);
  const isComplete = launchState === 'complete';
  const isRunning = launchState === 'running';
  const isLaunching = launchState === 'launching';

  const activeSessions = sessions.filter(
    s => s.state === 'downloading' || s.state === 'resolving' || s.state === 'installing'
  );
  const completedSessions = sessions.filter(s => s.state === 'complete');
  const failedSessions = sessions.filter(s => s.state === 'error');

  // Calculate total stats
  const totalProgress = activeSessions.length > 0
    ? activeSessions.reduce((sum, s) => sum + s.progress, 0) / activeSessions.length
    : 0;
  const totalSpeed = activeSessions.reduce((sum, s) => sum + (s.downloadSpeed || 0), 0);

  if (sessions.length === 0 && !currentServer) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-slate-400">
        <Gamepad2 size={64} className="mb-4 opacity-30" />
        <p className="text-lg font-medium text-slate-300">No Active Downloads</p>
        <p className="text-sm mt-2">Join a server from the list to start downloading mods</p>
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="bg-slate-800/50 border-b border-slate-700 p-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 bg-blue-500/20 rounded-lg flex items-center justify-center">
              <Download size={20} className="text-blue-400" />
            </div>
            <div>
              <h2 className="font-semibold text-slate-100">Downloads</h2>
              <p className="text-xs text-slate-400">
                {activeSessions.length > 0
                  ? `${activeSessions.length} item${activeSessions.length > 1 ? 's' : ''} downloading`
                  : completedSessions.length > 0
                    ? `${completedSessions.length} ready to play`
                    : failedSessions.length > 0
                      ? `${failedSessions.length} failed`
                      : 'No active downloads'}
              </p>
            </div>
          </div>
          {activeSessions.length > 0 && (
            <div className="flex items-center gap-4 text-xs text-slate-400">
              <div className="flex items-center gap-1.5">
                <HardDrive size={14} />
                <span>{formatSpeed(totalSpeed)}</span>
              </div>
              <div className="flex items-center gap-1.5">
                <Clock size={14} />
                <span>{Math.round(totalProgress)}% complete</span>
              </div>
            </div>
          )}
        </div>
        {activeSessions.length > 0 && (
          <div className="mt-3">
            <div className="w-full bg-slate-700 rounded-full h-1.5">
              <div
                className="bg-blue-500 h-1.5 rounded-full transition-all duration-500"
                style={{ width: `${totalProgress}%` }}
              />
            </div>
          </div>
        )}
      </div>

      {/* Current Server Status */}
      {currentServer && (
        <div className={`p-4 border-b border-slate-700 ${
          isRunning ? 'bg-emerald-500/10' : isComplete ? 'bg-emerald-900/20' : 'bg-blue-900/20'
        }`}>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              {isRunning ? (
                <div className="w-10 h-10 bg-emerald-500/20 rounded-lg flex items-center justify-center">
                  <Gamepad2 size={20} className="text-emerald-400" />
                </div>
              ) : isComplete ? (
                <div className="w-10 h-10 bg-emerald-500/20 rounded-lg flex items-center justify-center">
                  <CheckCircle size={20} className="text-emerald-400" />
                </div>
              ) : (
                <div className="w-10 h-10 bg-blue-500/20 rounded-lg flex items-center justify-center">
                  <Loader2 size={20} className="text-blue-400 animate-spin" />
                </div>
              )}
              <div>
                <h3 className="font-medium text-slate-100">{currentServer.name}</h3>
                <p className="text-xs text-slate-400">
                  {isRunning ? 'Game is running' : isComplete ? 'Ready to launch' : isLaunching ? 'Launching...' : 'Preparing mods...'}
                </p>
              </div>
            </div>
            {(isComplete || isRunning) && onLaunch && (
              <button
                onClick={onLaunch}
                disabled={isRunning}
                className={`flex items-center gap-2 px-5 py-2 rounded-lg text-sm font-medium transition-colors ${
                  isRunning
                    ? 'bg-slate-700 text-slate-400 cursor-not-allowed'
                    : 'bg-emerald-600 hover:bg-emerald-500 text-white'
                }`}
              >
                {isRunning ? (
                  <><Gamepad2 size={16} /> Running</>
                ) : (
                  <><Play size={16} /> Launch Game</>
                )}
              </button>
            )}
          </div>
        </div>
      )}

      {/* Download Queue */}
      <div className="flex-1 overflow-auto p-4 space-y-4">
        {activeSessions.length > 0 && (
          <div className="space-y-2">
            <h4 className="text-xs font-semibold text-slate-500 uppercase tracking-wide px-1">
              Downloading ({activeSessions.length})
            </h4>
            {activeSessions.map(session => (
              <DownloadRow key={session.sessionId} session={session} />
            ))}
          </div>
        )}

        {completedSessions.length > 0 && (
          <div className="space-y-2">
            <h4 className="text-xs font-semibold text-slate-500 uppercase tracking-wide px-1 mt-6">
              Ready to Play ({completedSessions.length})
            </h4>
            {completedSessions.map(session => (
              <CompletedRow key={session.sessionId} session={session} />
            ))}
          </div>
        )}

        {failedSessions.length > 0 && (
          <div className="space-y-2">
            <h4 className="text-xs font-semibold text-red-500 uppercase tracking-wide px-1 mt-6">
              Failed ({failedSessions.length})
            </h4>
            {failedSessions.map(session => (
              <FailedRow key={session.sessionId} session={session} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

// Helper functions
function formatSpeed(speed?: number): string {
  if (!speed) return '';
  if (speed > 1024 * 1024) return `${(speed / 1024 / 1024).toFixed(1)} MB/s`;
  if (speed > 1024) return `${(speed / 1024).toFixed(1)} KB/s`;
  return `${speed} B/s`;
}

function formatETA(eta?: number): string {
  if (!eta) return '';
  if (eta > 3600) return `${Math.floor(eta / 3600)}h ${Math.floor((eta % 3600) / 60)}m`;
  if (eta > 60) return `${Math.floor(eta / 60)}m ${eta % 60}s`;
  return `${eta}s`;
}

function formatBytes(bytes?: number): string {
  if (bytes === undefined || bytes === null) return '0 B';
  if (bytes > 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`;
  if (bytes > 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
  if (bytes > 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${bytes} B`;
}

function DownloadRow({ session }: { session: SessionStatus }) {
  const hasSize = session.downloadTotal && session.downloadTotal > 0;
  const title = session.serverName || session.currentMod || 'Preparing...';
  return (
    <div className="bg-slate-800/50 hover:bg-slate-800 border border-slate-700/50 rounded-lg p-3 transition-colors">
      <div className="flex items-center gap-3">
        <div className="w-10 h-10 bg-blue-500/10 rounded-lg flex items-center justify-center shrink-0">
          {session.state === 'downloading' ? (
            <Loader2 size={18} className="text-blue-400 animate-spin" />
          ) : (
            <div className="w-4 h-4 rounded-full border-2 border-slate-600 border-t-blue-400 animate-spin" />
          )}
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between mb-1">
            <span className="font-medium text-slate-200 truncate" title={title}>
              {title}
            </span>
            <span className="text-sm text-slate-400 shrink-0 ml-2">{Math.round(session.progress)}%</span>
          </div>
          <div className="w-full bg-slate-700 rounded-full h-2 mb-2">
            <div
              className="bg-blue-500 h-2 rounded-full transition-all duration-300"
              style={{ width: `${session.progress}%` }}
            />
          </div>
          <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-slate-400">
            {session.downloadSpeed ? (
              <span className="flex items-center gap-1">
                <HardDrive size={12} />
                {formatSpeed(session.downloadSpeed)}
              </span>
            ) : null}
            {hasSize ? (
              <span className="flex items-center gap-1" title="Downloaded size">
                <File size={12} />
                {formatBytes(session.downloadCurrent)} / {formatBytes(session.downloadTotal)}
              </span>
            ) : null}
            {session.eta !== undefined && session.eta > 0 ? (
              <span className="flex items-center gap-1">
                <Clock size={12} />
                {formatETA(session.eta)} remaining
              </span>
            ) : null}
            <span className="ml-auto capitalize text-slate-500">{session.state}</span>
          </div>
          {session.currentModUrl && (
            <p className="text-xs text-slate-600 break-all mt-1.5" title="Download URL">
              {session.currentModUrl}
            </p>
          )}
        </div>
      </div>
    </div>
  );
}

function CompletedRow({ session }: { session: SessionStatus }) {
  return (
    <div className="bg-slate-800/30 hover:bg-slate-800/50 border border-slate-700/30 rounded-lg p-3 transition-colors">
      <div className="flex items-center gap-3">
        <div className="w-10 h-10 bg-emerald-500/10 rounded-lg flex items-center justify-center shrink-0">
          <CheckCircle size={18} className="text-emerald-400" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between">
            <span className="font-medium text-slate-200 truncate">
              {session.serverName || `Session ${session.sessionId.slice(-8)}`}
            </span>
            <span className="text-xs text-emerald-400 shrink-0 ml-2">Ready</span>
          </div>
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-slate-500 mt-0.5">
            {session.serverId && (
              <span className="flex items-center gap-1">
                <Server size={12} />
                {session.serverId}
              </span>
            )}
            {session.gameVersion && (
              <span className="flex items-center gap-1">
                <Gamepad2 size={12} />
                Game {session.gameVersion}
              </span>
            )}
            {session.manifestVersion && (
              <span className="flex items-center gap-1">
                <File size={12} />
                Manifest v{session.manifestVersion}
              </span>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function FailedRow({ session }: { session: SessionStatus }) {
  const lastError = session.errors?.[session.errors.length - 1] ?? 'Unknown error';
  return (
    <div className="bg-red-900/10 hover:bg-red-900/20 border border-red-500/20 rounded-lg p-3 transition-colors">
      <div className="flex items-center gap-3">
        <div className="w-10 h-10 bg-red-500/10 rounded-lg flex items-center justify-center shrink-0">
          <AlertCircle size={18} className="text-red-400" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between">
            <span className="font-medium text-slate-200 truncate">
              {session.serverName || `Session ${session.sessionId.slice(-8)}`}
            </span>
            <span className="text-xs text-red-400 shrink-0 ml-2">Failed</span>
          </div>
          <p className="text-xs text-red-400/80 truncate" title={lastError}>{lastError}</p>
        </div>
      </div>
    </div>
  );
}
