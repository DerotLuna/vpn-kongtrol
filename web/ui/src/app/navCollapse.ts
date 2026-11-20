// Kongtrol UI — collapsible side menu state · v1.0.0 · 2026-11-19
//
// The side menu (≥ 600px) is either the sidebar with labels or an icon
// rail with tooltips (owner feedback 2026-11-19). On wide windows
// (≥ 1100px) the user's choice is remembered on this device (localStorage,
// lib/storage.ts: never throws); below that the menu starts collapsed so
// the page keeps its room, and "Expandir menú" opens it for this visit
// only. Pure state, no layout: AppShell renders it as data-nav.
import { useCallback, useEffect, useState } from 'react'
import { readStored, writeStored } from '../lib/storage'

export const NAV_KEY = 'kongtrol.nav'
/** Below this width the menu starts collapsed (Fluent NavigationView: compact under ~1008–1200px). */
export const NAV_WIDE_QUERY = '(min-width: 1100px)'

function wideNow(): boolean {
  try {
    return window.matchMedia(NAV_WIDE_QUERY).matches
  } catch {
    return true
  }
}

export function useNavCollapse(): { collapsed: boolean; toggle: () => void } {
  const [wide, setWide] = useState(wideNow)
  const [stored, setStored] = useState(() => readStored(NAV_KEY))
  // Narrow windows: a per-visit choice, back to collapsed when the width changes.
  const [narrowOpen, setNarrowOpen] = useState(false)

  useEffect(() => {
    let mq: MediaQueryList
    try {
      mq = window.matchMedia(NAV_WIDE_QUERY)
    } catch {
      return
    }
    const on = (e: MediaQueryListEvent) => {
      setWide(e.matches)
      setNarrowOpen(false)
    }
    mq.addEventListener?.('change', on)
    return () => mq.removeEventListener?.('change', on)
  }, [])

  const collapsed = wide ? stored === 'collapsed' : !narrowOpen
  const toggle = useCallback(() => {
    if (wide) {
      const next = collapsed ? 'expanded' : 'collapsed'
      writeStored(NAV_KEY, next)
      setStored(next)
    } else setNarrowOpen(collapsed)
  }, [wide, collapsed])
  return { collapsed, toggle }
}
