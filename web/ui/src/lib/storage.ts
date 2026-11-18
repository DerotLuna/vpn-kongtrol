// localStorage that never throws (private windows, blocked site data).
// Keys predate this UI (the classic dashboard used the same ones), so a
// choice saved before the switch still applies.

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
