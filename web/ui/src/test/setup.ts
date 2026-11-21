import '@testing-library/jest-dom/vitest'
import { cleanup, configure } from '@testing-library/react'
import { afterEach, beforeEach, vi } from 'vitest'
import { authState, resetSessionForTests } from '../api/auth'
import { connection, liveState } from '../data/connection'
import { toasts } from '../data/toast'
import { resetTransitions } from '../data/transition'

// Pages load on demand (route `lazy`, ADR-0041). In a test, the first visit
// to a page used to transform and import its module inside the test, after
// the clock of findBy*/waitFor (1 s) had started: under CPU load that cold
// import alone could take longer, so each file's first test failed at random
// (seen on Actividad, Perfiles, Grupos, Ajustes, Seguridad). Importing every
// page module here, before any test runs, takes that cost out of the timed
// part; the route's lazy() then resolves from the module cache at once.
await Promise.all(Object.values(import.meta.glob('../pages/*Page.tsx')).map((load) => load()))

// Last resort, measured: with the cold imports gone, 1 of 10 full runs still
// failed while `go build -a` kept every core busy — always a file's first
// test, which also pays the first React/jsdom render of the whole App shell
// in that worker. findBy*/waitFor wait for a condition and return as soon
// as it holds, so a higher ceiling (3 s instead of 1 s) slows only a test
// that is failing anyway; it never hides a wrong result.
configure({ asyncUtilTimeout: 3_000 })

// Start every test on Inicio. BASE_URL is "/" in tests and builds alike;
// the router's basename follows it.
beforeEach(() => {
  window.history.replaceState(null, '', import.meta.env.BASE_URL)
})

afterEach(() => {
  cleanup()
  toasts.clear()
  resetTransitions()
  resetSessionForTests()
  authState.set('ok')
  connection.set('unknown')
  liveState.set('connecting')
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  localStorage.clear()
})

// jsdom's UA stylesheet hides [popover]:not(:popover-open) but jsdom has no
// showPopover(), so the kit's floating layers (menu, listbox, tooltip,
// calendar; floating.ts) would stay hidden from queries. They are mounted
// only while open, so showing every popover is the browser's behaviour.
const popoverStyle = document.createElement('style')
popoverStyle.textContent = '[popover] { display: block !important; }'
document.head.append(popoverStyle)
// Floating UI reads ResizeObserver in autoUpdate; jsdom has none.
if (!('ResizeObserver' in window)) {
  ;(window as unknown as { ResizeObserver: unknown }).ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
}

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
