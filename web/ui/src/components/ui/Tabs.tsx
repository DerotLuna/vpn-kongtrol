// Kongtrol UI kit — Tabs · v0.2.0 · 2026-11-20
//
// Two kinds, never mixed:
// - <Tabs>: in-page panels, WAI-ARIA APG tabs pattern (roving tabindex,
//   ←/→/Home/End move and activate, panels labelled by their tab).
// - <TabLinks>: sub-pages with their own URL (e.g. Perfiles · Grupos); a nav
//   of links with aria-current, so Back/Forward and deep links work.
// Both draw one selection mark that slides to the chosen tab (ADR-0078):
// position and width are CSSOM writes (allowed by the page CSP), measured
// after layout and on resize; no slide on first paint or with reduced motion.
import { useId, useLayoutEffect, useRef, type KeyboardEvent, type ReactNode } from 'react'
import { NavLink, useLocation } from 'react-router'

/** Moves the list's ink under its selected tab ([aria-selected=true] or [aria-current=page]). */
function useInk<T extends HTMLElement>(key: unknown) {
  const list = useRef<T>(null)
  const ink = useRef<HTMLSpanElement>(null)
  useLayoutEffect(() => {
    const el = list.current
    const mark = ink.current
    if (!el || !mark) return
    const place = () => {
      const tab = el.querySelector<HTMLElement>('[aria-selected="true"], [aria-current="page"]')
      if (!tab) {
        mark.style.opacity = '0'
        return
      }
      mark.style.opacity = '1'
      mark.style.width = `${Math.max(0, tab.offsetWidth - 20)}px`
      mark.style.transform = `translateX(${tab.offsetLeft + 10}px)`
    }
    place()
    // slide only after the first placement
    requestAnimationFrame(() => mark.setAttribute('data-ready', ''))
    const ro = new ResizeObserver(place)
    ro.observe(el)
    return () => ro.disconnect()
  }, [key])
  return { list, ink }
}

export interface TabItem {
  id: string
  label: string
  panel: ReactNode
}

interface TabsProps {
  label: string
  items: TabItem[]
  value: string
  onChange: (id: string) => void
}

export function Tabs({ label, items, value, onChange }: TabsProps) {
  const base = useId()
  const refs = useRef<(HTMLButtonElement | null)[]>([])
  const index = Math.max(0, items.findIndex((i) => i.id === value))
  const { list: listRef, ink: inkRef } = useInk<HTMLDivElement>(value)

  const onKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    const last = items.length - 1
    const next = { ArrowRight: index === last ? 0 : index + 1, ArrowLeft: index === 0 ? last : index - 1, Home: 0, End: last }[
      e.key
    ]
    if (next === undefined) return
    e.preventDefault()
    const item = items[next]
    if (!item) return
    onChange(item.id)
    refs.current[next]?.focus()
  }

  return (
    <div className="k-tabs">
      <div ref={listRef} role="tablist" aria-label={label} className="k-tabs__list k-tabs__list--ink" onKeyDown={onKeyDown}>
        {items.map((item, i) => {
          const selected = i === index
          return (
            <button
              key={item.id}
              ref={(el) => {
                refs.current[i] = el
              }}
              type="button"
              role="tab"
              id={`${base}-tab-${item.id}`}
              aria-selected={selected}
              aria-controls={`${base}-panel-${item.id}`}
              tabIndex={selected ? 0 : -1}
              className="k-tab"
              onClick={() => onChange(item.id)}
            >
              {item.label}
            </button>
          )
        })}
        <span ref={inkRef} className="k-tabs__ink" aria-hidden="true" />
      </div>
      {items.map((item, i) => (
        <div
          key={item.id}
          role="tabpanel"
          id={`${base}-panel-${item.id}`}
          aria-labelledby={`${base}-tab-${item.id}`}
          hidden={i !== index}
          tabIndex={0}
          className="k-tabs__panel"
        >
          {i === index && item.panel}
        </div>
      ))}
    </div>
  )
}

export interface TabLinkItem {
  to: string
  label: string
  end?: boolean
}

export function TabLinks({ label, items }: { label: string; items: TabLinkItem[] }) {
  const { pathname } = useLocation()
  const { list: listRef, ink: inkRef } = useInk<HTMLElement>(pathname)
  return (
    <nav ref={listRef} aria-label={label} className="k-tabs__list k-tabs__list--links k-tabs__list--ink">
      {items.map((item) => (
        <NavLink key={item.to} to={item.to} end={item.end ?? true} className="k-tab">
          {item.label}
        </NavLink>
      ))}
      <span ref={inkRef} className="k-tabs__ink" aria-hidden="true" />
    </nav>
  )
}
