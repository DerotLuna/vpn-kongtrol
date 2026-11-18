// Kongtrol UI — the daemon's live feed · v0.1.0 · 2026-11-18
//
// One WebSocket per page (GET /api/v1/ws/metrics, internal/api/ws.go): the
// daemon pushes the whole tunnel map on every state change and on a 5s
// heartbeat. Each frame is written into the same cache entry the REST
// snapshot uses (keys.tunnels); structural sharing keeps unchanged tunnels
// identical, so only rows that changed re-render.
//
// While the socket is down, liveState is 'down' and useTunnels polls REST
// instead. Reconnects use jittered exponential backoff; a reopened socket
// after an outage also refreshes every other resource (the daemon may have
// restarted with a new config).
import type { QueryClient } from '@tanstack/react-query'
import { sortTunnels } from '../api/client'
import type { TunnelMetrics } from '../api/types'
import { backoffDelay } from './backoff'
import { connection, liveState } from './connection'
import { keys } from './keys'

export interface LiveFeedOptions {
  /** Injected in tests. */
  WebSocketImpl?: typeof WebSocket
  url?: string
  delay?: (attempt: number) => number
}

export function liveFeedURL(): string {
  const scheme = location.protocol === 'https:' ? 'wss' : 'ws'
  return `${scheme}://${location.host}/api/v1/ws/metrics`
}

/** Parses one frame: a name-keyed map of TunnelMetrics. Null if malformed. */
export function parseFrame(data: string): TunnelMetrics[] | null {
  try {
    const map = JSON.parse(data) as Record<string, TunnelMetrics> | null
    if (map !== null && typeof map !== 'object') return null
    return sortTunnels(Object.values(map ?? {}))
  } catch {
    return null
  }
}

export function startLiveFeed(client: QueryClient, opts: LiveFeedOptions = {}): () => void {
  const WS = opts.WebSocketImpl ?? globalThis.WebSocket
  const delay = opts.delay ?? ((n: number) => backoffDelay(n))
  if (!WS) {
    liveState.set('down')
    return () => {}
  }
  let socket: WebSocket | null = null
  let closed = false
  let attempt = 0
  let hadOutage = false
  let timer: ReturnType<typeof setTimeout> | undefined

  const open = () => {
    liveState.set(attempt === 0 ? 'connecting' : 'down')
    socket = new WS(opts.url ?? liveFeedURL())
    socket.onopen = () => {
      attempt = 0
      liveState.set('live')
      connection.set('online')
      if (hadOutage) {
        hadOutage = false
        void client.invalidateQueries()
      }
    }
    socket.onmessage = (event: MessageEvent<string>) => {
      const list = parseFrame(event.data)
      if (list) client.setQueryData(keys.tunnels, list)
    }
    socket.onclose = () => {
      if (closed) return
      hadOutage = true
      liveState.set('down')
      timer = setTimeout(open, delay(attempt++))
    }
  }
  open()

  return () => {
    closed = true
    clearTimeout(timer)
    socket?.close()
  }
}
