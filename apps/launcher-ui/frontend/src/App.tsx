import { useState, ReactNode } from 'react';
import { ServerList } from './components/ServerList';
import { ServerDetail } from './components/ServerDetail';
import { DownloadPanel } from './components/DownloadPanel';
import { SettingsPanel } from './components/SettingsPanel';
import { TraceViewer } from './components/TraceViewer';
import { useLauncherEvents } from './hooks/useRealEvents';
import { useDownloadsStore } from './stores/downloads.store';
import { useServersStore } from './stores/servers.store';
import { useSessionStore } from './stores/session.store';
import { launcherApi } from './wails';
import { ServerInfo } from './types';
import { Home, Download, Settings, Activity } from 'lucide-react';

type View = 'servers' | 'downloads' | 'settings' | 'trace';

function App() {
  const [currentView, setCurrentView] = useState<View>('servers');

  // Phase 2.1: Connect real launcher events → Zustand stores
  useLauncherEvents();

  // Phase 2.2: Read state from stores
  const { selectedServer, selectServer } = useServersStore();
  const { getActiveDownloads } = useDownloadsStore();
  const launchState = useSessionStore((s) => s.launchState);
  const currentServer = useSessionStore((s) => s.currentServer);

  const handleJoinServer = async (server: ServerInfo) => {
    try {
      useSessionStore.getState().setCurrentServer(server);
      await launcherApi.joinServer(server.id);
      setCurrentView('downloads');
    } catch (err) {
      console.error('Failed to join server:', err);
    }
  };

  const handleLaunchServer = async (server: ServerInfo) => {
    try {
      await launcherApi.launchServer(server.id);
    } catch (err) {
      console.error('Failed to launch game:', err);
    }
  };

  const canLaunch =
    launchState === 'complete' &&
    currentServer != null &&
    selectedServer?.id === currentServer.id;

  const hasActiveDownloads = getActiveDownloads().length > 0;

  return (
    <div className="flex h-screen bg-slate-900 text-slate-100 font-sans">
      {/* Sidebar */}
      <div className="w-64 bg-slate-800 border-r border-slate-700 flex flex-col">
        <div className="p-6">
          <h1 className="text-xl font-bold text-emerald-400">PZ Launcher</h1>
          <p className="text-xs text-slate-400 mt-1">v1.0.0</p>
        </div>
        
        <nav className="flex-1 px-4">
          <SidebarButton 
            icon={<Home size={20} />}
            label="Servers"
            active={currentView === 'servers'}
            onClick={() => setCurrentView('servers')}
          />
          <SidebarButton 
            icon={<Download size={20} />}
            label="Downloads"
            active={currentView === 'downloads'}
            onClick={() => setCurrentView('downloads')}
            badge={hasActiveDownloads ? '●' : undefined}
          />
          <SidebarButton 
            icon={<Activity size={20} />}
            label="Session Trace"
            active={currentView === 'trace'}
            onClick={() => setCurrentView('trace')}
          />
          <SidebarButton 
            icon={<Settings size={20} />}
            label="Settings"
            active={currentView === 'settings'}
            onClick={() => setCurrentView('settings')}
          />
        </nav>
        
        <div className="p-4 border-t border-slate-700">
          <div className="flex items-center gap-2 text-sm text-slate-400">
            <div className="w-2 h-2 rounded-full bg-emerald-500" />
            <span>Ready</span>
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="flex-1 flex flex-col overflow-hidden">
        <header className="bg-slate-800 border-b border-slate-700 px-6 py-4">
          <h2 className="text-lg font-semibold">
            {currentView === 'servers' && 'Server List'}
            {currentView === 'downloads' && 'Active Downloads'}
            {currentView === 'trace' && 'Session Trace'}
            {currentView === 'settings' && 'Settings'}
          </h2>
        </header>

        <main className="flex-1 overflow-auto p-6">
          {currentView === 'servers' && (
            <ServerList 
              onSelectServer={(server) => selectServer(server)}
              onJoinServer={handleJoinServer}
            />
          )}
          
          {currentView === 'downloads' && (
            <DownloadPanel sessions={getActiveDownloads()} />
          )}
          
          {currentView === 'settings' && <SettingsPanel />}
          
          {currentView === 'trace' && (
            <TraceViewer />
          )}
        </main>
      </div>

      {/* Server Detail Modal */}
      {selectedServer && (
        <ServerDetail 
          server={selectedServer}
          onClose={() => selectServer(null)}
          onJoin={() => handleJoinServer(selectedServer)}
          onLaunch={canLaunch ? () => handleLaunchServer(selectedServer) : undefined}
        />
      )}
    </div>
  );
}

interface SidebarButtonProps {
  icon: ReactNode;
  label: string;
  active: boolean;
  onClick: () => void;
  badge?: string;
}

function SidebarButton({ icon, label, active, onClick, badge }: SidebarButtonProps) {
  return (
    <button
      onClick={onClick}
      className={`w-full flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
        active 
          ? 'bg-emerald-600 text-white' 
          : 'text-slate-400 hover:bg-slate-700 hover:text-slate-200'
      }`}
    >
      {icon}
      <span className="flex-1 text-left">{label}</span>
      {badge && (
        <span className="text-emerald-400 text-xs">{badge}</span>
      )}
    </button>
  );
}

export default App;
