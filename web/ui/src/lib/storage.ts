// localStorage that never throws (private windows, blocked site data).
// Keys are shared with the legacy dashboard (same origin), so a choice made
// in either UI applies to both.

export function readStored(key: string): string {
  try {
    return localStorage.getItem(key) ?? ''
  } catch {
    return ''
  }
}

export function writeStored(key: string, value: string): void {
  try {
    if (value) localStorage.setItem(key, value)
    else localStorage.removeItem(key)
  } catch {
    // Storage unavailable: the choice still applies for this page view.
  }
}
