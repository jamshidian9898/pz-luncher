import { useState } from 'react'
import Dashboard from './pages/Dashboard'
import ServerDetail from './pages/ServerDetail'
import ContentBrowser from './pages/ContentBrowser'
import TokenManager from './pages/TokenManager'
import { Database, KeyRound, Shield } from 'lucide-react'

type View =
  | { page: 'dashboard' }
  | { page: 'server'; id: string; name: string }
  | { page: 'content' }
  | { page: 'tokens' }

export default function App() {
  const [view, setView] = useState<View>({ page: 'dashboard' })

  const navBtn = (active: boolean) =>
    `px-3 py-1.5 rounded-lg text-xs font-medium flex items-center gap-1.5 transition-colors ${
      active
        ? 'bg-indigo-600/20 text-indigo-300 border border-indigo-500/40'
        : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800 border border-transparent'
    }`

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100">
      {/* Top nav */}
      <header className="border-b border-slate-800 bg-slate-900/80 backdrop-blur sticky top-0 z-10">
        <div className="max-w-5xl mx-auto px-6 h-14 flex items-center gap-3">
          <button onClick={() => setView({ page: 'dashboard' })} className="flex items-center gap-2">
            <div className="p-1.5 bg-indigo-600/20 rounded-lg">
              <Shield size={16} className="text-indigo-400" />
            </div>
            <span className="font-bold text-slate-100 hover:text-white tracking-tight">PZ Admin</span>
          </button>
          {view.page === 'server' && (
            <span className="text-slate-600 text-sm">/ {view.name}</span>
          )}
          <nav className="ml-auto flex items-center gap-1">
            <button onClick={() => setView({ page: 'content' })} className={navBtn(view.page === 'content')}>
              <Database size={13} /> Content
            </button>
            <button onClick={() => setView({ page: 'tokens' })} className={navBtn(view.page === 'tokens')}>
              <KeyRound size={13} /> Tokens
            </button>
          </nav>
        </div>
      </header>

      {/* Main */}
      <main className="max-w-5xl mx-auto px-6 py-8">
        {view.page === 'dashboard' && (
          <Dashboard
            onSelectServer={(id, name) => setView({ page: 'server', id, name })}
          />
        )}
        {view.page === 'server' && (
          <ServerDetail
            serverId={view.id}
            serverName={view.name}
            onBack={() => setView({ page: 'dashboard' })}
          />
        )}
        {view.page === 'content' && <ContentBrowser />}
        {view.page === 'tokens' && <TokenManager />}
      </main>
    </div>
  )
}
