const BASE = '/api/v1'

export interface ServerRecord {
  id: string
  name: string
  description?: string
  region?: string
  gameVersion?: string
  playerCount: number
  maxPlayers?: number
  status: string
  tags?: string[]
}

export interface AgentState {
  serverId: string
  status: 'online' | 'degraded' | 'offline'
  lastSeen: string
  modCount: number
  version?: string
}

export interface VersionSummary {
  version: number
  modCount: number
  publishedAt: string
}

export interface ModEntry {
  id: string
  name: string
  version: string
  sha256: string
  sizeBytes?: number
}

export interface ManifestDiff {
  serverId: string
  fromVersion: number
  toVersion: number
  added: ModEntry[] | null
  removed: ModEntry[] | null
  updated: ModEntry[] | null
  unchanged: number
}

export async function fetchServers(): Promise<ServerRecord[]> {
  const r = await fetch(`${BASE}/servers`)
  const d = await r.json()
  return d.servers ?? []
}

export async function fetchAgents(): Promise<AgentState[]> {
  const r = await fetch(`${BASE}/agents`)
  const d = await r.json()
  return d.agents ?? []
}

export async function fetchHistory(serverId: string): Promise<VersionSummary[]> {
  const r = await fetch(`${BASE}/manifests/${serverId}/history`)
  if (!r.ok) return []
  const d = await r.json()
  return d.versions ?? []
}

export async function fetchDiff(serverId: string, from: number, to: number): Promise<ManifestDiff | null> {
  const r = await fetch(`${BASE}/manifests/${serverId}/diff?from=${from}&to=${to}`)
  if (!r.ok) return null
  return r.json()
}

export interface PlatformMetrics {
  joinTotal: number
  joinDurationP50: number | null
  blobUploadTotal: number
  blobDownloadTotal: number
  manifestPublishTotal: number
  heartbeatTotal: number
  agentsOnline: number
  agentsDegraded: number
  agentsOffline: number
}

// fetchMetrics parses the Prometheus text exposition format from GET /metrics.
export async function fetchMetrics(): Promise<PlatformMetrics> {
  const r = await fetch('/metrics')
  if (!r.ok) throw new Error('metrics unavailable')
  const text = await r.text()

  function gauge(name: string): number {
    const m = text.match(new RegExp(`^${name}(?:\\{[^}]*\\})? ([\\d.e+\\-]+)`, 'm'))
    return m ? parseFloat(m[1]) : 0
  }
  function sumCounters(name: string): number {
    let total = 0
    const re = new RegExp(`^${name}(?:\\{[^}]*\\})? ([\\d.e+\\-]+)`, 'gm')
    let m: RegExpExecArray | null
    while ((m = re.exec(text)) !== null) total += parseFloat(m[1])
    return total
  }

  return {
    joinTotal:            sumCounters('pz_join_total'),
    joinDurationP50:      null,
    blobUploadTotal:      gauge('pz_blob_upload_total'),
    blobDownloadTotal:    gauge('pz_blob_download_total'),
    manifestPublishTotal: sumCounters('pz_manifest_publish_total'),
    heartbeatTotal:       sumCounters('pz_heartbeat_total'),
    agentsOnline:         gauge('pz_agents_online'),
    agentsDegraded:       gauge('pz_agents_degraded'),
    agentsOffline:        gauge('pz_agents_offline'),
  }
}
