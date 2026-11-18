// Kongtrol UI — is the daemon answering? · v0.1.0 · 2026-11-18
//
// The daemon serves this page, so "offline" here never means "no internet":
// it means Kongtrol stopped (or restarted) while the page stayed open. Every
// request reports here (api/client.ts) and so does the live feed. The app
// shows one banner while offline and refetches everything when it is back.
//
// navigator.onLine is deliberately ignored: the API is 127.0.0.1, reachable
// with or without a network (the data layer uses networkMode 'always').
import { createStore, useStore } from './store'

export type DaemonState = 'unknown' | 'online' | 'offline'

export const connection = createStore<DaemonState>('unknown')

export function useDaemonState(): DaemonState {
  return useStore(connection)
}

/** State of the WebSocket live feed; 'down' means the UI is polling instead. */
export type LiveState = 'connecting' | 'live' | 'down'

export const liveState = createStore<LiveState>('connecting')

export function useLiveState(): LiveState {
  return useStore(liveState)
}
