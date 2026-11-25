// Kongtrol UI — tray flyout bridge · v1.2.0 · 2026-11-20 (ADR-0036, ADR-0056)
//
// Inside the tray flyout (cmd/kongtrol-tray/flyout_windows.go) the tray
// injects `window.kongtrolTray` before any page script, only on this exact
// page; it signs each message with a per-launch nonce. The names below are
// the closed list the tray accepts (flyoutMessages in flyout.go). In a
// normal browser tab there is no bridge and the page works as a compact
// status view without the window-only controls.

export type TrayMessage =
  | 'ready'
  | 'close'
  | 'open-dashboard'
  | 'open-add'
  | 'open-diagnostics'
  | 'reauth'
  | 'quit-tray'
  | 'quit-all'
  | AddProviderMessage

/**
 * A notice's "Agregar <App>" (ADR-0102): the panel's "＋ Agregar" sheet with
 * that provider chosen. One message per provider keeps the list closed
 * (flyoutAddProviders in flyout.go): the page never sends a free value.
 */
const ADD_PROVIDER_MESSAGES = {
  protonvpn: 'open-add-protonvpn',
  nordvpn: 'open-add-nordvpn',
  surfshark: 'open-add-surfshark',
  forticlient: 'open-add-forticlient',
  mullvad: 'open-add-mullvad',
  pia: 'open-add-pia',
  expressvpn: 'open-add-expressvpn',
  windscribe: 'open-add-windscribe',
  cyberghost: 'open-add-cyberghost',
} as const

export type AddProviderMessage = (typeof ADD_PROVIDER_MESSAGES)[keyof typeof ADD_PROVIDER_MESSAGES]

/** The message for "Agregar <App>", or plain 'open-add' for a type the tray does not list. */
export function addProviderMessage(type: string): TrayMessage {
  return Object.hasOwn(ADD_PROVIDER_MESSAGES, type) ? ADD_PROVIDER_MESSAGES[type as keyof typeof ADD_PROVIDER_MESSAGES] : 'open-add'
}

interface TrayHost {
  post: (message: TrayMessage) => void
}

declare global {
  interface Window {
    kongtrolTray?: TrayHost
  }
}

/** The tray host when this page runs inside the flyout window, else null. */
export function trayHost(): TrayHost | null {
  return typeof window !== 'undefined' && window.kongtrolTray ? window.kongtrolTray : null
}

// The flyout page is the `tray` route at /tray (App.tsx; flyoutPath in
// cmd/kongtrol-tray/flyout.go).

/** Sent by the tray every time it shows the flyout (a pre-warmed page gets it on the first show too, ADR-0074). */
export const SHOWN_EVENT = 'kongtrol:shown'

/**
 * Posts 'ready' once the page is really on screen: visible and painting a
 * frame. The tray waits for it after every show and, without it within a
 * few seconds, closes the flyout and opens its native menu (ADR-0056), so
 * 'ready' is never sent for a page nobody can see. Returns a cleanup that
 * cancels a pending announcement.
 */
export function announceReady(
  host: TrayHost,
  doc: Document = document,
  raf: (cb: () => void) => unknown = (cb) => requestAnimationFrame(cb),
): () => void {
  let done = false
  const post = () => {
    if (done || doc.visibilityState !== 'visible') return
    done = true
    doc.removeEventListener('visibilitychange', post)
    raf(() => host.post('ready'))
  }
  doc.addEventListener('visibilitychange', post)
  post()
  return () => {
    done = true
    doc.removeEventListener('visibilitychange', post)
  }
}
