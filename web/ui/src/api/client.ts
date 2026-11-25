// Kongtrol UI — typed client for the daemon's local API · v0.9.0 · 2026-11-25
//
// Same-origin only; never the file token. Authentication (cookie, the page's
// session header, session bootstrap, 401 handling) lives only in api/auth.ts. The daemon
// translates errors when told the language (X-Kongtrol-Lang).
//
// Every request also reports reachability to data/connection.ts: a network
// failure means the daemon that serves this page stopped answering.
// Caching, de-dup, retries and the live feed live in src/data/ (ADR-0038).
import { connection } from '../data/connection'
import { authHeaders, authReady, authState, denySession } from './auth'
import type { Lang } from '../i18n/messages'
import type {
  ApiErrorBody,
  DoctorReport,
  Group,
  HelperStatus,
  Preferences,
  PublicIPStatus,
  PurgeResult,
  SecurityStatus,
  TunnelActionResult,
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
  /** The whole JSON body, for answers with extra fields (e.g. 409 restart_required). */
  readonly body: ApiErrorBody & { [key: string]: unknown }

  constructor(kind: ApiErrorKind, status: number, body: ApiErrorBody = {}) {
    super(body.error ?? `HTTP ${status}`)
    this.name = 'ApiError'
    this.kind = kind
    this.status = status
    this.hint = body.hint ?? ''
    this.key = body.error_key ?? body.code ?? ''
    this.body = { ...body }
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
  if (authState.get() === 'unauthorized') throw new ApiError('unauthorized', 401)
  for (const [name, value] of Object.entries(authHeaders())) headers.set(name, value)
  let res: Response
  try {
    res = await fetch(path, { ...init, headers, credentials: 'same-origin' })
  } catch (err) {
    // Aborted by the caller (e.g. the Add sheet closed during a native
    // pick): not a lost daemon. The caller sees the AbortError and
    // treats it as cancelled; see isAbort.
    if (init.signal?.aborted) throw err
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
    if (auth) denySession()
    throw new ApiError(auth ? 'unauthorized' : 'server', res.status, err)
  }
  return body as T
}

function withJSON(method: string, payload: unknown): RequestInit {
  if (payload === undefined) return { method }
  return { method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload) }
}

/**
 * The one way to call the daemon's JSON API. Every call carries the same
 * language header and auth, reports reachability to data/connection.ts and
 * throws ApiError (offline / unauthorized / server, with the daemon's
 * translated `hint`) on failure. Data modules (src/data/<domain>.ts) keep
 * their endpoints next to their hooks and call these; pages never fetch.
 * `post` without a body sends none (several actions take no payload).
 */
export const http = {
  get: <T>(path: string) => request<T>(path),
  /** GET that also returns the response headers (e.g. X-Kongtrol-Audit-More paging). */
  getWithHeaders: async <T>(path: string): Promise<{ data: T; headers: Headers }> => {
    const res = await send(path, {}, 'application/json')
    return { data: await parse<T>(res), headers: res.headers }
  },
  /** `signal` aborts the request; the daemon's request context is cancelled with it. */
  post: <T>(path: string, payload?: unknown, opts: { signal?: AbortSignal } = {}) =>
    request<T>(path, opts.signal ? { ...withJSON('POST', payload), signal: opts.signal } : withJSON('POST', payload)),
  put: <T>(path: string, payload: unknown) => request<T>(path, withJSON('PUT', payload)),
  delete: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
}

/** A request the caller aborted (AbortController): a cancel, never an error to show. */
export function isAbort(err: unknown): boolean {
  return typeof err === 'object' && err !== null && (err as { name?: unknown }).name === 'AbortError'
}

const enc = encodeURIComponent

/** One order everywhere (REST and WebSocket), so unchanged rows keep their identity. */
export function sortTunnels(list: TunnelMetrics[]): TunnelMetrics[] {
  return [...list].sort((a, b) => a.Name.localeCompare(b.Name))
}

export const api = {
  tunnels: () => http.get<TunnelMetrics[] | null>('/api/v1/tunnels').then((l) => sortTunnels(l ?? [])),
  /** Accepted (202) at once; the tunnel feed reports when it is up. */
  connect: (name: string) => http.post<TunnelActionResult>(`/api/v1/tunnels/${enc(name)}/connect`),
  /** 202 waiting_user: the provider app must be disconnected by hand. */
  disconnect: (name: string) =>
    http.post<TunnelActionResult>(`/api/v1/tunnels/${enc(name)}/disconnect`),
  cancelConnect: (name: string) =>
    http.post<unknown>(`/api/v1/tunnels/${enc(name)}/cancel_connect`),
  helperStatus: () => http.get<HelperStatus>('/api/v1/helper/status'),
  /** Shows one OS permission prompt; may take minutes. Empty body by design. */
  installHelper: () => http.post<HelperStatus>('/api/v1/helper/install'),
  /** Daemon build (version, commit, date); same as `kongtrol version`. */
  version: () => http.get<VersionInfo>('/api/v1/version'),
  preferences: () => http.get<Preferences>('/api/v1/preferences'),
  savePreferences: (update: Partial<Preferences>) =>
    http.put<Preferences>('/api/v1/preferences', update),
  /** Kill switch, DNS guard and public IP state. Reads state only; no external lookup. */
  security: () => http.get<SecurityStatus>('/api/v1/security/status'),
  setKillSwitch: (enabled: boolean) => http.post<unknown>('/api/v1/security/killswitch', { enabled }),
  setDNSGuard: (enabled: boolean) => http.post<unknown>('/api/v1/security/dnsguard', { enabled }),
  /** One lookup now (the user's "Revisar"). */
  checkPublicIP: () => http.post<PublicIPStatus>('/api/v1/security/publicip/check'),
  /** Remembered state only; reading never contacts GitHub. */
  updates: () => http.get<UpdateStatus>('/api/v1/updates'),
  /** Opt in/out of the update notice (updates.check_enabled). */
  setUpdates: (enabled: boolean) => http.post<UpdateStatus>('/api/v1/updates', { enabled }),
  /** "Borrar todos mis datos": the daemon then restarts into first-run setup. */
  purgeData: () => http.post<PurgeResult>('/api/v1/data/purge', { confirm: true }),
  groups: () => http.get<Group[]>('/api/v1/groups'),
  connectGroup: (name: string) => http.post<unknown>(`/api/v1/groups/${enc(name)}/connect`),
  disconnectGroup: (name: string) => http.post<unknown>(`/api/v1/groups/${enc(name)}/disconnect`),
  /** Same checks as `kongtrol doctor`, run by the daemon; takes a few seconds. */
  doctor: () => http.get<DoctorReport>('/api/v1/doctor'),
  /** kongtrol.yaml as a file (no passwords); errors arrive translated. */
  exportConfig: async (): Promise<Blob> => {
    const res = await send('/api/v1/config/export', {}, 'application/yaml, application/json')
    if (!res.ok) return parse<Blob>(res)
    return res.blob()
  },
}
