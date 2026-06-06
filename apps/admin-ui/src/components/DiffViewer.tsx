import { ManifestDiff, ModEntry } from '../api'
import { Plus, Minus, RefreshCw } from 'lucide-react'

interface Props {
  diff: ManifestDiff
}

function ModRow({ mod, icon, color }: { mod: ModEntry; icon: React.ReactNode; color: string }) {
  return (
    <div className={`flex items-center gap-3 px-3 py-2 rounded-lg ${color}`}>
      <span className="shrink-0">{icon}</span>
      <div className="min-w-0">
        <div className="text-sm font-medium text-slate-100 truncate">{mod.name || mod.id}</div>
        <div className="text-xs text-slate-400 font-mono truncate">{mod.sha256.slice(0, 16)}…</div>
      </div>
    </div>
  )
}

export default function DiffViewer({ diff }: Props) {
  const added   = diff.added   ?? []
  const removed = diff.removed ?? []
  const updated = diff.updated ?? []

  return (
    <div className="space-y-3">
      <div className="text-xs text-slate-500 mb-2">
        v{diff.fromVersion} → v{diff.toVersion} · {diff.unchanged} unchanged
      </div>

      {added.length > 0 && (
        <div className="space-y-1">
          <div className="text-xs font-semibold text-emerald-400 uppercase tracking-wide">
            Added ({added.length})
          </div>
          {added.map(m => (
            <ModRow key={m.sha256} mod={m}
              icon={<Plus size={14} className="text-emerald-400" />}
              color="bg-emerald-950/50 border border-emerald-900/50" />
          ))}
        </div>
      )}

      {removed.length > 0 && (
        <div className="space-y-1">
          <div className="text-xs font-semibold text-red-400 uppercase tracking-wide">
            Removed ({removed.length})
          </div>
          {removed.map(m => (
            <ModRow key={m.sha256} mod={m}
              icon={<Minus size={14} className="text-red-400" />}
              color="bg-red-950/50 border border-red-900/50" />
          ))}
        </div>
      )}

      {updated.length > 0 && (
        <div className="space-y-1">
          <div className="text-xs font-semibold text-yellow-400 uppercase tracking-wide">
            Updated ({updated.length})
          </div>
          {updated.map(m => (
            <ModRow key={m.sha256} mod={m}
              icon={<RefreshCw size={14} className="text-yellow-400" />}
              color="bg-yellow-950/50 border border-yellow-900/50" />
          ))}
        </div>
      )}

      {added.length === 0 && removed.length === 0 && updated.length === 0 && (
        <div className="text-sm text-slate-500 italic">No changes between these versions.</div>
      )}
    </div>
  )
}
