// Kongtrol UI — typed client for the daemon's local API · v0.4.0 · 2026-11-18
//
// Same-origin only; requests carry no token in JS. Authentication (cookie,
// session bootstrap, 401 handling) lives only in api/auth.ts. The daemon
// translates errors when told the language (X-Kongtrol-Lang).
//
// Every request also reports reachability to data/connection.ts: a network
// failure means the daemon that serves this page stopped answering.
// Caching, de-dup, retries and the live feed live in src/data/ (ADR-0037).
import { connection } from '../data/connection'
import { authReady, noteAuthStatus } from './auth'
import type { Lang } from '../i18n/messages'
import type {
  ApiErrorBody,
  DoctorReport,
  HelperStatus,
  Preferences,
  PublicIPStatus,
  PurgeResult,
  SecurityStatus,
  TunnelMetrics,
  UpdateStatus,
  VersionInfo,
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
  await authReady()
  let res: Response
  try {
    res = await fetch(path, { ...init, headers, credentials: 'same-origin' })
  } catch {
    connection.set('offline')
    throw new ApiError('offline', 0)
  }
  connection.set('online')
  return res
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
    const err = (body ?? {}) as ApiErrorBody
    // 401, or 403 without a translated key, comes from the auth middleware
    // (cookie or Origin). A translated 403 is an ordinary failure, e.g. the
    // user declined the OS permission prompt while installing the helper.
    const auth = res.status === 401 || (res.status === 403 && !err.error_key)
    noteAuthStatus(res.status, auth)
    throw new ApiError(auth ? 'unauthorized' : 'server', res.status, err)
  }
  noteAuthStatus(res.status, false)
  return body as T
}

function sendJSON<T>(method: 'PUT' | 'POST', path: string, payload: unknown): Promise<T> {
  return request<T>(path, {
    method,
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
}

const enc = encodeURIComponent

/** One order everywhere (REST and WebSocket), so unchanged rows keep their identity. */
export function sortTunnels(list: TunnelMetrics[]): TunnelMetrics[] {
  return [...list].sort((a, b) => a.Name.localeCompare(b.Name))
}

export const api = {
  tunnels: () => request<TunnelMetrics[] | null>('/api/v1/tunnels').then((l) => sortTunnels(l ?? [])),
  connect: (name: string) => request<unknown>(`/api/v1/tunnels/${enc(name)}/connect`, { method: 'POST' }),
  disconnect: (name: string) => request<unknown>(`/api/v1/tunnels/${enc(name)}/disconnect`, { method: 'POST' }),
  cancelConnect: (name: string) =>
    request<unknown>(`/api/v1/tunnels/${enc(name)}/cancel_connect`, { method: 'POST' }),
  helperStatus: () => request<HelperStatus>('/api/v1/helper/status'),
  /** Shows one OS permission prompt; may take minutes. Empty body by design. */
  installHelper: () => request<HelperStatus>('/api/v1/helper/install', { method: 'POST' }),
  /** Daemon build (version, commit, date); same as `kongtrol version`. */
  version: () => request<VersionInfo>('/api/v1/version'),
  preferences: () => request<Preferences>('/api/v1/preferences'),
  savePreferences: (update: Partial<Preferences>) =>
    sendJSON<Preferences>('PUT', '/api/v1/preferences', update),
  /** Kill switch, DNS guard and public IP state. Reads state only; no external lookup. */
  security: () => request<SecurityStatus>('/api/v1/security/status'),
  setKillSwitch: (enabled: boolean) => sendJSON<unknown>('POST', '/api/v1/security/killswitch', { enabled }),
  setDNSGuard: (enabled: boolean) => sendJSON<unknown>('POST', '/api/v1/security/dnsguard', { enabled }),
  /** Reads state only; no external lookup. */
  publicIP: () => request<PublicIPStatus>('/api/v1/security/status'),
  /** Opt in/out of the external public IP lookup. */
  setPublicIP: (enabled: boolean) => sendJSON<PublicIPStatus>('POST', '/api/v1/security/publicip', { enabled }),
  /** One lookup now (the user's "Revisar"). */
  checkPublicIP: () => request<PublicIPStatus>('/api/v1/security/publicip/check', { method: 'POST' }),
  /** Remembered state only; reading never contacts GitHub. */
  updates: () => request<UpdateStatus>('/api/v1/updates'),
  /** Opt in/out of the update notice (updates.check_enabled). */
  setUpdates: (enabled: boolean) => sendJSON<UpdateStatus>('POST', '/api/v1/updates', { enabled }),
  /** "Borrar todos mis datos": the daemon then restarts into first-run setup. */
  purgeData: () => sendJSON<PurgeResult>('POST', '/api/v1/data/purge', { confirm: true }),
  /** Same checks as `kongtrol doctor`, run by the daemon; takes a few seconds. */
  doctor: () => request<DoctorReport>('/api/v1/doctor'),
  /** kongtrol.yaml as a file (no passwords); errors arrive translated. */
  exportConfig: async (): Promise<Blob> => {
    const res = await send('/api/v1/config/export', {}, 'application/yaml, application/json')
    if (!res.ok) return parse<Blob>(res)
    return res.blob()
  },
}
