import { useEffect, useState, useCallback } from 'react'
import { AgentState, ServerRecord, fetchAgents, fetchServers } from '../api'
import StatusBadge from '../components/StatusBadge'
import MetricsBar from '../components/MetricsBar'
import { RefreshCw, Server, Users, Activity } from 'lucide-react'

interface Props {
  onSelectServer: (id: string, name: string) => void
}

function timeAgo(iso: string) {
  const diff = Math.floor((Date.now() - new Date(iso).getTime()) / 1000)
  if (diff < 60) return `${diff}s ago`
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`
  return `${Math.floor(diff / 3600)}h ago`
}

export default function Dashboard({ onSelectServer }: Props) {
  const [servers, setServers]       = useState<ServerRecord[]>([])
  const [agents, setAgents]         = useState<AgentState[]>([])
  const [loading, setLoading]       = useState(true)
  const [lastRefresh, setLastRefresh] = useState<Date>(new Date())

  const load = useCallback(async () => {
    try {
      const [s, a] = await Promise.all([fetchServers(), fetchAgents()])
      setServers(s)
      setAgents(a)
      setLastRefresh(new Date())
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
    const interval = setInterval(load, 30_000)
    return () => clearInterval(interval)
  }, [load])

  const agentMap = Object.fromEntries(agents.map(a => [a.serverId, a]))

  const onlineCount   = agents.filter(a => a.status === 'online').length
  const degradedCount = agents.filter(a => a.status === 'degraded').length
  const offlineCount  = agents.filter(a => a.status === 'offline').length

  return (
    <div className="space-y-6">
      {/* Stats bar */}
      <div className="grid grid-cols-3 gap-4">
        <div className="bg-slate-800/60 rounded-xl border border-slate-700/50 p-4 flex items-center gap-3">
          <div className="p-2 bg-indigo-600/20 rounded-lg"><Server size={18} className="text-indigo-400" /></div>
          <div>
            <div className="text-2xl font-bold text-slate-100">{servers.length}</div>
            <div className="text-xs text-slate-500">Servers</div>
          </div>
        </div>
        <div className="bg-slate-800/60 rounded-xl border border-slate-700/50 p-4 flex items-center gap-3">
          <div className="p-2 bg-emerald-600/20 rounded-lg"><Activity size={18} className="text-emerald-400" /></div>
          <div>
            <div className="text-2xl font-bold text-slate-100">{onlineCount}</div>
            <div className="text-xs text-slate-500">Online Agents</div>
          </div>
        </div>
        <div className="bg-slate-800/60 rounded-xl border border-slate-700/50 p-4 flex items-center gap-3">
          <div className="p-2 bg-slate-600/20 rounded-lg"><Users size={18} className="text-slate-400" /></div>
          <div>
            <div className="text-2xl font-bold text-slate-100">
              {servers.reduce((n, s) => n + (s.playerCount ?? 0), 0)}
            </div>
            <div className="text-xs text-slate-500">Total Players</div>
          </div>
        </div>
      </div>

      {/* Platform metrics */}
      <MetricsBar />

      {/* Agent summary pills */}
      {agents.length > 0 && (
        <div className="flex gap-3 text-xs">
          {onlineCount > 0   && <span className="px-2.5 py-1 rounded-full bg-emerald-900/40 text-emerald-300 border border-emerald-800/50">{onlineCount} online</span>}
          {degradedCount > 0 && <span className="px-2.5 py-1 rounded-full bg-yellow-900/40 text-yellow-300 border border-yellow-800/50">{degradedCount} degraded</span>}
          {offlineCount > 0  && <span className="px-2.5 py-1 rounded-full bg-red-900/40 text-red-300 border border-red-800/50">{offlineCount} offline</span>}
        </div>
      )}

      {/* Server list */}
      <div>
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-wider">Servers</h2>
          <button
            onClick={load}
            className="p-1.5 rounded-lg hover:bg-slate-700 text-slate-500 hover:text-slate-300 transition-colors"
            title="Refresh"
          >
            <RefreshCw size={14} className={loading ? 'animate-spin' : ''} />
          </button>
        </div>

        {loading ? (
          <div className="text-sm text-slate-500 py-8 text-center">Loading…</div>
        ) : servers.length === 0 ? (
          <div className="text-sm text-slate-500 py-8 text-center">No servers in registry.</div>
        ) : (
          <div className="space-y-2">
            {servers.map(server => {
              const agent = agentMap[server.id]
              return (
                <button
                  key={server.id}
                  onClick={() => onSelectServer(server.id, server.name)}
                  className="w-full text-left bg-slate-800/60 hover:bg-slate-700/60 border border-slate-700/50 hover:border-slate-600/50 rounded-xl p-4 transition-all group"
                >
                  <div className="flex items-start justify-between gap-4">
                    <div className="min-w-0">
                      <div className="font-semibold text-slate-100 group-hover:text-white transition-colors">
                        {server.name}
                      </div>
                      {server.description && (
                        <div className="text-xs text-slate-500 mt-0.5 truncate">{server.description}</div>
                      )}
                      <div className="flex items-center gap-3 mt-2 text-xs text-slate-500">
                        <span className="font-mono">{server.id}</span>
                        {server.gameVersion && <span>· {server.gameVersion}</span>}
                        {server.tags?.map(t => (
                          <span key={t} className="px-1.5 py-0.5 bg-slate-700/60 rounded text-slate-400">{t}</span>
                        ))}
                      </div>
                    </div>
                    <div className="flex flex-col items-end gap-2 shrink-0">
                      {agent ? (
                        <>
                          <StatusBadge status={agent.status} />
                          <span className="text-xs text-slate-500">{timeAgo(agent.lastSeen)}</span>
                          <span className="text-xs text-slate-400">{agent.modCount} mod{agent.modCount !== 1 ? 's' : ''}</span>
                        </>
                      ) : (
                        <span className="text-xs text-slate-600 italic">no agent</span>
                      )}
                    </div>
                  </div>
                </button>
              )
            })}
          </div>
        )}

        <div className="mt-3 text-right text-xs text-slate-600">
          Last refresh: {lastRefresh.toLocaleTimeString()} · auto every 30s
        </div>
      </div>
    </div>
  )
}
