import { useEffect, useState, useMemo } from 'react';
import { ServerInfo, ServerDetails } from '../types';
import { launcherApi } from '../wails';
import { useSessionStore } from '../stores/session.store';
import { useDownloadsStore } from '../stores/downloads.store';
import {
  Monitor, Users, Wifi, Package, ChevronRight,
  RefreshCw, Star, StarOff, AlertCircle, Play,
  CheckCircle2, Loader2, Gamepad2, Download,
} from 'lucide-react';

interface ServerBrowserProps {
  onJoin: (server: ServerInfo) => void;
  onLaunch?: (server: ServerInfo) => void;
}

export function ServerBrowser({ onJoin, onLaunch }: ServerBrowserProps) {
  const [servers, setServers]       = useState<ServerInfo[]>([]);
  const [loading, setLoading]       = useState(true);
  const [error, setError]           = useState<string | null>(null);
  const [selected, setSelected]     = useState<string | null>(null);
  const [details, setDetails]       = useState<ServerDetails | null>(null);
  const [detailsLoading, setDetailsLoading] = useState(false);
  const [favorites, setFavorites]   = useState<Set<string>>(() => {
    try { return new Set(JSON.parse(localStorage.getItem('pz_favorites') ?? '[]')); }
    catch { return new Set(); }
  });

  useEffect(() => { load(); }, []);

  async function load() {
    setLoading(true);
    setError(null);
    try {
      const list = await launcherApi.getServerList();
      setServers(list);
    } catch {
      setError('Could not load server list. Is dev-api running?');
    } finally {
      setLoading(false);
    }
  }

  async function selectServer(server: ServerInfo) {
    setSelected(server.id);
    setDetails(null);
    setDetailsLoading(true);
    try {
      const d = await launcherApi.getServerDetails(server.id);
      setDetails(d);
    } catch {
      // details are optional
    } finally {
      setDetailsLoading(false);
    }
  }

  function toggleFavorite(id: string, e: React.MouseEvent) {
    e.stopPropagation();
    setFavorites(prev => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      localStorage.setItem('pz_favorites', JSON.stringify([...next]));
      return next;
    });
  }

  const sorted = [...servers].sort((a, b) => {
    const fa = favorites.has(a.id) ? 0 : 1;
    const fb = favorites.has(b.id) ? 0 : 1;
    return fa - fb;
  });

  if (loading) return <LoadingState />;
  if (error)   return <ErrorState message={error} onRetry={load} />;
  if (servers.length === 0) return <EmptyState onRetry={load} />;

  return (
    <div className="flex gap-4 h-full">
      {/* Server list */}
      <div className="flex flex-col gap-2 w-96 shrink-0 overflow-y-auto pr-1">
        <div className="flex items-center justify-between mb-1">
          <span className="text-xs text-slate-500 uppercase tracking-wide">
            {servers.length} server{servers.length !== 1 ? 's' : ''}
          </span>
          <button
            onClick={load}
            className="p-1.5 text-slate-500 hover:text-slate-300 rounded transition-colors"
            title="Refresh"
          >
            <RefreshCw size={14} />
          </button>
        </div>

        {sorted.map(server => (
          <ServerCard
            key={server.id}
            server={server}
            selected={selected === server.id}
            favorite={favorites.has(server.id)}
            onSelect={() => selectServer(server)}
            onFavorite={(e) => toggleFavorite(server.id, e)}
            onJoin={() => onJoin(server)}
            onLaunch={onLaunch ? () => onLaunch(server) : undefined}
          />
        ))}
      </div>

      {/* Detail panel */}
      <div className="flex-1 overflow-y-auto">
        {selected && (
          detailsLoading
            ? <DetailSkeleton />
            : details
            ? <DetailPanel details={details} onJoin={() => onJoin(details)} />
            : <DetailPanelFallback
                server={servers.find(s => s.id === selected)!}
                onJoin={() => {
                  const srv = servers.find(s => s.id === selected);
                  if (srv) onJoin(srv);
                }}
              />
        )}
        {!selected && (
          <div className="flex flex-col items-center justify-center h-full text-slate-500">
            <Monitor size={40} className="mb-3 opacity-30" />
            <p className="text-sm">Select a server to see details</p>
          </div>
        )}
      </div>
    </div>
  );
}

/* ── Server Card ── */
interface ServerCardProps {
  server: ServerInfo;
  selected: boolean;
  favorite: boolean;
  onSelect: () => void;
  onFavorite: (e: React.MouseEvent) => void;
  onJoin: () => void;
  onLaunch?: () => void;
}

