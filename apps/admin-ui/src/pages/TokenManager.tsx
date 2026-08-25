import { useCallback, useEffect, useState } from 'react'
import { TokenState, fetchTokens, issueToken, revokeToken } from '../api'
import StatusBadge from '../components/StatusBadge'
import { Check, Copy, KeyRound, Plus, RefreshCw, Trash2 } from 'lucide-react'

const STORAGE_KEY = 'pz-admin-token'

export default function TokenManager() {
  const [adminToken, setAdminToken] = useState(() => localStorage.getItem(STORAGE_KEY) ?? '')
  const [tokens, setTokens] = useState<TokenState[]>([])
  const [newServerId, setNewServerId] = useState('')
  const [issued, setIssued] = useState<{ serverId: string; token: string } | null>(null)
  const [copied, setCopied] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const load = useCallback(async () => {
    if (!adminToken) return
    setLoading(true)
    try {
      setError(null)
      setTokens(await fetchTokens(adminToken))
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
      setTokens([])
    } finally {
      setLoading(false)
    }
  }, [adminToken])

  useEffect(() => {
    load()
  }, [load])

  function saveToken(t: string) {
    setAdminToken(t)
    localStorage.setItem(STORAGE_KEY, t)
  }

  async function handleIssue() {
    const id = newServerId.trim()
    if (!id) return
    try {
      setError(null)
      const token = await issueToken(adminToken, id)
      setIssued({ serverId: id, token })
      setCopied(false)
      setNewServerId('')
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    }
  }

  async function handleRevoke(serverId: string) {
    try {
      setError(null)
      await revokeToken(adminToken, serverId)
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h2 className="text-xl font-bold text-slate-100 flex items-center gap-2">
            <KeyRound size={18} className="text-indigo-400" /> Agent Enrollment Tokens
          </h2>
          <div className="text-xs text-slate-500 mt-1">Issue or revoke agent tokens. Issued tokens are shown once.</div>
        </div>
        <button
          onClick={load}
          disabled={!adminToken}
          className="p-2 rounded-lg hover:bg-slate-800 text-slate-400 hover:text-slate-200 transition-colors disabled:opacity-40"
          title="Refresh"
        >
          <RefreshCw size={16} className={loading ? 'animate-spin' : ''} />
        </button>
      </div>

      {/* Admin key */}
      <div className="bg-slate-800/60 rounded-xl border border-slate-700/50 p-4 flex gap-3 items-center">
        <label className="text-xs text-slate-500 shrink-0 w-24">Admin key</label>
        <input
          type="password"
          value={adminToken}
          onChange={e => saveToken(e.target.value)}
          placeholder="Value of -admin-token passed to the backend"
          className="flex-1 bg-slate-900/60 border border-slate-700/50 rounded-lg px-3 py-2 text-sm font-mono text-slate-200 placeholder:text-slate-600 focus:outline-none focus:border-indigo-500/50"
        />
      </div>

      {/* Issue new */}
      <div className="bg-slate-800/60 rounded-xl border border-slate-700/50 p-4 flex gap-3 items-center">
        <input
          value={newServerId}
          onChange={e => setNewServerId(e.target.value)}
          onKeyDown={e => e.key === 'Enter' && handleIssue()}
          placeholder="server-id (e.g. pz-test-3)"
          disabled={!adminToken}
          className="flex-1 bg-slate-900/60 border border-slate-700/50 rounded-lg px-3 py-2 text-sm font-mono text-slate-200 placeholder:text-slate-600 focus:outline-none focus:border-indigo-500/50 disabled:opacity-40"
        />
        <button
          onClick={handleIssue}
          disabled={!adminToken || !newServerId.trim()}
          className="px-4 py-2 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium flex items-center gap-2 transition-colors disabled:opacity-40"
        >
          <Plus size={14} /> Issue token
        </button>
      </div>

      {error && (
        <div className="bg-red-950/40 border border-red-900/50 text-red-300 text-sm rounded-lg px-4 py-3">{error}</div>
      )}

      {/* Newly issued token */}
      {issued && (
        <div className="bg-emerald-950/40 border border-emerald-800/50 rounded-xl p-4 space-y-2">
          <div className="text-sm font-semibold text-emerald-300">
            Token issued for <span className="font-mono">{issued.serverId}</span> — copy it now, it won't be shown again:
          </div>
          <div className="flex gap-2 items-center">
            <code className="flex-1 bg-slate-950/70 rounded-lg px-3 py-2 text-xs font-mono text-emerald-200 break-all">
              {issued.token}
            </code>
            <button
              onClick={() => {
                navigator.clipboard.writeText(issued.token)
                setCopied(true)
              }}
              className="p-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 transition-colors shrink-0"
              title="Copy"
            >
              {copied ? <Check size={14} className="text-emerald-400" /> : <Copy size={14} />}
            </button>
          </div>
        </div>
      )}

      {/* Token list */}
      {adminToken && (
        <div className="bg-slate-800/60 rounded-xl border border-slate-700/50 divide-y divide-slate-700/40">
          {tokens.length === 0 ? (
            <div className="px-4 py-8 text-sm text-slate-500 text-center italic">
              No agents with active tokens.
            </div>
          ) : (
            tokens.map(t => (
              <div key={t.serverId} className="flex items-center justify-between px-4 py-3 hover:bg-slate-700/30 transition-colors">
                <div className="min-w-0">
                  <div className="text-sm font-medium text-slate-200 font-mono">{t.serverId}</div>
                  <div className="text-xs mt-0.5">
                    {t.agentStatus ? (
                      <StatusBadge status={t.agentStatus} />
                    ) : (
                      <span className="text-slate-600 italic">no agent connected yet</span>
                    )}
                  </div>
                </div>
                <button
                  onClick={() => handleRevoke(t.serverId)}
                  className="p-2 rounded-lg hover:bg-red-950/60 text-slate-500 hover:text-red-400 transition-colors shrink-0"
                  title={`Revoke token for ${t.serverId}`}
                >
                  <Trash2 size={14} />
                </button>
              </div>
            ))
          )}
        </div>
      )}
    </div>
  )
}
