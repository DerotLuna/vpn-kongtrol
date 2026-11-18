// Kongtrol UI — motion helpers · v1.0.0 · 2026-11-18
//
// The durations live in tokens.css (--dur-*); JS only needs them to wait
// for an exit animation before removing an element. With reduced motion,
// or where the stylesheet is absent (tests), the wait is 0: removal is
// immediate and nothing is left half-faded.

const REDUCE = '(prefers-reduced-motion: reduce)'

/** Milliseconds of a --dur-* token, or 0 when motion is reduced or unknown. */
export function motionMs(token: '--dur-exit' | '--dur-fast' | '--dur-med'): number {
  if (typeof window === 'undefined') return 0
  if (window.matchMedia?.(REDUCE).matches) return 0
  const raw = getComputedStyle(document.documentElement).getPropertyValue(token).trim()
  const n = parseFloat(raw)
  if (!Number.isFinite(n)) return 0
  return raw.endsWith('ms') ? n : raw.endsWith('s') ? n * 1000 : 0
}
