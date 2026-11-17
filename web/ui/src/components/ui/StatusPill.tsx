import type { ReactNode } from 'react'

export type PillTone = 'ok' | 'warn' | 'bad' | 'idle'

export function StatusPill({ tone, children }: { tone: PillTone; children: ReactNode }) {
  return <span className={tone === 'idle' ? 'pill' : `pill ${tone}`}>{children}</span>
}
