// Kongtrol UI — a tunnel action stays "in progress" until the daemon says it ended · v1.0.0 · 2026-11-19
//
// The daemon accepts a connect at once (202 "connecting") and does the work
// in the background; a disconnect returns before the live feed shows the
// tunnel down. Showing the spinner only while the HTTP request was pending
// made the row fall back to "Conectar" while the tunnel was still coming up
// (owner feedback 2026-11-19). Now the action stays pending until the
// tunnel's live state (WebSocket frames or a re-read, both land in the
// keys.tunnels cache) reaches an end state for that action, or a timeout
// with the "what to do" line. Shared by Inicio, Grupos and the tray flyout.
import type { QueryClient } from '@tanstack/react-query'
import type { TunnelMetrics } from '../api/types'
import { keys } from './keys'
import { createStore, useStore } from './store'

export type TunnelAction = 'connect' | 'cancel' | 'disconnect' | 'reload'

/** Which action runs on which tunnel right now (rows show it in place). */
export const pendingActions = createStore<Readonly<Record<string, TunnelAction>>>({})

export function setPending(name: string, action: TunnelAction | null): void {
  const next = { ...pendingActions.get() }
  if (action) next[name] = action
  else delete next[name]
  pendingActions.set(next)
}

export function usePendingAction(name: string): TunnelAction | undefined {
  return useStore(pendingActions)[name]
}

/** True while any of `names` has an action in progress (group rows). */
export function usePendingAny(names: readonly string[]): boolean {
  const all = useStore(pendingActions)
  return names.some((n) => all[n] !== undefined)
}

/** How long an accepted action may stay in progress before we say so. */
export const SETTLE_TIMEOUT_MS: Record<TunnelAction, number> = {
  // A provider app may take a while to start; waiting for the user ends
  // the wait earlier (attach phase waiting_user).
  connect: 90_000,
  cancel: 30_000,
  disconnect: 45_000,
  reload: 45_000,
}

/** What the tunnel looked like when the action was accepted, plus what was seen since. */
export interface Baseline {
  connectedAt: string
  /** The daemon reported the tunnel in transition (not the start state) at least once. */
  sawProgress: boolean
}

/**
 * True when `tn` (a state the daemon reported) ends `action`:
 * - connect: connected; an error (state or last_error); the provider app is
 *   waiting for the user (the row then says what to do); or back to
 *   disconnected after it was seen connecting (cancelled elsewhere).
 * - cancel / disconnect: disconnected or error; disconnect also ends when a
 *   provider app waits for the user to disconnect in the app.
 * - reload: connected again with a new ConnectedAt, an error, or a drop
 *   after it was seen restarting.
 * A missing tunnel (deleted, renamed) ends anything. Pure, for tests.
 */
export function transitionSettled(tn: TunnelMetrics | undefined, action: TunnelAction, base: Baseline): boolean {
  if (!tn) return true
  const failed = tn.Status === 'error' || (tn.Status !== 'connected' && !!(tn.last_error || tn.last_error_key))
  switch (action) {
    case 'connect':
      return (
        tn.Status === 'connected' ||
        failed ||
        tn.attach?.phase === 'waiting_user' ||
        (tn.Status === 'disconnected' && base.sawProgress)
      )
    case 'cancel':
      return tn.Status === 'disconnected' || tn.Status === 'error'
    case 'disconnect':
      return tn.Status === 'disconnected' || tn.Status === 'error' || tn.attach?.phase === 'waiting_disconnect'
    case 'reload':
      return (
        failed ||
        (tn.Status === 'connected' && (tn.ConnectedAt !== base.connectedAt || base.sawProgress)) ||
        (tn.Status === 'disconnected' && base.sawProgress)
      )
  }
}

/** Whether `tn` shows the action moving (so a later "disconnected" is an end, not a stale frame). */
function inProgress(tn: TunnelMetrics, action: TunnelAction): boolean {
  if (action === 'reload') return tn.Status !== 'connected'
  return tn.Status === 'connecting'
}

interface Watch {
  action: TunnelAction
  stop: () => void
}
const watches = new Map<string, Watch>()

export interface WatchOptions {
  /** Called once if the action has not ended after SETTLE_TIMEOUT_MS. */
  onTimeout?: (name: string, action: TunnelAction) => void
  /** ConnectedAt when the user pressed (reload); default: from the cache now. */
  connectedAt?: string
}

/**
 * Keeps `name` pending with `action` until the daemon reports an end state
 * (transitionSettled) or the timeout. A newer action on the same tunnel
 * (Cancelar during a connect) replaces the watch.
 */
export function watchTransition(client: QueryClient, name: string, action: TunnelAction, opts: WatchOptions = {}): void {
  watches.get(name)?.stop()
  setPending(name, action)
  const now = client.getQueryData<TunnelMetrics[]>(keys.tunnels)?.find((tn) => tn.Name === name)
  const base: Baseline = { connectedAt: opts.connectedAt ?? now?.ConnectedAt ?? '', sawProgress: false }

  const finish = () => {
    stop()
    if (pendingActions.get()[name] === action) setPending(name, null)
  }
  const unsubscribe = client.getQueryCache().subscribe((event) => {
    if (event.type !== 'updated' || event.action.type !== 'success') return
    const key = event.query.queryKey
    if (key.length !== 1 || key[0] !== keys.tunnels[0]) return
    const list = event.action.data
    if (!Array.isArray(list)) return
    const tn = (list as TunnelMetrics[]).find((x) => x.Name === name)
    if (transitionSettled(tn, action, base)) finish()
    else if (tn && inProgress(tn, action)) base.sawProgress = true
  })
  const timer = setTimeout(() => {
    finish()
    opts.onTimeout?.(name, action)
  }, SETTLE_TIMEOUT_MS[action])
  const stop = () => {
    unsubscribe()
    clearTimeout(timer)
    if (watches.get(name)?.stop === stop) watches.delete(name)
  }
  watches.set(name, { action, stop })
}

/** Stops watching (the request failed: the caller rolls back). */
export function endTransition(name: string): void {
  watches.get(name)?.stop()
  setPending(name, null)
}

/** Test helper: forget every watch and pending mark. */
export function resetTransitions(): void {
  for (const w of [...watches.values()]) w.stop()
  watches.clear()
  pendingActions.set({})
}
