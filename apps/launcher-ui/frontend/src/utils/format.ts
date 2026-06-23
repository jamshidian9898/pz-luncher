export function formatBytes(bytes: number): string {
  if (bytes === 0) return '—';
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

export function formatSeconds(seconds: number): string {
  if (seconds < 60) return `${Math.round(seconds)}s`;
  if (seconds < 3600) return `${Math.round(seconds / 60)}m`;
  return `${Math.round(seconds / 3600)}h`;
}

export function daysSince(dateStr: string): number {
  try {
    const date = new Date(dateStr);
    const now = new Date();
    const ms = now.getTime() - date.getTime();
    return Math.floor(ms / (1000 * 60 * 60 * 24));
  } catch {
    return 0;
  }
}
