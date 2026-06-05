import { useState } from 'react';
import { useSessionStore } from '../stores/session.store';
import { useDownloadsStore } from '../stores/downloads.store';
import { useSettingsStore } from '../stores/settings.store';
import { ClipboardCopy, CheckCircle } from 'lucide-react';

function detectOS(): string {
  const ua = navigator.userAgent;
  if (ua.includes('Win')) return 'Windows';
  if (ua.includes('Mac')) return 'macOS';
  if (ua.includes('Linux')) return 'Linux';
  return 'Unknown';
}

function buildDiagnostics() {
  const session   = useSessionStore.getState();
  const downloads = useDownloadsStore.getState();
  const settings  = useSettingsStore.getState();

  const sessions = Array.from(downloads.sessions.values());
  const lastSession = sessions[sessions.length - 1];

  const joinDuration = (session.joinStartedAt && session.launchState === 'complete')
    ? Math.round((Date.now() - session.joinStartedAt) / 1000)
    : null;

  return {
    version: '1.0.0-beta',
    timestamp: new Date().toISOString(),
    os: detectOS(),
    launchState: session.launchState,
    lastError: session.lastError ?? null,
    currentServer: session.currentServer?.id ?? null,
    joinDurationSeconds: joinDuration,
    settings: {
      gamePath: settings.settings?.gamePath ? '(set)' : '(empty)',
      cacheLocation: settings.settings?.cacheLocation || '(default)',
      profilesLocation: settings.settings?.profilesLocation || '(default)',
      verifyChecksum: settings.settings?.verifyChecksum ?? true,
    },
    lastSession: lastSession
      ? {
          sessionId: lastSession.sessionId,
          state: lastSession.state,
          progress: lastSession.progress,
          errors: lastSession.errors ?? [],
        }
      : null,
    totalSessions: sessions.length,
    failedSessions: sessions.filter(s => s.state === 'error').length,
  };
}

export function DiagnosticsButton() {
  const [copied, setCopied] = useState(false);

  async function copyDiagnostics() {
    const data = buildDiagnostics();
    const text = JSON.stringify(data, null, 2);
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2500);
    } catch {
      // fallback: open in textarea
      const w = window.open('', '_blank');
      if (w) {
        w.document.write(`<pre style="font-family:monospace;font-size:13px;padding:16px">${text}</pre>`);
        w.document.title = 'PZ Launcher Diagnostics';
      }
    }
  }

  return (
    <button
      onClick={copyDiagnostics}
      className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all ${
        copied
          ? 'bg-emerald-600/20 border border-emerald-500/40 text-emerald-400'
          : 'bg-slate-700 hover:bg-slate-600 border border-slate-600 text-slate-300'
      }`}
    >
      {copied
        ? <><CheckCircle size={14} /> Copied!</>
        : <><ClipboardCopy size={14} /> Copy Diagnostics</>
      }
    </button>
  );
}
