import type { ReactNode } from 'react'

// ok: up / ready (green, only for that). on: a setting that is switched on.
// warn: in progress. caution: needs attention, not broken. bad: error.
export type PillTone = 'ok' | 'on' | 'warn' | 'caution' | 'bad' | 'idle'

export function StatusPill({ tone, children }: { tone: PillTone; children: ReactNode }) {
  return <span className={tone === 'idle' ? 'pill' : `pill ${tone}`}>{children}</span>
}
