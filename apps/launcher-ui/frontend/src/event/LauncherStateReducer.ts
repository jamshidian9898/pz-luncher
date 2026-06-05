import { LauncherEvent, LauncherEventType } from '../interfaces/LauncherEvent';
import { SessionStatus } from '../types';
import { LaunchState } from '../stores/session.store';
import { TraceNode } from '../stores/trace.store';
import { ServerInfo } from '../types';

export type TraceNodePayload = Omit<TraceNode, 'id' | 'timestamp'>;

export interface DownloadsEventPatch {
  sessionUpdate?: SessionStatus;
  completeSessionId?: string;
  failSession?: { sessionId: string; error: string };
}

export interface TraceEventPatch {
  addEvents?: TraceNodePayload[];
  updateEventProgress?: { packageId: string; progress: number; speed?: number };
  completeNode?: { packageId: string };
  activeTrace?: string | null;
}

export interface SessionEventPatch {
  currentSessionId?: string | null;
  launchState?: LaunchState;
  currentServer?: ServerInfo | null;
  lastError?: string | null;
  resetSession?: boolean;
}

export interface ServersEventPatch {
  joining?: boolean;
}

export interface LauncherEventPatch {
  downloads?: DownloadsEventPatch;
  trace?: TraceEventPatch;
  session?: SessionEventPatch;
  servers?: ServersEventPatch;
}

function createSessionStatus(sessionId: string): SessionStatus {
  return {
    sessionId,
    state: 'resolving',
    progress: 0,
    currentMod: 'Starting session...',
    errors: [],
  };
}

export function reduceLauncherEvent(
  event: LauncherEvent,
  currentSession: SessionStatus | undefined
): LauncherEventPatch {
  const session = currentSession ?? createSessionStatus(event.sessionId);
  const packageId = event.payload?.packageId ?? 'unknown';
  const progress = event.payload?.progress;

  switch (event.type) {
    case LauncherEventType.SessionStart:
      return {
        downloads: {
          sessionUpdate: createSessionStatus(event.sessionId),
        },
        session: {
          currentSessionId: event.sessionId,
          launchState: 'resolving',
        },
        trace: {
          activeTrace: event.sessionId,
          addEvents: [
            {
              type: 'resolve',
              modId: 'session',
              modName: 'Session started',
              state: 'resolving',
            },
          ],
        },
      };

    case LauncherEventType.ModResolveStart:
      return {
        downloads: {
          sessionUpdate: {
            ...session,
            state: 'resolving',
            progress: 0,
            currentMod: 'Resolving mods...',
          },
        },
        session: {
          launchState: 'resolving',
        },
        trace: {
          addEvents: [
            {
              type: 'resolve',
              modId: packageId,
              modName: 'Mod resolution started',
              state: 'resolving',
            },
          ],
        },
      };

    case LauncherEventType.ModResolveComplete:
      return {
        downloads: {
          sessionUpdate: {
            ...session,
            state: session.state === 'resolving' ? 'downloading' : session.state,
          },
        },
        session: {
          launchState: 'downloading',
        },
        trace: {
          addEvents: [
            {
              type: 'complete',
              modId: packageId,
              modName: 'Mod resolution complete',
              state: 'complete',
            },
          ],
        },
      };

    case LauncherEventType.DownloadStart:
      return {
        downloads: {
          sessionUpdate: {
            ...session,
            state: 'downloading',
            currentMod: packageId,
          },
        },
        session: {
          launchState: 'downloading',
        },
        trace: {
          addEvents: [
            {
              type: 'download',
              modId: packageId,
              modName: packageId,
              state: 'downloading',
              progress: progress ?? { current: 0, total: 100, percent: 0 },
            },
          ],
        },
      };

    case LauncherEventType.DownloadProgress:
      if (!progress) {
        return {};
      }

      return {
        downloads: {
          sessionUpdate: {
            ...session,
            state: 'downloading',
            progress: progress.percent,
            currentMod: packageId === 'unknown' ? session.currentMod : packageId,
            downloadSpeed: progress.speed,
            eta: progress.eta,
          },
        },
        trace: {
          updateEventProgress: {
            packageId,
            progress: progress.percent,
            speed: progress.speed,
          },
        },
      };

    case LauncherEventType.DownloadComplete:
      return {
        downloads: {
          sessionUpdate: {
            ...session,
            progress: progress?.percent ?? session.progress,
            state: 'downloading',
          },
        },
        trace: {
          completeNode: { packageId },
          addEvents: [
            {
              type: 'verify',
              modId: packageId,
              modName: packageId,
              state: 'complete',
            },
          ],
        },
      };

    case LauncherEventType.InstallStart:
      return {
        downloads: {
          sessionUpdate: {
            ...session,
            state: 'installing',
            currentMod: packageId,
          },
        },
        session: {
          launchState: 'installing',
        },
        trace: {
          addEvents: [
            {
              type: 'install',
              modId: packageId,
              modName: packageId,
              state: 'installing',
            },
          ],
        },
      };

    case LauncherEventType.InstallComplete:
      return {
        downloads: {
          sessionUpdate: {
            ...session,
            state: 'installing',
            progress: progress?.percent ?? session.progress,
          },
        },
        session: {
          launchState: 'verifying',
        },
        trace: {
          addEvents: [
            {
              type: 'complete',
              modId: packageId,
              modName: packageId,
              state: 'complete',
            },
          ],
        },
      };

    case LauncherEventType.SessionComplete:
      return {
        downloads: {
          completeSessionId: event.sessionId,
        },
        session: {
          launchState: 'complete',
        },
        trace: {
          addEvents: [
            {
              type: 'complete',
              modId: 'session',
              modName: 'Session completed',
              state: 'complete',
            },
          ],
        },
        servers: {
          joining: false,
        },
      };

    case LauncherEventType.Error:
      return {
        downloads: event.payload?.error
          ? { failSession: { sessionId: event.sessionId, error: event.payload.error } }
          : undefined,
        session: {
          launchState: 'error',
          lastError: event.payload?.error ?? 'Unknown error',
        },
        trace: {
          addEvents: [
            {
              type: 'error',
              modId: packageId,
              modName: packageId,
              state: 'error',
              error: event.payload?.error,
            },
          ],
        },
      };

    case LauncherEventType.LaunchStateChanged:
      return {
        session: {
          launchState: event.payload?.metadata?.state as LaunchState | undefined,
        },
      };

    default:
      return {
        trace: {
          addEvents: [
            {
              type: 'verify',
              modId: packageId,
              modName: `Unhandled event ${event.type}`,
              state: 'complete',
            },
          ],
        },
      };
  }
}

import { PatchSchemaRegistry } from './PatchSchemaRegistry';

export function validateLauncherEventPatch(patch: LauncherEventPatch): string[] {
  const errors: string[] = [];
  if (typeof patch !== 'object' || patch === null || Array.isArray(patch)) {
    return ['LauncherEventPatch must be an object'];
  }

  // Validate each domain
  if ((patch as any).downloads !== undefined) {
    errors.push(...PatchSchemaRegistry.validateAgainstSchema('downloads', (patch as any).downloads));
  }

  if ((patch as any).trace !== undefined) {
    errors.push(...PatchSchemaRegistry.validateAgainstSchema('trace', (patch as any).trace));
  }

  if ((patch as any).session !== undefined) {
    errors.push(...PatchSchemaRegistry.validateAgainstSchema('session', (patch as any).session));
  }

  if ((patch as any).servers !== undefined) {
    errors.push(...PatchSchemaRegistry.validateAgainstSchema('servers', (patch as any).servers));
  }

  return errors;
}
