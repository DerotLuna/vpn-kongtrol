// Kongtrol UI — retry delays · v0.1.0 · 2026-11-18
//
// Exponential backoff with "full jitter" (AWS Architecture Blog, "Exponential
// Backoff And Jitter"): delay = random(0, min(cap, base * 2^attempt)). Jitter
// spreads reconnects of several open tabs after a daemon restart. A small
// floor keeps the first retry from being instant.
export interface BackoffOptions {
  base?: number
  cap?: number
  floor?: number
  random?: () => number
}

export function backoffDelay(attempt: number, opts: BackoffOptions = {}): number {
  const { base = 1_000, cap = 30_000, floor = 250, random = Math.random } = opts
  const ceiling = Math.min(cap, base * 2 ** Math.max(0, attempt))
  return Math.max(floor, Math.round(random() * ceiling))
}
