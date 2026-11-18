// Kongtrol UI — tray flyout bridge · v1.0.0 · 2026-11-18 (ADR-0036)
//
// Inside the tray flyout (cmd/kongtrol-tray/flyout_windows.go) the tray
// injects `window.kongtrolTray` before any page script, only on this exact
// page; it signs each message with a per-launch nonce. The names below are
// the closed list the tray accepts (flyoutMessages in flyout.go). In a
// normal browser tab there is no bridge and the page works as a compact
// status view without the window-only controls.

export type TrayMessage = 'ready' | 'close' | 'open-dashboard' | 'quit-tray' | 'quit-all'

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

// The flyout page is the `tray` route under /app (App.tsx; flyoutPath in
// cmd/kongtrol-tray/flyout.go).

/** Sent by the tray every time the flyout is shown again (not the first time). */
export const SHOWN_EVENT = 'kongtrol:shown'
