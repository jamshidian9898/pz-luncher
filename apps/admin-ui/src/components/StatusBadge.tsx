interface Props {
  status: string
}

const config: Record<string, { dot: string; text: string; label: string }> = {
  online:   { dot: 'bg-emerald-400', text: 'text-emerald-300', label: 'Online' },
  degraded: { dot: 'bg-yellow-400',  text: 'text-yellow-300',  label: 'Degraded' },
  offline:  { dot: 'bg-red-500',     text: 'text-red-400',     label: 'Offline' },
}

export default function StatusBadge({ status }: Props) {
  const c = config[status] ?? { dot: 'bg-slate-500', text: 'text-slate-400', label: status }
  return (
    <span className={`inline-flex items-center gap-1.5 text-xs font-medium ${c.text}`}>
      <span className={`w-2 h-2 rounded-full ${c.dot} ${status === 'online' ? 'animate-pulse' : ''}`} />
      {c.label}
    </span>
  )
}
