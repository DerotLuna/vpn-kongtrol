// Kongtrol UI — Tráfico › Recientes: what the list shows · v1.0.0 · 2026-11-24
//
// ADR-0106: a live list (refreshed every 15 s, up to 500 entries) is read
// with "load more" + filters instead of pages, and live updates never move
// what the user is reading:
//   - the rows on screen are a snapshot; each row's own data (Visto, count)
//     still updates in place, but no row is added, removed or reordered
//     until the user asks ("N nuevas · Mostrar");
//   - the first 10 rows show, then 20 more per "Ver 20 más (quedan N)";
//   - search (site, IP or program) and "Salió por" filter the whole list
//     client-side (the daemon keeps at most 500 entries).
// Same filter rules as `kongtrol traffic recent --search/--via`
// (cmd/kongtrol/cmd_traffic.go filterRecent).
import { useState } from 'react'
import type { RecentConnection } from '../../data/recent'

export const RECENT_FIRST = 10
export const RECENT_STEP = 20

/** "Salió por" choices: every entry, the normal connection, or one profile ("p:<name>"). */
export type ViaFilter = 'all' | 'direct' | `p:${string}`

export const viaProfile = (name: string): ViaFilter => `p:${name}`

/** One entry per destination, program and interface (internal/monitor/recent.go recentKey). */
export function recentKey(c: RecentConnection): string {
  return `${c.remote_ip}|${c.port}|${c.process ?? ''}|${c.interface ?? ''}|${c.via ?? ''}`
}

export function matchesRecent(c: RecentConnection, search: string, via: ViaFilter): boolean {
  if (via === 'direct' && c.via) return false
  if (via.startsWith('p:') && c.via !== via.slice(2)) return false
  const q = search.trim().toLocaleLowerCase()
  if (!q) return true
  return [c.host, c.remote_ip, c.process].some((s) => !!s && s.toLocaleLowerCase().includes(q))
}

/** The profiles found in the list, sorted, for the "Salió por" choices. */
export function viaNames(rows: readonly RecentConnection[]): string[] {
  const names = new Set<string>()
  for (const r of rows) if (r.via) names.add(r.via)
  return [...names].sort((a, b) => a.localeCompare(b))
}

export interface StableList<T> {
  /** The rows on screen: the snapshot's order, each row's latest data. */
  shown: T[]
  /** Entries that arrived after the snapshot (newest first). */
  fresh: T[]
  /** Takes the current list as the new snapshot (new entries on top). */
  merge: () => void
}

/**
 * Keeps the visible order stable across live refreshes. The first non-empty
 * list is taken as is; later lists only refresh the data of rows already
 * shown, and report the rest as `fresh` until `merge()`. A row that left the
 * daemon's list (forgotten after an hour) stays until the next merge.
 */
export function useStableList<T>(rows: readonly T[] | undefined, keyOf: (row: T) => string): StableList<T> {
  const [snap, setSnap] = useState<readonly T[] | null>(null)
  // Nothing on screen yet (first load, or an empty list): nothing can move,
  // so take the list as it comes (state adjusted during render).
  if (rows && (snap === null || (snap.length === 0 && rows.length > 0))) setSnap(rows)
  const current = snap ?? rows ?? []
  const latest = new Map<string, T>()
  for (const r of rows ?? []) latest.set(keyOf(r), r)
  const onScreen = new Set(current.map(keyOf))
  return {
    shown: current.map((r) => latest.get(keyOf(r)) ?? r),
    fresh: (rows ?? []).filter((r) => !onScreen.has(keyOf(r))),
    merge: () => setSnap(rows ?? []),
  }
}
