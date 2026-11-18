// Kongtrol UI — the only module that deals with authentication · v0.2.0 · 2026-11-18
//
// One-time bootstrap (ADR-0039): the tray / CLI open
// `/app/#k=<one-time nonce>` (daemonctl.Client.PanelURL), this module trades
// the nonce for the HttpOnly SameSite=Strict `kongtrol_session` cookie
// (POST /api/v1/session) and the page never sees the file token. The daemon
// no longer sets any cookie when it serves a page. A 404 on the exchange
// means an older daemon.
//
// Rules: nothing else reads the URL fragment, stores credentials or sets
// auth headers; a 401 from any request flips `authState` to
// 'unauthorized' and the shell tells the user to reopen the panel from
// Kongtrol (no silent failure, no retry loop).
import { createStore, useStore } from '../data/store'

export type AuthState = 'ok' | 'unauthorized'

export const authState = createStore<AuthState>('ok')

export function useAuthState(): AuthState {
  return useStore(authState)
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
 * for the session cookie when the page was opened with one. Uses fetch
 * directly (not the API client) to avoid waiting on itself.
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
    (res) => {
      // 404 = daemon without the endpoint yet; the cookie path still applies.
      if (res.status === 401 || res.status === 403) authState.set('unauthorized')
    },
    () => {
      // Daemon unreachable: the connection banner explains it.
    },
  )
  return ready
}

/** Called by the API client for every answer. */
export function noteAuthStatus(status: number, unauthorized: boolean): void {
  if (unauthorized) authState.set('unauthorized')
  else if (status < 400) authState.set('ok')
}
