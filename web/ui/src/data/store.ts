// Kongtrol UI — tiny external store · v0.1.0 · 2026-11-18
//
// For app-wide state that is not server data (daemon reachability, live feed
// state, toasts). Read with React's useSyncExternalStore, which is the
// tearing-free way to subscribe to state outside React
// (https://react.dev/reference/react/useSyncExternalStore).
import { useSyncExternalStore } from 'react'

export interface Store<T> {
  get: () => T
  set: (next: T) => void
  subscribe: (listener: () => void) => () => void
}

export function createStore<T>(initial: T): Store<T> {
  let value = initial
  const listeners = new Set<() => void>()
  return {
    get: () => value,
    set: (next) => {
      if (Object.is(next, value)) return
      value = next
      for (const l of [...listeners]) l()
    },
    subscribe: (listener) => {
      listeners.add(listener)
      return () => listeners.delete(listener)
    },
  }
}

export function useStore<T>(store: Store<T>): T {
  return useSyncExternalStore(store.subscribe, store.get, store.get)
}
