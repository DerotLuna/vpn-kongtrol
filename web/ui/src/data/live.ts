// Kongtrol UI — the daemon's live feed · v0.3.0 · 2026-11-20
//
// One WebSocket per page (GET /api/v1/ws/metrics, internal/api/ws.go), each
// open with a fresh one-time ticket from api/auth.ts (browsers cannot add
// headers to a WebSocket; ADR-0062). The daemon pushes the whole tunnel map
// on every state change and on a 5s
// heartbeat. Each frame is written into the same cache entry the REST
// snapshot uses (keys.tunnels); structural sharing keeps unchanged tunnels
// identical, so only rows that changed re-render.
//
// While the socket is down, liveState is 'down' and useTunnels polls REST
// instead. Reconnects use jittered exponential backoff; a reopened socket
// after an outage also refreshes every other resource (the daemon may have
// restarted with a new config).
//
// The page asks for events (?events=1, ADR-0072): frames are envelopes,
// {"type":"metrics","tunnels":{…}} or {"type":"changed","rev":N} after a
// settings change made anywhere (another tab, the CLI, the tray). A change
// refetches every other resource and announces CHANGED_EVENT so the
// preferences (theme, language) are read again too.
import type { QueryClient } from '@tanstack/react-query'
import { authState, liveTicket } from '../api/auth'
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
  /** One-time WebSocket ticket (api/auth.ts); injected in tests. */
  ticket?: () => Promise<string>
}

export function liveFeedURL(): string {
  const scheme = location.protocol === 'https:' ? 'wss' : 'ws'
  return `${scheme}://${location.host}/api/v1/ws/metrics?events=1`
}

/** Window event fired when the daemon reports a settings change. */
export const CHANGED_EVENT = 'kongtrol:changed'

/** The feed URL with its one-time ticket. */
export function withTicket(url: string, ticket: string): string {
  return `${url}${url.includes('?') ? '&' : '?'}ticket=${encodeURIComponent(ticket)}`
}

export type LiveFrame = { type: 'metrics'; tunnels: TunnelMetrics[] } | { type: 'changed' }

/**
 * Parses one frame: an envelope ({"type":"metrics","tunnels":{…}} or
 * {"type":"changed"}). Null if malformed or of an unknown type.
 */
export function parseFrame(data: string): LiveFrame | null {
  try {
    const msg = JSON.parse(data) as { type?: unknown; tunnels?: Record<string, TunnelMetrics> | null } | null
    if (msg === null || typeof msg !== 'object') return null
    if (msg.type === 'changed') return { type: 'changed' }
    if (msg.type !== 'metrics' || (msg.tunnels !== null && typeof msg.tunnels !== 'object')) return null
    return { type: 'metrics', tunnels: sortTunnels(Object.values(msg.tunnels ?? {})) }
  } catch {
    return null
  }
}

export function startLiveFeed(client: QueryClient, opts: LiveFeedOptions = {}): () => void {
  const WS = opts.WebSocketImpl ?? globalThis.WebSocket
  const delay = opts.delay ?? ((n: number) => backoffDelay(n))
  const ticket = opts.ticket ?? (() => liveTicket())
  if (!WS) {
    liveState.set('down')
    return () => {}
  }
  let socket: WebSocket | null = null
  let closed = false
  let attempt = 0
  let hadOutage = false
  let timer: ReturnType<typeof setTimeout> | undefined

  const retry = () => {
    if (closed) return
    hadOutage = true
    liveState.set('down')
    if (authState.get() === 'ok') timer = setTimeout(open, delay(attempt++))
  }

  const open = () => {
    if (closed || authState.get() === 'unauthorized') return
    liveState.set(attempt === 0 ? 'connecting' : 'down')
    ticket().then(connect, retry)
  }

  const connect = (t: string) => {
    if (closed || authState.get() === 'unauthorized') return
    socket = new WS(withTicket(opts.url ?? liveFeedURL(), t))
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
      const frame = parseFrame(event.data)
      if (frame?.type === 'metrics') client.setQueryData(keys.tunnels, frame.tunnels)
      if (frame?.type === 'changed') {
        void client.invalidateQueries({ predicate: (q) => q.queryKey[0] !== keys.tunnels[0] })
        window.dispatchEvent(new Event(CHANGED_EVENT))
      }
    }
    socket.onclose = retry
  }
  const unsubscribe = authState.subscribe(() => {
    if (authState.get() === 'unauthorized') {
      clearTimeout(timer)
      socket?.close()
      socket = null
      liveState.set('down')
    } else {
      open()
    }
  })
  open()

  return () => {
    closed = true
    unsubscribe()
    clearTimeout(timer)
    socket?.close()
  }
}
