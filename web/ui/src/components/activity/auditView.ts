// Kongtrol UI — Actividad › Registro de auditoría: what the list shows · v1.0.0 · 2026-11-25
//
// Same reading rules as Tráfico › Recientes (recentView.ts, ADR-0106):
//   - the first 10 entries show, then 20 more per "Ver 20 más"; when the
//     entries already loaded run out, the next older page is asked from the
//     daemon (`before` = the last timestamp, X-Kongtrol-Audit-More);
//   - "En vivo" never pushes rows under the reader: entries newer than the
//     newest one on screen wait behind "N nuevas · Mostrar".
// Unlike Recientes the log only grows at the top (newest first) and older
// pages are appended at the bottom, so "new" means newer than the snapshot's
// newest entry, not "not seen yet" (an older page is never "new").
import { useState } from 'react'
import type { AuditEvent } from '../../data/activity'

export const AUDIT_FIRST = 10
export const AUDIT_STEP = 20

const at = (ev: Pick<AuditEvent, 'timestamp'>) => Date.parse(ev.timestamp)

export interface AuditView {
  /** Entries up to the snapshot's newest one, newest first. */
  shown: AuditEvent[]
  /** Entries that arrived after the snapshot (newest first). */
  fresh: AuditEvent[]
  /** Takes the current list as the new snapshot (new entries on top). */
  merge: () => void
  /** Forgets the snapshot: the next list is taken as it comes (after a manual refresh). */
  reset: () => void
}

/**
 * Splits the loaded entries into what is on screen and what arrived since.
 * `scope` names the current filters: a new filter starts a new snapshot.
 */
export function useAuditView(rows: readonly AuditEvent[] | undefined, scope: string): AuditView {
  const [snap, setSnap] = useState<{ scope: string; newest: number | null } | null>(null)
  const top = rows?.[0]
  // Nothing on screen yet, an empty list, or new filters: nothing can move
  // under the reader, so take the list as it comes (state adjusted during render).
  if (rows && (snap === null || snap.scope !== scope || (snap.newest === null && top))) {
    setSnap({ scope, newest: top ? at(top) : null })
  }
  const newest = snap && snap.scope === scope ? snap.newest : top ? at(top) : null
  const all = rows ?? []
  const reset = () => setSnap(null)
  if (newest === null) return { shown: [...all], fresh: [], merge: reset, reset }
  return {
    shown: all.filter((ev) => at(ev) <= newest),
    fresh: all.filter((ev) => at(ev) > newest),
    merge: () => setSnap({ scope, newest: top ? at(top) : null }),
    reset,
  }
}
