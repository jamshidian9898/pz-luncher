import { useState, useEffect } from 'react';
import { ServerInfo } from '../types';
import { useServerList } from '../hooks/useLauncherEvents';
import { Monitor, Users, CheckCircle, AlertCircle } from 'lucide-react';

interface ServerListProps {
  onSelectServer: (server: ServerInfo) => void;
  onJoinServer: (server: ServerInfo) => void;
}

export function ServerList({ onSelectServer, onJoinServer }: ServerListProps) {
  const [servers, setServers] = useState<ServerInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const { fetchServers } = useServerList();

  useEffect(() => {
    loadServers();
  }, []);

  const loadServers = async () => {
    setLoading(true);
    const data = await fetchServers();
    setServers(data);
    setLoading(false);
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full text-slate-400">
        <div className="animate-spin mr-2">⟳</div>
        Loading servers...
      </div>
    );
  }

  return (
    <div className="grid gap-4">
      {servers.map((server) => (
        <ServerCard
          key={server.id}
          server={server}
          onClick={() => onSelectServer(server)}
          onJoin={() => onJoinServer(server)}
        />
      ))}
    </div>
  );
}

interface ServerCardProps {
  server: ServerInfo;
  onClick: () => void;
  onJoin: () => void;
}

function ServerCard({ server, onClick, onJoin }: ServerCardProps) {
  const statusColor = server.installed
    ? server.upToDate
      ? 'text-emerald-400'
      : 'text-amber-400'
    : 'text-slate-400';

  const statusText = server.installed
    ? server.upToDate
      ? 'Ready'
      : 'Update Available'
    : 'Not Installed';

  const statusIcon = server.installed ? (
    server.upToDate ? (
      <CheckCircle size={16} className="text-emerald-400" />
    ) : (
      <AlertCircle size={16} className="text-amber-400" />
    )
  ) : null;

  return (
    <div
      onClick={onClick}
      className="bg-slate-800 border border-slate-700 rounded-lg p-4 hover:border-slate-600 cursor-pointer transition-colors"
    >
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-1">
            <Monitor size={18} className="text-emerald-400" />
            <h3 className="text-lg font-semibold text-slate-100">{server.name}</h3>
          </div>
          <p className="text-slate-400 text-sm mb-3">{server.description}</p>

          <div className="flex items-center gap-4 text-sm">
            <div className="flex items-center gap-1 text-slate-400">
              <Users size={14} />
              <span>
                {server.playerCount}/{server.maxPlayers}
              </span>
            </div>
            <div className="text-slate-400">{server.ping}ms</div>
            <div className="text-slate-400">{server.modCount} mods</div>
          </div>
        </div>

        <div className="flex flex-col items-end gap-2">
          <div className={`flex items-center gap-1 text-sm ${statusColor}`}>
            {statusIcon}
            <span>{statusText}</span>
          </div>

          <button
            onClick={(e) => {
              e.stopPropagation();
              onJoin();
            }}
            className="px-4 py-2 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg text-sm font-medium transition-colors"
          >
            {server.installed ? 'Play' : 'Install'}
          </button>
        </div>
      </div>
    </div>
  );
}
