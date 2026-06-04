import { ServerInfo } from '../types';
import { X, Download, Play, CheckCircle, AlertCircle } from 'lucide-react';

interface ServerDetailProps {
  server: ServerInfo;
  onClose: () => void;
  onJoin: () => void;
  onLaunch?: () => void;
}

export function ServerDetail({ server, onClose, onJoin, onLaunch }: ServerDetailProps) {
  const canPlay = server.installed && server.upToDate;
  const needsUpdate = server.installed && !server.upToDate;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-slate-800 border border-slate-700 rounded-xl w-full max-w-2xl m-4">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-slate-700">
          <div>
            <h2 className="text-xl font-bold text-slate-100">{server.name}</h2>
            <p className="text-slate-400 text-sm mt-1">{server.description}</p>
          </div>
          <button
            onClick={onClose}
            className="p-2 hover:bg-slate-700 rounded-lg transition-colors"
          >
            <X size={20} className="text-slate-400" />
          </button>
        </div>

        {/* Content */}
        <div className="p-6 space-y-4">
          <StatusRow server={server} />

          <div className="grid grid-cols-3 gap-4">
            <StatCard label="Players" value={`${server.playerCount}/${server.maxPlayers}`} />
            <StatCard label="Ping" value={`${server.ping}ms`} />
            <StatCard label="Mods" value={`${server.modCount}`} />
          </div>

          <div className="bg-slate-900 rounded-lg p-4">
            <h3 className="text-sm font-semibold text-slate-300 mb-3">Mod Status</h3>
            <ModList server={server} />
          </div>
        </div>

        {/* Actions */}
        <div className="flex items-center justify-end gap-3 p-6 border-t border-slate-700">
          <button
            onClick={onClose}
            className="px-4 py-2 text-slate-400 hover:text-slate-200 transition-colors"
          >
            Cancel
          </button>
          {onLaunch ? (
            <button
              onClick={onLaunch}
              className="flex items-center gap-2 px-6 py-2 rounded-lg font-medium bg-emerald-600 hover:bg-emerald-500 text-white"
            >
              <Play size={18} />
              Launch Game
            </button>
          ) : (
            <button
              onClick={onJoin}
              className={`flex items-center gap-2 px-6 py-2 rounded-lg font-medium transition-colors ${
                canPlay
                  ? 'bg-emerald-600 hover:bg-emerald-500 text-white'
                  : needsUpdate
                  ? 'bg-amber-600 hover:bg-amber-500 text-white'
                  : 'bg-blue-600 hover:bg-blue-500 text-white'
              }`}
            >
              {canPlay ? (
                <>
                  <Play size={18} />
                  Play
                </>
              ) : needsUpdate ? (
                <>
                  <Download size={18} />
                  Update
                </>
              ) : (
                <>
                  <Download size={18} />
                  Install & Join
                </>
              )}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}

function StatusRow({ server }: { server: ServerInfo }) {
  if (server.installed && server.upToDate) {
    return (
      <div className="flex items-center gap-2 text-emerald-400 bg-emerald-400/10 rounded-lg p-3">
        <CheckCircle size={18} />
        <span className="font-medium">Ready to play - All mods installed</span>
      </div>
    );
  }

  if (server.installed && !server.upToDate) {
    return (
      <div className="flex items-center gap-2 text-amber-400 bg-amber-400/10 rounded-lg p-3">
        <AlertCircle size={18} />
        <span className="font-medium">Update available - Mods need refresh</span>
      </div>
    );
  }

  return (
    <div className="flex items-center gap-2 text-blue-400 bg-blue-400/10 rounded-lg p-3">
      <Download size={18} />
      <span className="font-medium">Installation required - {server.modCount} mods to download</span>
    </div>
  );
}

function StatCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="bg-slate-900 rounded-lg p-3 text-center">
      <div className="text-2xl font-bold text-slate-100">{value}</div>
      <div className="text-xs text-slate-400 mt-1">{label}</div>
    </div>
  );
}

function ModList({ server }: { server: ServerInfo }) {
  // Mock mod list based on server state
  const mods = [
    { name: 'Brita Weapons', status: server.installed ? 'installed' : 'missing', size: '100 MB' },
    { name: 'Common Sense', status: server.installed ? 'installed' : 'missing', size: '50 MB' },
    { name: 'True Music', status: server.installed ? 'installed' : 'missing', size: '20 MB' },
  ];

  return (
    <div className="space-y-2">
      {mods.map((mod) => (
        <div key={mod.name} className="flex items-center justify-between py-2 border-b border-slate-800 last:border-0">
          <div className="flex items-center gap-2">
            {mod.status === 'installed' ? (
              <CheckCircle size={14} className="text-emerald-400" />
            ) : (
              <Download size={14} className="text-blue-400" />
            )}
            <span className="text-sm text-slate-300">{mod.name}</span>
          </div>
          <span className="text-xs text-slate-500">{mod.size}</span>
        </div>
      ))}
    </div>
  );
}
