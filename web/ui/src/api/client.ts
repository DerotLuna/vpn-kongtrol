// Kongtrol UI — typed client for the daemon's local API · v0.3.0 · 2026-11-18
//
// Same-origin only: the daemon serves this UI and sets the HttpOnly
// `kongtrol_control` cookie, so requests carry no token in JS. The daemon
// translates errors when told the language (X-Kongtrol-Lang).
import type { Lang } from '../i18n/messages'
import type {
  ApiErrorBody,
  DoctorReport,
  HelperStatus,
  Preferences,
  PublicIPStatus,
  TunnelMetrics,
} from './types'

export type ApiErrorKind = 'offline' | 'unauthorized' | 'server'

export class ApiError extends Error {
  readonly kind: ApiErrorKind
  readonly status: number
  /** Already translated by the daemon when it sent one. */
  readonly hint: string
  readonly key: string

  constructor(kind: ApiErrorKind, status: number, body: ApiErrorBody = {}) {
    super(body.error ?? `HTTP ${status}`)
    this.name = 'ApiError'
    this.kind = kind
    this.status = status
    this.hint = body.hint ?? ''
    this.key = body.error_key ?? body.code ?? ''
  }
}

let currentLang: Lang = 'es'

/** Sets the language the daemon uses for translated errors. */
export function setApiLang(lang: Lang): void {
  currentLang = lang
}

async function send(path: string, init: RequestInit, accept: string): Promise<Response> {
  const headers = new Headers(init.headers)
  headers.set('X-Kongtrol-Lang', currentLang)
  headers.set('Accept', accept)
  try {
    return await fetch(path, { ...init, headers, credentials: 'same-origin' })
  } catch {
    throw new ApiError('offline', 0)
  }
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const res = await send(path, init, 'application/json')
  return parse<T>(res)
}

async function parse<T>(res: Response): Promise<T> {
  const text = await res.text()
  let body: unknown = null
  if (text) {
    try {
      body = JSON.parse(text)
    } catch {
      body = { error: text.trim() }
    }
  }
  if (!res.ok) {
    const kind: ApiErrorKind = res.status === 401 || res.status === 403 ? 'unauthorized' : 'server'
    throw new ApiError(kind, res.status, (body ?? {}) as ApiErrorBody)
  }
  return body as T
}

function sendJSON<T>(method: 'PUT' | 'POST', path: string, payload: unknown): Promise<T> {
  return request<T>(path, {
    method,
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
}

export const api = {
  tunnels: () => request<TunnelMetrics[]>('/api/v1/tunnels'),
  helperStatus: () => request<HelperStatus>('/api/v1/helper/status'),
  /** Shows one OS permission prompt; may take minutes. Empty body by design. */
  installHelper: () => request<HelperStatus>('/api/v1/helper/install', { method: 'POST' }),
  preferences: () => request<Preferences>('/api/v1/preferences'),
  savePreferences: (update: Partial<Preferences>) =>
    sendJSON<Preferences>('PUT', '/api/v1/preferences', update),
  /** Reads state only; no external lookup. */
  publicIP: () => request<PublicIPStatus>('/api/v1/security/status'),
  /** Opt in/out of the external public IP lookup. */
  setPublicIP: (enabled: boolean) => sendJSON<PublicIPStatus>('POST', '/api/v1/security/publicip', { enabled }),
  /** One lookup now (the user's "Revisar"). */
  checkPublicIP: () => request<PublicIPStatus>('/api/v1/security/publicip/check', { method: 'POST' }),
  /** Same checks as `kongtrol doctor`, run by the daemon; takes a few seconds. */
  doctor: () => request<DoctorReport>('/api/v1/doctor'),
  /** kongtrol.yaml as a file (no passwords); errors arrive translated. */
  exportConfig: async (): Promise<Blob> => {
    const res = await send('/api/v1/config/export', {}, 'application/yaml, application/json')
    if (!res.ok) return parse<Blob>(res)
    return res.blob()
  },
}

/**
 * Subscribes to the live tunnel feed (/api/v1/ws/metrics). Reconnects with
 * backoff; calls onDown while disconnected so callers can fall back to
 * polling. Returns a function that closes the feed.
 */
export function subscribeTunnels(
  onSnapshot: (tunnels: TunnelMetrics[]) => void,
  onDown: () => void,
): () => void {
  let socket: WebSocket | null = null
  let closed = false
  let attempt = 0
  let timer: ReturnType<typeof setTimeout> | undefined

  const connect = () => {
    const scheme = location.protocol === 'https:' ? 'wss' : 'ws'
    socket = new WebSocket(`${scheme}://${location.host}/api/v1/ws/metrics`)
    socket.onopen = () => {
      attempt = 0
    }
    socket.onmessage = (event: MessageEvent<string>) => {
      try {
        const map = JSON.parse(event.data) as Record<string, TunnelMetrics> | null
        onSnapshot(Object.values(map ?? {}))
      } catch {
        // Ignore a malformed frame; the next snapshot replaces it.
      }
    }
    socket.onclose = () => {
      if (closed) return
      onDown()
      const delay = Math.min(30_000, 1_000 * 2 ** attempt++)
      timer = setTimeout(connect, delay)
    }
  }
  connect()

  return () => {
    closed = true
    clearTimeout(timer)
    socket?.close()
  }
}
