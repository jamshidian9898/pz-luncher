import { useEffect, useState } from 'react';
import { Settings } from '../types';
import { useSettingsStore } from '../stores/settings.store';
import { HealthCheck } from './HealthCheck';
import { DiagnosticsButton } from './DiagnosticsButton';
import { EventLogPanel } from './EventLogPanel';
import { Folder, Save, RefreshCw, SlidersHorizontal, ShieldCheck, Activity } from 'lucide-react';

type Tab = 'general' | 'health' | 'events';

export function SettingsPanel() {
  const settings = useSettingsStore((state) => state.settings);
  const loading = useSettingsStore((state) => state.loading);
  const fetchSettings = useSettingsStore((state) => state.fetchSettings);
  const saveSettings = useSettingsStore((state) => state.saveSettings);
  const setSettings = useSettingsStore((state) => state.setSettings);
  const [saving, setSaving] = useState(false);
  const [tab, setTab] = useState<Tab>('general');

  useEffect(() => {
    fetchSettings();
  }, [fetchSettings]);

  const handleSave = async () => {
    if (!settings) return;
    setSaving(true);
    await saveSettings(settings);
    setSaving(false);
  };

  if (!settings || loading) {
    return (
      <div className="flex items-center justify-center h-full text-slate-400">
        <div className="animate-spin mr-2">⟳</div>
        Loading settings...
      </div>
    );
  }

  return (
    <div className="max-w-2xl space-y-6">
      {/* Tab bar */}
      <div className="flex gap-1 p-1 bg-slate-800 rounded-lg w-fit">
        <TabButton icon={<SlidersHorizontal size={14} />} label="General" active={tab === 'general'} onClick={() => setTab('general')} />
        <TabButton icon={<ShieldCheck size={14} />} label="Health Check" active={tab === 'health'} onClick={() => setTab('health')} />
        <TabButton icon={<Activity size={14} />} label="Event Log" active={tab === 'events'} onClick={() => setTab('events')} />
      </div>

      {tab === 'events' && (
        <div className="bg-slate-800 border border-slate-700 rounded-lg p-5">
          <EventLogPanel />
        </div>
      )}

      {tab === 'health' && (
        <div className="space-y-4">
          <div className="bg-slate-800 border border-slate-700 rounded-lg p-5">
            <HealthCheck />
          </div>
          <div className="bg-slate-800 border border-slate-700 rounded-lg px-5 py-4 flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-slate-200">Diagnostics</p>
              <p className="text-xs text-slate-500 mt-0.5">Copy system state as JSON — send this when reporting a bug</p>
            </div>
            <DiagnosticsButton />
          </div>
        </div>
      )}

      {tab === 'general' && (
        <>
        <SettingSection title="Paths">
        <SettingRow label="Game Install (PZ_PATH)">
          <input
            type="text"
            value={settings.gamePath}
            onChange={(e) => setSettings({ ...settings, gamePath: e.target.value })}
            placeholder="/path/to/ProjectZomboid"
            className="w-96 max-w-full bg-slate-900 border border-slate-700 rounded-lg px-3 py-2 text-sm text-slate-300 focus:outline-none focus:border-emerald-500"
          />
        </SettingRow>

        <SettingRow label="Backend URL">
          <input
            type="text"
            value={settings.backendUrl}
            onChange={(e) =>
              setSettings({ ...settings, backendUrl: e.target.value })
            }
            placeholder="http://localhost:8080"
            className="w-96 max-w-full bg-slate-900 border border-slate-700 rounded-lg px-3 py-2 text-sm text-slate-300 focus:outline-none focus:border-emerald-500"
          />
        </SettingRow>

        <SettingRow label="Cache Location">
          <div className="flex items-center gap-2">
            <input
              type="text"
              value={settings.cacheLocation}
              onChange={(e) =>
                setSettings({ ...settings, cacheLocation: e.target.value })
              }
              className="flex-1 bg-slate-900 border border-slate-700 rounded-lg px-3 py-2 text-sm text-slate-300 focus:outline-none focus:border-emerald-500"
            />
            <button className="p-2 bg-slate-700 hover:bg-slate-600 rounded-lg transition-colors">
              <Folder size={18} />
            </button>
          </div>
        </SettingRow>

        <SettingRow label="Profiles Location">
          <div className="flex items-center gap-2">
            <input
              type="text"
              value={settings.profilesLocation}
              onChange={(e) =>
                setSettings({ ...settings, profilesLocation: e.target.value })
              }
              className="flex-1 bg-slate-900 border border-slate-700 rounded-lg px-3 py-2 text-sm text-slate-300 focus:outline-none focus:border-emerald-500"
            />
            <button className="p-2 bg-slate-700 hover:bg-slate-600 rounded-lg transition-colors">
              <Folder size={18} />
            </button>
          </div>
        </SettingRow>
      </SettingSection>

      {/* Performance */}
      <SettingSection title="Performance">
        <SettingRow label="Max Concurrent Downloads">
          <input
            type="number"
            min={1}
            max={10}
            value={settings.maxConcurrent}
            onChange={(e) =>
              setSettings({ ...settings, maxConcurrent: parseInt(e.target.value) || 1 })
            }
            className="w-24 bg-slate-900 border border-slate-700 rounded-lg px-3 py-2 text-sm text-slate-300 focus:outline-none focus:border-emerald-500"
          />
        </SettingRow>

        <SettingRow label="Bandwidth Limit (MB/s)">
          <input
            type="number"
            min={0}
            value={settings.bandwidthLimit}
            onChange={(e) =>
              setSettings({ ...settings, bandwidthLimit: parseInt(e.target.value) || 0 })
            }
            className="w-24 bg-slate-900 border border-slate-700 rounded-lg px-3 py-2 text-sm text-slate-300 focus:outline-none focus:border-emerald-500"
          />
          <span className="text-xs text-slate-500 ml-2">0 = Unlimited</span>
        </SettingRow>
      </SettingSection>

      <SettingSection title="Integrity">
        <SettingRow label="Verify checksums after download">
          <input
            type="checkbox"
            checked={settings.verifyChecksum}
            onChange={(e) => setSettings({ ...settings, verifyChecksum: e.target.checked })}
            className="w-4 h-4 accent-emerald-500"
          />
        </SettingRow>
      </SettingSection>

      {/* Actions */}
      <div className="flex items-center justify-between pt-4">
        <button
          onClick={fetchSettings}
          className="flex items-center gap-2 px-4 py-2 text-slate-400 hover:text-slate-200 transition-colors"
        >
          <RefreshCw size={16} />
          Reset
        </button>

        <button
          onClick={handleSave}
          disabled={saving}
          className="flex items-center gap-2 px-6 py-2 bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white rounded-lg font-medium transition-colors"
        >
          {saving ? (
            <>
              <div className="animate-spin">⟳</div>
              Saving...
            </>
          ) : (
            <>
              <Save size={18} />
              Save Settings
            </>
          )}
        </button>
      </div>
        </>
      )}
    </div>
  );
}

function TabButton({ icon, label, active, onClick }: { icon: React.ReactNode; label: string; active: boolean; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className={`flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${
        active ? 'bg-slate-700 text-slate-100' : 'text-slate-400 hover:text-slate-200'
      }`}
    >
      {icon}{label}
    </button>
  );
}

function SettingSection({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="bg-slate-800 border border-slate-700 rounded-lg p-6">
      <h3 className="text-lg font-semibold text-slate-200 mb-4">{title}</h3>
      <div className="space-y-4">{children}</div>
    </div>
  );
}

function SettingRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between">
      <label className="text-sm text-slate-400">{label}</label>
      <div className="flex items-center">{children}</div>
    </div>
  );
}
