import { useState } from 'react';
import { Settings } from '../types';
import { settingsApi } from '../wails';
import { CheckCircle, Folder, ArrowRight, AlertCircle, Loader2 } from 'lucide-react';

interface FirstRunWizardProps {
  onComplete: (settings: Settings) => void;
}

type Step = 'welcome' | 'gamepath' | 'cache' | 'done';

const DEFAULT_SETTINGS: Settings = {
  gamePath: '',
  backendUrl: 'http://localhost:8080',
  cacheLocation: '',
  profilesLocation: '',
  maxConcurrent: 3,
  bandwidthLimit: 0,
  verifyChecksum: true,
};

export function FirstRunWizard({ onComplete }: FirstRunWizardProps) {
  const [step, setStep]         = useState<Step>('welcome');
  const [settings, setSettings] = useState<Settings>(DEFAULT_SETTINGS);
  const [saving, setSaving]     = useState(false);
  const [error, setError]       = useState<string | null>(null);

  async function finish() {
    setSaving(true);
    setError(null);
    try {
      await settingsApi.saveSettings(settings);
      onComplete(settings);
    } catch {
      setError('Could not save settings. Check permissions and try again.');
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="fixed inset-0 bg-slate-900 z-50 flex items-center justify-center p-6">
      <div className="w-full max-w-lg">
        {/* Progress dots */}
        <div className="flex items-center justify-center gap-2 mb-8">
          {(['welcome','gamepath','cache','done'] as Step[]).map((s, i) => (
            <div key={s} className="flex items-center gap-2">
              <div className={`w-2.5 h-2.5 rounded-full transition-colors ${
                s === step ? 'bg-emerald-400' :
                stepIndex(s) < stepIndex(step) ? 'bg-emerald-600' : 'bg-slate-700'
              }`} />
              {i < 3 && <div className={`h-px w-8 ${
                stepIndex(s) < stepIndex(step) ? 'bg-emerald-600' : 'bg-slate-700'
              }`} />}
            </div>
          ))}
        </div>

        {step === 'welcome' && (
          <WelcomeStep onNext={() => setStep('gamepath')} />
        )}
        {step === 'gamepath' && (
          <GamePathStep
            settings={settings}
            onChange={(v) => setSettings(s => ({ ...s, gamePath: v }))}
            onNext={() => setStep('cache')}
            onSkip={() => setStep('cache')}
          />
        )}
        {step === 'cache' && (
          <CacheStep
            settings={settings}
            onChange={(field, v) => setSettings(s => ({ ...s, [field]: v }))}
            onNext={finish}
            saving={saving}
            error={error}
          />
        )}
        {step === 'done' && (
          <DoneStep onComplete={() => onComplete(settings)} />
        )}
      </div>
    </div>
  );
}

function stepIndex(s: Step): number {
  return ['welcome','gamepath','cache','done'].indexOf(s);
}

/* ── Step: Welcome ── */
function WelcomeStep({ onNext }: { onNext: () => void }) {
  return (
    <div className="space-y-6 text-center">
      <div>
        <h1 className="text-3xl font-bold text-emerald-400 mb-2">PZ Launcher</h1>
        <p className="text-slate-400">
          Let's get you set up in a few quick steps.
        </p>
      </div>

      <div className="bg-slate-800 border border-slate-700 rounded-xl p-5 text-left space-y-3">
        {[
          'Find your Project Zomboid installation',
          'Set up the mod cache and profiles folder',
          'Start joining servers',
        ].map((text, i) => (
          <div key={i} className="flex items-center gap-3 text-sm text-slate-300">
            <div className="w-6 h-6 rounded-full bg-emerald-600/20 text-emerald-400 flex items-center justify-center text-xs font-bold shrink-0">
              {i + 1}
            </div>
            {text}
          </div>
        ))}
      </div>

      <button
        onClick={onNext}
        className="w-full flex items-center justify-center gap-2 py-3 bg-emerald-600 hover:bg-emerald-500 text-white rounded-xl font-medium transition-colors"
      >
        Get Started
        <ArrowRight size={18} />
      </button>
    </div>
  );
}

/* ── Step: Game Path ── */
interface GamePathStepProps {
  settings: Settings;
  onChange: (v: string) => void;
  onNext: () => void;
  onSkip: () => void;
}

function GamePathStep({ settings, onChange, onNext, onSkip }: GamePathStepProps) {
  const commonPaths = [
    '/Applications/ProjectZomboid',
    'C:\\Program Files (x86)\\Steam\\steamapps\\common\\ProjectZomboid',
    '~/.steam/steam/steamapps/common/ProjectZomboid',
  ];

  return (
    <div className="space-y-5">
      <div>
        <h2 className="text-xl font-bold text-slate-100 mb-1">Game Installation</h2>
        <p className="text-sm text-slate-400">
          Where is Project Zomboid installed? This is used to launch the game.
        </p>
      </div>

      <div className="space-y-2">
        <label className="text-xs text-slate-400 uppercase tracking-wide">Game Path</label>
        <div className="flex items-center gap-2">
          <input
            type="text"
            value={settings.gamePath}
            onChange={(e) => onChange(e.target.value)}
            placeholder="/path/to/ProjectZomboid"
            className="flex-1 bg-slate-900 border border-slate-700 rounded-lg px-3 py-2.5 text-sm text-slate-300 focus:outline-none focus:border-emerald-500"
          />
          <button className="p-2.5 bg-slate-700 hover:bg-slate-600 rounded-lg transition-colors">
            <Folder size={18} className="text-slate-300" />
          </button>
        </div>
      </div>

      <div className="space-y-1.5">
        <p className="text-xs text-slate-500">Common locations:</p>
        {commonPaths.map(p => (
          <button
            key={p}
            onClick={() => onChange(p)}
            className="w-full text-left text-xs text-slate-400 hover:text-emerald-400 font-mono bg-slate-800/50 hover:bg-slate-800 px-3 py-1.5 rounded transition-colors truncate"
          >
            {p}
          </button>
        ))}
      </div>

      <div className="flex gap-3 pt-2">
        <button
          onClick={onSkip}
          className="flex-1 py-2.5 text-slate-400 hover:text-slate-200 text-sm transition-colors"
        >
          Skip for now
        </button>
        <button
          onClick={onNext}
          disabled={!settings.gamePath}
          className="flex-1 flex items-center justify-center gap-2 py-2.5 bg-emerald-600 hover:bg-emerald-500 disabled:opacity-40 text-white rounded-xl text-sm font-medium transition-colors"
        >
          Continue <ArrowRight size={16} />
        </button>
      </div>
    </div>
  );
}

/* ── Step: Cache & Profiles ── */
interface CacheStepProps {
  settings: Settings;
  onChange: (field: keyof Settings, v: string) => void;
  onNext: () => void;
  saving: boolean;
  error: string | null;
}

function CacheStep({ settings, onChange, onNext, saving, error }: CacheStepProps) {
  return (
    <div className="space-y-5">
      <div>
        <h2 className="text-xl font-bold text-slate-100 mb-1">Storage</h2>
        <p className="text-sm text-slate-400">
          Where should mods and profiles be stored? Leave blank for defaults.
        </p>
      </div>

      <div className="space-y-4">
        <PathField
          label="Mod Cache"
          hint="Downloaded mods are stored here and reused across servers"
          value={settings.cacheLocation}
          onChange={(v) => onChange('cacheLocation', v)}
          placeholder="(default: ./cache)"
        />
        <PathField
          label="Profiles"
          hint="Each server gets an isolated mod + save folder"
          value={settings.profilesLocation}
          onChange={(v) => onChange('profilesLocation', v)}
          placeholder="(default: ./profiles)"
        />
      </div>

      {error && (
        <div className="flex items-center gap-2 p-3 bg-red-900/20 border border-red-500/30 rounded-lg">
          <AlertCircle size={14} className="text-red-400 shrink-0" />
          <p className="text-xs text-red-400">{error}</p>
        </div>
      )}

      <button
        onClick={onNext}
        disabled={saving}
        className="w-full flex items-center justify-center gap-2 py-3 bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white rounded-xl font-medium transition-colors"
      >
        {saving
          ? <><Loader2 size={16} className="animate-spin" /> Saving…</>
          : <><CheckCircle size={16} /> Finish Setup</>
        }
      </button>
    </div>
  );
}

/* ── Step: Done ── */
function DoneStep({ onComplete }: { onComplete: () => void }) {
  return (
    <div className="space-y-6 text-center">
      <div className="w-16 h-16 rounded-full bg-emerald-500/20 flex items-center justify-center mx-auto">
        <CheckCircle size={36} className="text-emerald-400" />
      </div>
      <div>
        <h2 className="text-xl font-bold text-slate-100 mb-2">You're all set!</h2>
        <p className="text-sm text-slate-400">
          PZ Launcher is ready. Browse servers and start playing.
        </p>
      </div>
      <button
        onClick={onComplete}
        className="w-full py-3 bg-emerald-600 hover:bg-emerald-500 text-white rounded-xl font-medium transition-colors"
      >
        Open Server Browser
      </button>
    </div>
  );
}

/* ── Helper ── */
function PathField({
  label, hint, value, onChange, placeholder,
}: {
  label: string; hint: string; value: string;
  onChange: (v: string) => void; placeholder: string;
}) {
  return (
    <div className="space-y-1.5">
      <label className="text-xs text-slate-400 uppercase tracking-wide">{label}</label>
      <p className="text-xs text-slate-500">{hint}</p>
      <div className="flex items-center gap-2">
        <input
          type="text"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder}
          className="flex-1 bg-slate-900 border border-slate-700 rounded-lg px-3 py-2 text-sm text-slate-300 focus:outline-none focus:border-emerald-500"
        />
        <button className="p-2 bg-slate-700 hover:bg-slate-600 rounded-lg transition-colors">
          <Folder size={16} className="text-slate-300" />
        </button>
      </div>
    </div>
  );
}
