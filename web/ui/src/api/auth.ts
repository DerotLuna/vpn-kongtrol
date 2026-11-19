// Kongtrol UI — the only module that deals with authentication · v0.3.0 · 2026-11-19
//
// One-time bootstrap (ADR-0039, ADR-0062): the tray / CLI open
// `/#k=<one-time nonce>` (daemonctl.Client.PanelURL), this module trades
// the nonce (POST /api/v1/session) for the HttpOnly SameSite=Strict
// `kongtrol_session` cookie plus a per-page session token in the answer.
// The token is kept in sessionStorage (this origin only: scheme, host and
// port; it survives reloads of this tab, not other tabs) and sent as
// X-Kongtrol-Session on every request: browsers send the cookie to every
// port of 127.0.0.1, so the cookie alone is not accepted. The WebSocket
// cannot carry headers, so it gets a one-time ticket (liveTicket). The
// page never sees the file token.
//
// Rules: nothing else reads the URL fragment, stores credentials or sets
// auth headers; a 401 from any request flips `authState` to
// 'unauthorized' and the shell tells the user to reopen the panel from
// Kongtrol (no silent failure, no retry loop). A tab opened without a
// nonce has no token and shows that same guidance.
import { createStore, useStore } from '../data/store'

export type AuthState = 'ok' | 'unauthorized'

export const authState = createStore<AuthState>('ok')

export function useAuthState(): AuthState {
  return useStore(authState)
}

const SESSION_HEADER = 'X-Kongtrol-Session'
const STORAGE_KEY = 'kongtrol.session'

let sessionToken = ''

function loadToken(): string {
  if (sessionToken) return sessionToken
  try {
    sessionToken = sessionStorage.getItem(STORAGE_KEY) ?? ''
  } catch {
    // Storage blocked: the token lives in memory until the tab reloads.
  }
  return sessionToken
}

function saveToken(token: string): void {
  sessionToken = token
  try {
    if (token) sessionStorage.setItem(STORAGE_KEY, token)
    else sessionStorage.removeItem(STORAGE_KEY)
  } catch {
    // Storage blocked: memory only.
  }
}

/** Test seam: forget the page token (memory and sessionStorage). */
export function resetSessionForTests(): void {
  saveToken('')
}

/** Headers that authenticate a request of this page (empty before sign-in). */
export function authHeaders(): Record<string, string> {
  const token = loadToken()
  return token ? { [SESSION_HEADER]: token } : {}
}

/** Reads `#k=<nonce>` and removes it from the address bar and history at once. */
export function takeBootstrapNonce(loc: Location = location, hist: History = history): string {
  const params = new URLSearchParams(loc.hash.replace(/^#/, ''))
  const nonce = params.get('k') ?? ''
  if (!nonce) return ''
  params.delete('k')
  const rest = params.toString()
  hist.replaceState(hist.state, '', `${loc.pathname}${loc.search}${rest ? `#${rest}` : ''}`)
  return nonce
}

let ready: Promise<void> = Promise.resolve()

/** Every API request waits for this (api/client.ts), so none races the session exchange. */
export function authReady(): Promise<void> {
  return ready
}

/**
 * Call once at boot, before the first request. Exchanges a bootstrap nonce
 * for the session cookie and page token when the page was opened with one.
 * Uses fetch directly (not the API client) to avoid waiting on itself.
 */
export function startAuth(fetchImpl: typeof fetch = (...a) => fetch(...a)): Promise<void> {
  const nonce = takeBootstrapNonce()
  if (!nonce) return ready
  ready = fetchImpl('/api/v1/session', {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ nonce }),
  }).then(
    async (res) => {
      if (res.status === 401 || res.status === 403) {
        authState.set('unauthorized')
        return
      }
      if (!res.ok) return
      try {
        const body = (await res.json()) as { session_token?: unknown }
        if (typeof body.session_token === 'string' && body.session_token) saveToken(body.session_token)
      } catch {
        // No token in the answer: requests get 401 and the shell explains.
      }
    },
    () => {
      // Daemon unreachable: the connection banner explains it.
    },
  )
  return ready
}

/**
 * A one-time ticket (valid 30 s) for the live feed's WebSocket, which
 * cannot send X-Kongtrol-Session. Throws when there is none; a 401 also
 * marks the page unauthorized.
 */
export async function liveTicket(fetchImpl: typeof fetch = (...a) => fetch(...a)): Promise<string> {
  await ready
  const res = await fetchImpl('/api/v1/ws-ticket', {
    method: 'POST',
    credentials: 'same-origin',
    headers: authHeaders(),
  })
  if (res.status === 401 || res.status === 403) {
    authState.set('unauthorized')
    throw new Error('unauthorized')
  }
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  const body = (await res.json()) as { ticket?: unknown }
  if (typeof body.ticket !== 'string' || !body.ticket) throw new Error('no ticket')
  return body.ticket
}

/** Called by the API client for every answer. */
export function noteAuthStatus(status: number, unauthorized: boolean): void {
  if (unauthorized) authState.set('unauthorized')
  else if (status < 400) authState.set('ok')
}
