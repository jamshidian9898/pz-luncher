import { useCallback, useEffect, useMemo, useState } from 'react'
import { BlobInfo, fetchBlobs } from '../api'
import { Database, RefreshCw, Search } from 'lucide-react'

function formatSize(bytes: number) {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(2)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`
}

export default function ContentBrowser() {
  const [blobs, setBlobs] = useState<BlobInfo[]>([])
  const [totalBytes, setTotalBytes] = useState(0)
  const [query, setQuery] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const load = useCallback(async () => {
    try {
      setError(null)
      const d = await fetchBlobs()
      setBlobs(d.blobs ?? [])
      setTotalBytes(d.totalBytes ?? 0)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
    const t = setInterval(load, 60_000)
    return () => clearInterval(t)
  }, [load])

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return blobs
    return blobs.filter(b =>
      b.sha256.includes(q) ||
      (b.sourceServer ?? '').toLowerCase().includes(q),
    )
  }, [blobs, query])

  return (
    <div className="space-y-6">
      {/* Header + stats */}
      <div className="flex items-start justify-between gap-4">
        <div>
          <h2 className="text-xl font-bold text-slate-100 flex items-center gap-2">
            <Database size={18} className="text-indigo-400" /> Content Store
          </h2>
          <div className="text-xs text-slate-500 mt-1">
            {blobs.length} blob{blobs.length !== 1 ? 's' : ''} · {formatSize(totalBytes)} total
          </div>
        </div>
        <button
          onClick={load}
          className="p-2 rounded-lg hover:bg-slate-800 text-slate-400 hover:text-slate-200 transition-colors"
          title="Refresh"
        >
          <RefreshCw size={16} className={loading ? 'animate-spin' : ''} />
        </button>
      </div>

      {/* Search */}
      <div className="relative">
        <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500" />
        <input
          value={query}
          onChange={e => setQuery(e.target.value)}
          placeholder="Filter by sha256 or source server…"
          className="w-full bg-slate-800/60 border border-slate-700/50 rounded-lg pl-9 pr-3 py-2 text-sm text-slate-200 placeholder:text-slate-600 focus:outline-none focus:border-indigo-500/50"
        />
      </div>

      {error && (
        <div className="bg-red-950/40 border border-red-900/50 text-red-300 text-sm rounded-lg px-4 py-3">
          {error}
        </div>
      )}

      {/* Blob table */}
      {!error && (
        <div className="bg-slate-800/60 rounded-xl border border-slate-700/50 divide-y divide-slate-700/40">
          <div className="grid grid-cols-[1fr_90px_140px_70px_110px] gap-3 px-4 py-2.5 text-xs font-semibold text-slate-500 uppercase tracking-wider">
            <span>SHA256</span>
            <span className="text-right">Size</span>
            <span>Source</span>
            <span className="text-right">DLs</span>
            <span className="text-right">First Seen</span>
          </div>
          {filtered.length === 0 ? (
            <div className="px-4 py-8 text-sm text-slate-500 text-center italic">
              {loading ? 'Loading…' : query ? 'No blobs match this filter.' : 'Content store is empty.'}
            </div>
          ) : (
            filtered.map(b => (
              <div key={b.sha256} className="grid grid-cols-[1fr_90px_140px_70px_110px] gap-3 px-4 py-2.5 items-center hover:bg-slate-700/30 transition-colors">
                <a
                  href={`/api/v1/download/${b.sha256}`}
                  target="_blank"
                  rel="noreferrer"
                  className="font-mono text-xs text-slate-300 hover:text-indigo-400 truncate"
                  title={b.sha256}
                >
                  {b.sha256.slice(0, 24)}…
                </a>
                <span className="text-xs text-slate-400 text-right">{formatSize(b.sizeBytes)}</span>
                <span className="text-xs text-slate-400 truncate">{b.sourceServer || '—'}</span>
                <span className={`text-xs text-right ${b.downloads ? 'text-indigo-300' : 'text-slate-600'}`}>
                  {b.downloads ?? 0}
                </span>
                <span className="text-xs text-slate-500 text-right">
                  {b.firstSeenAt ? new Date(b.firstSeenAt).toLocaleDateString() : '—'}
                </span>
              </div>
            ))
          )}
        </div>
      )}
    </div>
  )
}