function ServerCard({ server, selected, favorite, onSelect, onFavorite, onJoin, onLaunch }: ServerCardProps) {
  const status = useServerStatus(server.id);
  const fill = server.maxPlayers > 0
    ? Math.round((server.playerCount / server.maxPlayers) * 100)
    : 0;

  // Determine button state based on server status
  const showLaunch = status === 'ready' || status === 'running';
  const isRunning = status === 'running';
  const isDownloading = status === 'downloading';

  return (
    <div
      onClick={onSelect}
      className={`group relative rounded-lg border p-3 cursor-pointer transition-all ${
        selected
          ? 'bg-slate-700 border-emerald-500/60'
          : 'bg-slate-800 border-slate-700 hover:border-slate-600'
      }`}
    >
      <div className="flex items-start gap-3">
        {/* Favorite star */}
        <button
          onClick={onFavorite}
          className="mt-0.5 shrink-0 text-slate-600 hover:text-amber-400 transition-colors"
        >
          {favorite
            ? <Star size={14} className="fill-amber-400 text-amber-400" />
            : <StarOff size={14} />}
        </button>

        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between gap-2">
            <span className="font-medium text-slate-100 text-sm truncate">{server.name}</span>
            <ChevronRight size={14} className="text-slate-600 shrink-0" />
          </div>
          <p className="text-xs text-slate-500 truncate mt-0.5">{server.description}</p>

          <div className="flex items-center gap-3 mt-2 text-xs text-slate-400">
            <span className="flex items-center gap-1">
              <Users size={11} />
              {server.playerCount}/{server.maxPlayers}
            </span>
            <span className="flex items-center gap-1">
              <Wifi size={11} />
              {server.ping > 0 ? `${server.ping}ms` : '—'}
            </span>
            <span className="flex items-center gap-1">
              <Package size={11} />
              {server.modCount} mods
            </span>
            <StatusBadge status={status} />
          </div>

          {/* Player fill bar */}
          {server.maxPlayers > 0 && (
            <div className="mt-2 w-full bg-slate-700 rounded-full h-1">
              <div
                className={`h-1 rounded-full transition-all ${
                  fill >= 90 ? 'bg-red-500' : fill >= 60 ? 'bg-amber-500' : 'bg-emerald-500'
                }`}
                style={{ width: `${fill}%` }}
              />
            </div>
          )}
        </div>
      </div>

      {/* Join/Launch/Running button — appears on hover / select */}
      {isDownloading ? (
        <button
          disabled
          className="absolute right-3 bottom-3 px-3 py-1 rounded text-xs font-medium bg-slate-700 text-slate-400 cursor-not-allowed flex items-center gap-1"
        >
          <Loader2 size={12} className="animate-spin" /> Downloading
        </button>
      ) : showLaunch && onLaunch ? (
        <button
          onClick={(e) => { e.stopPropagation(); if (!isRunning) onLaunch(); }}
          disabled={isRunning}
          className={`absolute right-3 bottom-3 px-3 py-1 rounded text-xs font-medium transition-all flex items-center gap-1 ${
            isRunning
              ? 'bg-slate-700 text-slate-400 cursor-not-allowed'
              : selected
                ? 'bg-emerald-600 hover:bg-emerald-500 text-white opacity-100'
                : 'bg-emerald-600 hover:bg-emerald-500 text-white opacity-0 group-hover:opacity-100'
          }`}
        >
          {isRunning ? <><Gamepad2 size={12} /> Running</> : <><Play size={12} /> Launch</>}
        </button>
      ) : (
        <button
          onClick={(e) => { e.stopPropagation(); onJoin(); }}
          className={`absolute right-3 bottom-3 px-3 py-1 rounded text-xs font-medium transition-all ${
            selected
              ? 'bg-emerald-600 hover:bg-emerald-500 text-white opacity-100'
              : 'bg-emerald-600 hover:bg-emerald-500 text-white opacity-0 group-hover:opacity-100'
          }`}
        >
          Join
        </button>
      )}
    </div>
  );
}

