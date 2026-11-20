// Kongtrol UI kit — StatusPill · v0.2.0 · 2026-11-18
//
// DESIGN.md Four States Rule: ok (connected / check passed) = green,
// connecting = steel, caution (up but not answering, degraded) = orange,
// bad (error) = red. Plus idle (off, neutral), unknown (dashed, not loaded
// or not verifiable) and on (a setting that is switched on: neutral, never
// green or gold — gold is the brand, not a state). Every pill carries its
// word; the dot shape differs per state too, so color is never the only cue.
import type { ReactNode } from 'react'

export type PillTone = 'ok' | 'connecting' | 'caution' | 'bad' | 'idle' | 'unknown' | 'on'

export function StatusPill({ tone, children }: { tone: PillTone; children: ReactNode }) {
  return <span className={tone === 'idle' ? 'pill' : `pill ${tone}`}>{children}</span>
}
