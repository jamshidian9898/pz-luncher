import { useEffect, useState } from 'react'
import { AgentState, ManifestDiff, VersionSummary, fetchAgents, fetchDiff, fetchHistory } from '../api'
import StatusBadge from '../components/StatusBadge'
import DiffViewer from '../components/DiffViewer'
import { ArrowLeft, GitBranch, Clock, Package } from 'lucide-react'

interface Props {
  serverId: string
  serverName: string
  onBack: () => void
}

function timeAgo(iso: string) {
  const diff = Math.floor((Date.now() - new Date(iso).getTime()) / 1000)
  if (diff < 60) return `${diff}s ago`
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`
  return `${Math.floor(diff / 3600)}h ago`
}

export default function ServerDetail({ serverId, serverName, onBack }: Props) {
  const [agent, setAgent]     = useState<AgentState | null>(null)
  const [history, setHistory] = useState<VersionSummary[]>([])
  const [diff, setDiff]       = useState<ManifestDiff | null>(null)
  const [fromVer, setFromVer] = useState(0)
  const [toVer, setToVer]     = useState(0)

  useEffect(() => {
    async function load() {
      const [agents, hist] = await Promise.all([
        fetchAgents(),
        fetchHistory(serverId),
      ])
      const ag = agents.find(a => a.serverId === serverId) ?? null
      setAgent(ag)
      setHistory(hist)
      if (hist.length >= 2) {
        setToVer(hist[0].version)
        setFromVer(hist[1].version)
      } else if (hist.length === 1) {
        setToVer(hist[0].version)
        setFromVer(0)
      }
    }
    load()
  }, [serverId])

  useEffect(() => {
    if (toVer === 0) return
    fetchDiff(serverId, fromVer, toVer).then(setDiff)
  }, [serverId, fromVer, toVer])

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <button
          onClick={onBack}
          className="p-2 rounded-lg hover:bg-slate-800 text-slate-400 hover:text-slate-200 transition-colors"
        >
          <ArrowLeft size={18} />
        </button>
        <div>
          <h2 className="text-xl font-bold text-slate-100">{serverName}</h2>
          <div className="text-xs text-slate-500 font-mono">{serverId}</div>
        </div>
        {agent && <div className="ml-auto"><StatusBadge status={agent.status} /></div>}
      </div>

      {/* Agent info card */}
      {agent ? (
        <div className="bg-slate-800/60 rounded-xl border border-slate-700/50 p-4 grid grid-cols-3 gap-4">
          <div>
            <div className="text-xs text-slate-500 mb-1 flex items-center gap-1"><Clock size={11} /> Last Seen</div>
            <div className="text-sm text-slate-200">{timeAgo(agent.lastSeen)}</div>
          </div>
          <div>
            <div className="text-xs text-slate-500 mb-1 flex items-center gap-1"><Package size={11} /> Mods</div>
            <div className="text-sm text-slate-200">{agent.modCount}</div>
          </div>
          <div>
            <div className="text-xs text-slate-500 mb-1 flex items-center gap-1"><GitBranch size={11} /> Versions</div>
            <div className="text-sm text-slate-200">{history.length}</div>
          </div>
        </div>
      ) : (
        <div className="bg-slate-800/60 rounded-xl border border-slate-700/50 p-4 text-sm text-slate-500">
          No agent registered for this server yet.
        </div>
      )}

      <div className="grid grid-cols-2 gap-6">
        {/* Version history */}
        <div className="bg-slate-800/60 rounded-xl border border-slate-700/50 p-4">
          <h3 className="text-sm font-semibold text-slate-300 mb-3 flex items-center gap-2">
            <GitBranch size={14} /> Version History
          </h3>
          {history.length === 0 ? (
            <div className="text-sm text-slate-500 italic">No versions published yet.</div>
          ) : (
            <div className="space-y-1">
              {history.map(v => (
                <button
                  key={v.version}
                  onClick={() => {
                    const idx = history.findIndex(h => h.version === v.version)
                    setToVer(v.version)
                    setFromVer(history[idx + 1]?.version ?? 0)
                  }}
                  className={`w-full text-left px-3 py-2 rounded-lg flex justify-between items-center text-sm transition-colors
                    ${toVer === v.version
                      ? 'bg-indigo-600/30 border border-indigo-500/50 text-indigo-200'
                      : 'hover:bg-slate-700/60 text-slate-300 border border-transparent'}`}
                >
                  <span className="font-mono font-medium">v{v.version}</span>
                  <span className="text-xs text-slate-500">{v.modCount} mod{v.modCount !== 1 ? 's' : ''}</span>
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Diff viewer */}
        <div className="bg-slate-800/60 rounded-xl border border-slate-700/50 p-4">
          <h3 className="text-sm font-semibold text-slate-300 mb-3">Diff</h3>
          {diff ? (
            <DiffViewer diff={diff} />
          ) : (
            <div className="text-sm text-slate-500 italic">Select a version to view diff.</div>
          )}
        </div>
      </div>
    </div>
  )
}
