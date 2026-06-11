import { useState, ReactNode, useEffect } from 'react';
import { ServerBrowser } from './components/ServerBrowser';
import { ServerDetail } from './components/ServerDetail';
import { DownloadPanel } from './components/DownloadPanel';
import { SettingsPanel } from './components/SettingsPanel';
import { TraceViewer } from './components/TraceViewer';
import { SessionProgressCard } from './components/SessionProgressCard';
import { FirstRunWizard } from './components/FirstRunWizard';
import { useLauncherEvents } from './hooks/useRealEvents';
import { useDownloadsStore } from './stores/downloads.store';
import { useServersStore } from './stores/servers.store';
import { useSessionStore } from './stores/session.store';
import { useSettingsStore } from './stores/settings.store';
import { launcherApi } from './wails';
import { ServerInfo, Settings } from './types';
import { Home, Download, Settings as SettingsIcon, Activity } from 'lucide-react';

type View = 'servers' | 'downloads' | 'settings' | 'trace';

function App() {
  const [currentView, setCurrentView] = useState<View>('servers');
  const [showWizard, setShowWizard]   = useState(false);

  // Phase 2.1: Connect real launcher events → Zustand stores
  useLauncherEvents();

  // Phase 2.2: Read state from stores
  const { selectedServer, selectServer } = useServersStore();
  const { sessions } = useDownloadsStore();
  const launchState   = useSessionStore((s) => s.launchState);
  const currentServer = useSessionStore((s) => s.currentServer);
  const { fetchSettings, settings } = useSettingsStore();

  // First Run: show wizard if gamePath is empty
  useEffect(() => {
    fetchSettings().then(() => {
      const s = useSettingsStore.getState().settings;
      if (s && !s.gamePath) setShowWizard(true);
    });
  }, [fetchSettings]);

  const handleJoinServer = async (server: ServerInfo) => {
    // Guard: prevent multiple joins while one is in progress
    const currentSessionId = useSessionStore.getState().currentSessionId;
    const currentState = useSessionStore.getState().launchState;
    if (currentSessionId && (currentState === 'resolving' || currentState === 'downloading' || currentState === 'installing')) {
      // Already joining, just switch to downloads view
      setCurrentView('downloads');
      return;
    }

    useSessionStore.getState().resetSession();
    useSessionStore.getState().setCurrentServer(server);
    selectServer(null);
    setCurrentView('downloads');

    // Optimistic placeholder so DownloadPanel shows immediately (real events will overwrite)
    const optimisticId = `optimistic-${server.id}-${Date.now()}`;
    useSessionStore.getState().setCurrentSession(optimisticId);
    useDownloadsStore.getState().updateSession({
      sessionId: optimisticId,
      state: 'resolving',
      progress: 0,
      currentMod: 'Connecting to backend…',
      errors: [],
      serverName: server.name,
      serverId: server.id,
    });

    try {
      await launcherApi.joinServer(server.id);
    } catch (err) {
      console.error('Failed to join server:', err);
      useDownloadsStore.getState().failSession(optimisticId, String(err));
      useSessionStore.getState().setLaunchState('error');
      useSessionStore.getState().setLastError(String(err));
    }
  };

  const handleRetry = () => {
    if (currentServer) handleJoinServer(currentServer);
  };

  const handleRepairCache = async () => {
    // RFC-0040: repair cache via settings reset — placeholder for now
    alert('Cache repair: clear ' + (settings?.cacheLocation || './cache') + ' and retry.');
  };

  const handleLaunchServer = async (server: ServerInfo) => {
    useSessionStore.getState().setLaunchState('launching');
    try {
      await launcherApi.launchServer(server.id);
      // Launch initiated successfully - game is now running
      useSessionStore.getState().setLaunchState('running');
    } catch (err) {
      console.error('Failed to launch game:', err);
      useSessionStore.getState().setLaunchState('error');
      useSessionStore.getState().setLastError(String(err));
    }
  };

  const canLaunch =
    launchState === 'complete' &&
    currentServer != null;

  const hasActiveDownloads = Array.from(sessions.values()).some(
    s => s.state === 'downloading' || s.state === 'resolving' || s.state === 'installing'
  );

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
            icon={<SettingsIcon size={20} />}
            label="Settings"
            active={currentView === 'settings'}
            onClick={() => setCurrentView('settings')}
          />
        </nav>
        
        <div className="p-4 border-t border-slate-700 space-y-3">
          <SessionProgressCard
            onLaunch={canLaunch && currentServer ? () => handleLaunchServer(currentServer) : undefined}
            onRetry={launchState === 'error' ? handleRetry : undefined}
            onRepairCache={launchState === 'error' ? handleRepairCache : undefined}
          />
          {!currentServer && (
            <div className="flex items-center gap-2 text-sm text-slate-400">
              <div className="w-2 h-2 rounded-full bg-emerald-500" />
              <span>Ready</span>
            </div>
          )}
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
            <ServerBrowser
              onJoin={handleJoinServer}
              onLaunch={handleLaunchServer}
            />
          )}
          
          {currentView === 'downloads' && (
            <DownloadPanel
              sessions={Array.from(sessions.values())}
              onLaunch={canLaunch && currentServer ? () => handleLaunchServer(currentServer) : undefined}
            />
          )}
          
          {currentView === 'settings' && <SettingsPanel />}
          
          {currentView === 'trace' && (
            <TraceViewer />
          )}
        </main>
      </div>

      {/* First Run Wizard */}
      {showWizard && (
        <FirstRunWizard
          onComplete={(_s: Settings) => setShowWizard(false)}
        />
      )}

      {/* Server Detail Modal (legacy — still used from ServersStore) */}
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
