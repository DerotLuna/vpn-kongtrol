// Kongtrol UI — Tráfico › Recientes: destinations this computer connected to lately · v1.0.0 · 2026-11-19
//
// GET /api/v1/connections/recent?limit=n (internal/api/handlers_connections.go,
// ADR-0054): kept only in the daemon's memory, on by default with a switch
// (settings `recent_connections_enabled`, Ajustes › Privacidad), never
// logged. `supported: false` where Kongtrol cannot list connections without
// admin rights (macOS today). An older daemon without the endpoint (404)
// answers null: the card is not shown.
import { useQuery } from '@tanstack/react-query'
import { ApiError, http } from '../api/client'

/** internal/monitor/recent.go `RecentConnection`. */
export interface RecentConnection {
  /** The name it was resolved from, "" when unknown (show remote_ip). */
  host?: string
  remote_ip: string
  port: number
  /** The program's file name, "" when the OS does not tell a standard user. */
  process?: string
  /** The Kongtrol profile whose tunnel it left through; "" = direct. */
  via?: string
  interface?: string
  first_seen: string
  last_seen: string
  count: number
}

/** handlers_connections.go `RecentConnectionsResponse`. */
export interface RecentConnections {
  supported: boolean
  enabled: boolean
  max: number
  ttl_seconds: number
  connections: RecentConnection[] | null
}

/** Same cadence as the daemon's check (monitor.RecentEvery = 15 s). */
export const RECENT_POLL_MS = 15_000

export const recentKeys = { recent: (limit: number) => ['connections', 'recent', limit] as const }

export function fetchRecent(limit: number): Promise<RecentConnections | null> {
  return http.get<RecentConnections>(`/api/v1/connections/recent?limit=${limit}`).then(
    (r) => ({ ...r, connections: r.connections ?? [] }),
    (err: unknown) => {
      if (err instanceof ApiError && err.status === 404) return null
      throw err
    },
  )
}

export function useRecentConnections(limit = 50) {
  return useQuery({
    queryKey: recentKeys.recent(limit),
    queryFn: () => fetchRecent(limit),
    refetchInterval: (q) => (q.state.data?.enabled && q.state.data.supported ? RECENT_POLL_MS : false),
  })
}

/** What the list shows as the destination: the name when known, else the address. */
export function destinationOf(c: RecentConnection): string {
  return c.host || c.remote_ip
}
