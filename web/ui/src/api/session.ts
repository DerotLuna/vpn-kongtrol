// Kongtrol UI — browser sign-in · v0.1.0 · 2026-11-18
//
// Kongtrol (tray, `kongtrol dashboard`, tray flyout) opens this UI with a
// one-time nonce in the fragment: /app/#k=<nonce> (ADR-0039). startSession
// removes it from the address bar and trades it for an HttpOnly session
// cookie (POST /api/v1/session). Without a nonce it does nothing: an
// existing session cookie may still be valid; if not, API calls answer 401
// and ErrorNotice shows how to open the panel from Kongtrol.

/** Reads and strips `k` from the fragment; returns '' when absent. */
export function takeNonce(loc: Location = location, hist: History = history): string {
  const params = new URLSearchParams(loc.hash.slice(1))
  const nonce = params.get('k') ?? ''
  if (!nonce) return ''
  params.delete('k')
  const rest = params.toString()
  try {
    hist.replaceState(hist.state, '', loc.pathname + loc.search + (rest ? `#${rest}` : ''))
  } catch {
    // Keep going: the nonce is single-use anyway.
  }
  return nonce
}

/** Signs this browser in when the URL carries a nonce. Never throws. */
export async function startSession(): Promise<void> {
  const nonce = takeNonce()
  if (!nonce) return
  try {
    await fetch('/api/v1/session', {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ nonce }),
    })
  } catch {
    // Offline daemon: the normal API error handling shows it.
  }
}