/* ── Detail Panel ── */
function DetailPanel({ details, onJoin, onLaunch }: { details: ServerDetails; onJoin: () => void; onLaunch?: () => void }) {
  const totalMB = (details.totalSize / 1024 / 1024).toFixed(0);
  const status = useServerStatus(details.id);

  return (
    <div className="bg-slate-800 border border-slate-700 rounded-xl p-5 space-y-5">
      {/* Header */}
      <div className="flex items-start justify-between gap-4">
        <div>
          <h2 className="text-lg font-bold text-slate-100">{details.name}</h2>
          <div className="flex items-center gap-2 mt-1">
            <p className="text-sm text-slate-400">{details.description}</p>
            <StatusBadge status={status} />
          </div>
        </div>
        <ServerActionButton
          serverId={details.id}
          onJoin={onJoin}
          onLaunch={onLaunch}
        />
      </div>

      {/* Stats */}
      <div className="grid grid-cols-3 gap-3">
        <Stat label="Players"  value={`${details.playerCount} / ${details.maxPlayers}`} />
        <Stat label="Ping"     value={details.ping > 0 ? `${details.ping}ms` : '—'} />
        <Stat label="Download" value={`${totalMB} MB`} />
      </div>

      {/* Mod list */}
      <div>
        <h3 className="text-xs font-semibold text-slate-400 uppercase tracking-wide mb-2">
          Mods ({details.mods.length})
        </h3>
        <div className="space-y-1.5 max-h-64 overflow-y-auto pr-1">
          {details.mods.map(mod => (
            <div key={mod.id} className="flex items-center justify-between py-1.5 border-b border-slate-700/50 last:border-0">
              <div className="min-w-0">
                <span className="text-sm text-slate-200 truncate block">{mod.name}</span>
                {mod.workshopId && (
                  <span className="text-xs text-slate-500">Workshop #{mod.workshopId}</span>
                )}
              </div>
              <div className="flex items-center gap-2 shrink-0 ml-3">
                {!mod.required && (
                  <span className="text-xs px-1.5 py-0.5 rounded bg-slate-700 text-slate-400">optional</span>
                )}
                <span className="text-xs text-slate-500">
                  {mod.size > 0 ? `${(mod.size / 1024 / 1024).toFixed(0)} MB` : ''}
                </span>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

/* ── Server Action Button Helper ── */
function ServerActionButton({ serverId, onJoin, onLaunch }: { serverId: string; onJoin: () => void; onLaunch?: () => void }) {
  const status = useServerStatus(serverId);

  // Downloading state
  if (status === 'downloading') {
    return (
      <button
        disabled
        className="shrink-0 flex items-center gap-2 px-5 py-2 bg-slate-700 text-slate-400 rounded-lg text-sm font-medium cursor-not-allowed"
      >
        <Loader2 size={16} className="animate-spin" /> Downloading...
      </button>
    );
  }

  // Running state
  if (status === 'running') {
    return (
      <button
        disabled
        className="shrink-0 flex items-center gap-2 px-5 py-2 bg-slate-700 text-emerald-400 rounded-lg text-sm font-medium cursor-not-allowed"
      >
        <Gamepad2 size={16} /> Game Running
      </button>
    );
  }

  // Ready to launch
  if (status === 'ready' && onLaunch) {
    return (
      <button
        onClick={onLaunch}
        className="shrink-0 flex items-center gap-2 px-5 py-2 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg text-sm font-medium transition-colors"
      >
        <Play size={16} /> Launch Game
      </button>
    );
  }

  // Needs update
  if (status === 'needs-update') {
    return (
      <button
        onClick={onJoin}
        className="shrink-0 flex items-center gap-2 px-5 py-2 bg-amber-600 hover:bg-amber-500 text-white rounded-lg text-sm font-medium transition-colors"
      >
        <Download size={16} /> Update Mods
      </button>
    );
  }

  // Not joined - default join button
  return (
    <button
      onClick={onJoin}
      className="shrink-0 flex items-center gap-2 px-5 py-2 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg text-sm font-medium transition-colors"
    >
      Join Server
    </button>
  );
}

/* ── Detail Panel Fallback (when details API fails) ── */
function DetailPanelFallback({ server, onJoin, onLaunch }: { server: ServerInfo; onJoin: () => void; onLaunch?: () => void }) {
  return (
    <div className="bg-slate-800 border border-slate-700 rounded-xl p-5 space-y-5">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h2 className="text-lg font-bold text-slate-100">{server.name}</h2>
          <p className="text-sm text-slate-400 mt-1">{server.description}</p>
        </div>
        <ServerActionButton
          serverId={server.id}
          onJoin={onJoin}
          onLaunch={onLaunch}
        />
      </div>

      <div className="grid grid-cols-3 gap-3">
        <Stat label="Players" value={`${server.playerCount} / ${server.maxPlayers}`} />
        <Stat label="Ping" value={server.ping > 0 ? `${server.ping}ms` : '—'} />
        <Stat label="Mods" value={`${server.modCount}`} />
      </div>

      <div className="bg-slate-900 rounded-lg p-4">
        <p className="text-sm text-slate-400">
          Click <span className="text-emerald-400 font-medium">Join Server</span> to install mods and connect.
        </p>
      </div>
    </div>
  );
}

/* ── Helpers ── */
function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="bg-slate-900 rounded-lg p-3 text-center">
      <div className="text-lg font-bold text-slate-100">{value}</div>
      <div className="text-xs text-slate-400 mt-0.5">{label}</div>
    </div>
  );
}

function LoadingState() {
  return (
    <div className="flex flex-col gap-2">
      {[1,2,3].map(i => (
        <div key={i} className="h-24 bg-slate-800 rounded-lg animate-pulse" />
      ))}
    </div>
  );
}

function DetailSkeleton() {
  return (
    <div className="bg-slate-800 border border-slate-700 rounded-xl p-5 space-y-4 animate-pulse">
      <div className="h-6 bg-slate-700 rounded w-1/2" />
      <div className="h-4 bg-slate-700 rounded w-3/4" />
      <div className="grid grid-cols-3 gap-3">
        {[1,2,3].map(i => <div key={i} className="h-16 bg-slate-700 rounded-lg" />)}
      </div>
    </div>
  );
}

function ErrorState({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center h-64 gap-4 text-slate-400">
      <AlertCircle size={36} className="text-red-400" />
      <p className="text-sm text-center max-w-xs">{message}</p>
      <button
        onClick={onRetry}
        className="flex items-center gap-2 px-4 py-2 bg-slate-700 hover:bg-slate-600 rounded-lg text-sm transition-colors"
      >
        <RefreshCw size={14} />
        Retry
      </button>
    </div>
  );
}

function EmptyState({ onRetry }: { onRetry: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center h-64 gap-3 text-slate-400">
      <Monitor size={36} className="opacity-30" />
      <p className="text-sm">No servers found</p>
      <button onClick={onRetry} className="text-xs text-emerald-400 hover:underline flex items-center gap-1">
        <RefreshCw size={12} /> Refresh
      </button>
    </div>
  );
}

/* ── Server Status Helpers ── */
type ServerStatus = 'not-joined' | 'downloading' | 'ready' | 'running' | 'needs-update';

function useServerStatus(serverId: string): ServerStatus {
  const sessions = useDownloadsStore(s => s.sessions);
  const currentServer = useSessionStore(s => s.currentServer);
  const launchState = useSessionStore(s => s.launchState);

  return useMemo(() => {
    // Check if game is currently running for this server
    if (currentServer?.id === serverId && launchState === 'running') {
      return 'running';
    }

    // Find any session for this server
    const serverSessions = Array.from(sessions.values()).filter(
      s => s.serverId === serverId
    );

    // Check if currently downloading/installing
    const activeSession = serverSessions.find(
      s => s.state === 'downloading' || s.state === 'resolving' || s.state === 'installing'
    );
    if (activeSession) {
      return 'downloading';
    }

    // Check if completed/ready
    const completedSession = serverSessions.find(s => s.state === 'complete');
    if (completedSession) {
      // Check if currently selected and ready
      if (currentServer?.id === serverId && launchState === 'complete') {
        return 'ready';
      }
      // Has been joined before but not currently selected
      return 'ready';
    }

    return 'not-joined';
  }, [sessions, currentServer, launchState, serverId]);
}

function StatusBadge({ status }: { status: ServerStatus }) {
  const configs = {
    'not-joined': { icon: null, text: '', className: '' },
    'downloading': {
      icon: <Loader2 size={10} className="animate-spin" />,
      text: 'Downloading',
      className: 'bg-blue-500/20 text-blue-400'
    },
    'ready': {
      icon: <CheckCircle2 size={10} />,
      text: 'Ready',
      className: 'bg-emerald-500/20 text-emerald-400'
    },
    'running': {
      icon: <Gamepad2 size={10} />,
      text: 'Playing',
      className: 'bg-emerald-500/30 text-emerald-300'
    },
    'needs-update': {
      icon: <Download size={10} />,
      text: 'Update',
      className: 'bg-amber-500/20 text-amber-400'
    },
  };

  const config = configs[status];
  if (!config.text) return null;

  return (
    <span className={`flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-medium ${config.className}`}>
      {config.icon}
      {config.text}
    </span>
  );
}
