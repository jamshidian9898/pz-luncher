import { useState } from 'react'
import Dashboard from './pages/Dashboard'
import ServerDetail from './pages/ServerDetail'
import { Shield } from 'lucide-react'

type View =
  | { page: 'dashboard' }
  | { page: 'server'; id: string; name: string }

export default function App() {
  const [view, setView] = useState<View>({ page: 'dashboard' })

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100">
      {/* Top nav */}
      <header className="border-b border-slate-800 bg-slate-900/80 backdrop-blur sticky top-0 z-10">
        <div className="max-w-5xl mx-auto px-6 h-14 flex items-center gap-3">
          <div className="p-1.5 bg-indigo-600/20 rounded-lg">
            <Shield size={16} className="text-indigo-400" />
          </div>
          <button
            onClick={() => setView({ page: 'dashboard' })}
            className="font-bold text-slate-100 hover:text-white tracking-tight"
          >
            PZ Admin
          </button>
          {view.page === 'server' && (
            <span className="text-slate-600 text-sm">/ {view.name}</span>
          )}
          <div className="ml-auto text-xs text-slate-600">v2.0.0-alpha.1</div>
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
      </main>
    </div>
  )
}
