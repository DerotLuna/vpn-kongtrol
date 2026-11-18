// Kongtrol UI kit — Tabs · v0.1.0 · 2026-11-18
//
// Two kinds, never mixed:
// - <Tabs>: in-page panels, WAI-ARIA APG tabs pattern (roving tabindex,
//   ←/→/Home/End move and activate, panels labelled by their tab).
// - <TabLinks>: sub-pages with their own URL (e.g. Perfiles · Grupos); a nav
//   of links with aria-current, so Back/Forward and deep links work.
import { useId, useRef, type KeyboardEvent, type ReactNode } from 'react'
import { NavLink } from 'react-router'

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
      <div role="tablist" aria-label={label} className="k-tabs__list" onKeyDown={onKeyDown}>
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
  return (
    <nav aria-label={label} className="k-tabs__list k-tabs__list--links">
      {items.map((item) => (
        <NavLink key={item.to} to={item.to} end={item.end ?? true} className="k-tab">
          {item.label}
        </NavLink>
      ))}
    </nav>
  )
}
