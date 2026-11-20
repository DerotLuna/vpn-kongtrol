// Kongtrol UI — tray flyout bridge · v1.2.0 · 2026-11-20 (ADR-0036, ADR-0056)
//
// Inside the tray flyout (cmd/kongtrol-tray/flyout_windows.go) the tray
// injects `window.kongtrolTray` before any page script, only on this exact
// page; it signs each message with a per-launch nonce. The names below are
// the closed list the tray accepts (flyoutMessages in flyout.go). In a
// normal browser tab there is no bridge and the page works as a compact
// status view without the window-only controls.

export type TrayMessage = 'ready' | 'close' | 'open-dashboard' | 'open-add' | 'quit-tray' | 'quit-all'

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

/** Sent by the tray every time the flyout is shown again (not the first time). */
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
