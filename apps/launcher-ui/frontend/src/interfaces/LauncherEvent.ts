export enum LauncherEventType {
  SessionStarted = 'SessionStarted',
  SessionCompleted = 'SessionCompleted',

  DownloadStarted = 'DownloadStarted',
  DownloadProgress = 'DownloadProgress',
  DownloadCompleted = 'DownloadCompleted',

  LaunchStateChanged = 'LaunchStateChanged',
  ProviderDecision = 'ProviderDecision',
  TraceUpdated = 'TraceUpdated',
  ServerUpdated = 'ServerUpdated',
  SettingsChanged = 'SettingsChanged',
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
  sessionId: string;
  payload?: LauncherEventPayload;
}

export function normalizeLauncherEvent(event: any): LauncherEvent {
  const timestamp = typeof event.timestamp === 'number' ? event.timestamp : Math.floor(Date.now() / 1000);
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
      return { type: LauncherEventType.SessionStarted, timestamp, sessionId, payload };
    case 'session.complete':
      return { type: LauncherEventType.SessionCompleted, timestamp, sessionId, payload };
    case 'download.start':
      return { type: LauncherEventType.DownloadStarted, timestamp, sessionId, payload };
    case 'download.progress':
      return { type: LauncherEventType.DownloadProgress, timestamp, sessionId, payload };
    case 'download.complete':
      return { type: LauncherEventType.DownloadCompleted, timestamp, sessionId, payload };
    case 'install.start':
      return { type: LauncherEventType.LaunchStateChanged, timestamp, sessionId, payload };
    case 'install.complete':
      return { type: LauncherEventType.LaunchStateChanged, timestamp, sessionId, payload };
    case 'error':
      return { type: LauncherEventType.TraceUpdated, timestamp, sessionId, payload };
    default:
      if (typeof event.type === 'string' && Object.values(LauncherEventType).includes(event.type as LauncherEventType)) {
        return { type: event.type as LauncherEventType, timestamp, sessionId, payload };
      }
      return { type: LauncherEventType.TraceUpdated, timestamp, sessionId, payload };
  }
}
