import { useEffect, useState } from 'react'
import { PlatformMetrics, fetchMetrics } from '../api'
import { Zap, Download, Upload, Heart, GitBranch } from 'lucide-react'

export default function MetricsBar() {
  const [m, setM] = useState<PlatformMetrics | null>(null)

  useEffect(() => {
    const load = () => fetchMetrics().then(setM).catch(() => {})
    load()
    const t = setInterval(load, 30_000)
    return () => clearInterval(t)
  }, [])

  if (!m) return null

  const items = [
    { icon: <Zap size={14} className="text-indigo-400" />,  label: 'Joins',     value: m.joinTotal },
    { icon: <Upload size={14} className="text-cyan-400" />,  label: 'Blob Uploads', value: m.blobUploadTotal },
    { icon: <Download size={14} className="text-sky-400" />, label: 'Downloads',  value: m.blobDownloadTotal },
    { icon: <GitBranch size={14} className="text-violet-400" />, label: 'Manifests', value: m.manifestPublishTotal },
    { icon: <Heart size={14} className="text-rose-400" />,   label: 'Heartbeats', value: m.heartbeatTotal },
  ]

  return (
    <div className="bg-slate-900/60 border border-slate-800 rounded-xl px-4 py-3 flex items-center gap-6 flex-wrap">
      <span className="text-xs font-semibold text-slate-500 uppercase tracking-wider shrink-0">Platform</span>
      {items.map(item => (
        <div key={item.label} className="flex items-center gap-2">
          {item.icon}
          <span className="text-sm font-semibold text-slate-200">{item.value.toLocaleString()}</span>
          <span className="text-xs text-slate-500">{item.label}</span>
        </div>
      ))}
    </div>
  )
}
