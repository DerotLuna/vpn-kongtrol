import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterEach, beforeEach, vi } from 'vitest'
import { authState } from '../api/auth'
import { connection, liveState } from '../data/connection'
import { toasts } from '../data/toast'

// Start every test on Inicio. Vitest's BASE_URL is "/" (the build's is
// "/app/"); the router's basename follows it either way.
beforeEach(() => {
  window.history.replaceState(null, '', import.meta.env.BASE_URL)
})

afterEach(() => {
  toasts.clear()
  authState.set('ok')
  connection.set('unknown')
  liveState.set('connecting')
  cleanup()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  localStorage.clear()
})

// jsdom has no matchMedia; default to a light system scheme.
if (!window.matchMedia) {
  window.matchMedia = (query: string) =>
    ({
      matches: false,
      media: query,
      onchange: null,
      addEventListener: () => {},
      removeEventListener: () => {},
      addListener: () => {},
      removeListener: () => {},
      dispatchEvent: () => false,
    }) as MediaQueryList
}
