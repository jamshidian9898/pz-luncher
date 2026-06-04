import { useEffect, useRef } from 'react';
import { Settings } from '../types';
import { eventsApi, launcherApi, settingsApi } from '../wails';
import { LauncherEvent } from '../interfaces/LauncherEvent';

// RFC 0022: UI Events Hook
export function useLauncherEvents(onEvent: (event: LauncherEvent) => void) {
  const callbackRef = useRef(onEvent);
  
  useEffect(() => {
    callbackRef.current = onEvent;
  }, [onEvent]);
  
  useEffect(() => {
    const unsubscribe = eventsApi.onLauncherEvent((event: LauncherEvent) => {
      callbackRef.current(event);
    });
    return () => { unsubscribe(); };
  }, []);
}

// Hook for fetching server list
export function useServerList() {
  const fetchServers = async () => {
    try {
      return await launcherApi.getServerList();
    } catch (err) {
      console.error('Failed to fetch servers:', err);
      return [];
    }
  };
  return { fetchServers };
}

// Hook for server details
export function useServerDetails() {
  const fetchDetails = async (serverId: string) => {
    try {
      return await launcherApi.getServerDetails(serverId);
    } catch (err) {
      console.error('Failed to fetch server details:', err);
      return null;
    }
  };
  return { fetchDetails };
}

// Hook for settings
export function useSettings() {
  const getSettings = async () => {
    try {
      return await settingsApi.getSettings();
    } catch (err) {
      console.error('Failed to get settings:', err);
      return null;
    }
  };

  const saveSettings = async (settings: Settings) => {
    try {
      await settingsApi.saveSettings(settings);
      return true;
    } catch (err) {
      console.error('Failed to save settings:', err);
      return false;
    }
  };

  return { getSettings, saveSettings };
}
