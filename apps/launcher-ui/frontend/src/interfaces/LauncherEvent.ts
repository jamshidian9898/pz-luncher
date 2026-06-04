export enum LauncherEventType {
  SessionStart = 'session.start',
  SessionComplete = 'session.complete',
  ModResolveStart = 'mod.resolve.start',
  ModResolveComplete = 'mod.resolve.complete',
  DownloadStart = 'download.start',
  DownloadProgress = 'download.progress',
  DownloadComplete = 'download.complete',
  InstallStart = 'install.start',
  InstallComplete = 'install.complete',
  Error = 'error',
  LaunchStateChanged = 'launch.state.changed',
  ProviderDecision = 'provider.decision',
  TraceUpdated = 'trace.updated',
  ServerUpdated = 'server.updated',
  SettingsChanged = 'settings.changed',
}

export interface LauncherEventPayload {
  packageId?: string;
  progress?: {
    current: number;
    total: number;
    percent: number;
    speed?: number;
    eta?: number;
  };
  error?: string;
  metadata?: Record<string, unknown>;
}

export interface LauncherEvent {
  type: LauncherEventType;
  timestamp: number;
  version?: number;
  sessionId: string;
  payload?: LauncherEventPayload;
}

export function normalizeLauncherEvent(event: any): LauncherEvent {
  const timestamp = typeof event.timestamp === 'number' ? event.timestamp : Math.floor(Date.now() / 1000);
  const version = typeof event.version === 'number' ? event.version : 1;
  const sessionId = typeof event.sessionId === 'string' ? event.sessionId : 'unknown-session';
  const payload = {
    packageId: event.packageId as string | undefined,
    progress: event.progress as LauncherEventPayload['progress'],
    error: typeof event.error === 'string' ? event.error : undefined,
    metadata: event.metadata as Record<string, unknown> | undefined,
    ...(event.payload || {}),
  };

  switch (event.type) {
    case 'session.start':
      return { type: LauncherEventType.SessionStart, timestamp, version, sessionId, payload };
    case 'session.complete':
      return { type: LauncherEventType.SessionComplete, timestamp, version, sessionId, payload };
    case 'mod.resolve.start':
      return { type: LauncherEventType.ModResolveStart, timestamp, version, sessionId, payload };
    case 'mod.resolve.complete':
      return { type: LauncherEventType.ModResolveComplete, timestamp, version, sessionId, payload };
    case 'download.start':
      return { type: LauncherEventType.DownloadStart, timestamp, version, sessionId, payload };
    case 'download.progress':
      return { type: LauncherEventType.DownloadProgress, timestamp, version, sessionId, payload };
    case 'download.complete':
      return { type: LauncherEventType.DownloadComplete, timestamp, version, sessionId, payload };
    case 'install.start':
      return { type: LauncherEventType.InstallStart, timestamp, version, sessionId, payload };
    case 'install.complete':
      return { type: LauncherEventType.InstallComplete, timestamp, version, sessionId, payload };
    case 'error':
      return { type: LauncherEventType.Error, timestamp, version, sessionId, payload };
    case 'launch.state.changed':
      return { type: LauncherEventType.LaunchStateChanged, timestamp, version, sessionId, payload };
    case 'trace.updated':
      return { type: LauncherEventType.TraceUpdated, timestamp, version, sessionId, payload };
    case 'server.updated':
      return { type: LauncherEventType.ServerUpdated, timestamp, version, sessionId, payload };
    case 'settings.changed':
      return { type: LauncherEventType.SettingsChanged, timestamp, version, sessionId, payload };
    default:
      if (typeof event.type === 'string' && Object.values(LauncherEventType).includes(event.type as LauncherEventType)) {
        return { type: event.type as LauncherEventType, timestamp, version, sessionId, payload };
      }
      return { type: LauncherEventType.TraceUpdated, timestamp, version, sessionId, payload };
  }
}
