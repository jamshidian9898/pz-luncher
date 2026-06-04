import { useEffect } from 'react';
import { eventsApi } from '../wails';
import { dispatchLauncherEvent } from '../event/LauncherEventDispatcher';

/**
 * Phase 2.1: Real Event Integration
 *
 * Connects launcher events to Zustand stores
 * RFC 0022: UI Events & Progress Streaming
 */
export function useLauncherEvents() {
  useEffect(() => {
    const unsubscribe = eventsApi.onLauncherEvent((event) => {
      console.log('[LauncherEvents] Received:', event.type, event);
      dispatchLauncherEvent(event);
    });

    return () => {
      unsubscribe();
    };
  }, []);
}
