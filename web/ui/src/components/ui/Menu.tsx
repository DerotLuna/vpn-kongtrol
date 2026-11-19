// Kongtrol UI kit — Menu (overflow "⋯" actions) · v1.0.0 · 2026-11-19
//
// WAI-ARIA APG "Menu Button" pattern: a button (aria-haspopup="menu",
// aria-expanded) opens a role="menu" list of role="menuitem" buttons.
// Opening moves focus to the first item (ArrowUp: the last); ArrowUp/Down
// wrap, Home/End jump, letters jump by label (type-ahead), Enter/Space
// choose, Esc closes and returns focus to the button, Tab closes. The list
// floats next to the button with Floating UI in the top layer (floating.ts,
// ADR-0064). Used where a row has more actions than fit (Inicio "Tus VPN").
import { useCallback, useId, useLayoutEffect, useRef, useState, type CSSProperties, type KeyboardEvent, type ReactNode } from 'react'
import type { ButtonSize } from './Button'
import { useAnchored, useDismiss, useTopLayer, useTypeahead } from './floating'
import { MoreIcon } from './icons'
import { Tooltip } from './Tooltip'

export interface MenuItem {
  id: string
  label: string
  icon?: ReactNode
  onSelect: () => void
  disabled?: boolean
  /** Destructive (red text). */
  danger?: boolean
}

interface MenuProps {
  /** Accessible name of the button and its tooltip ("Más acciones para trabajo"). */
  label: string
  items: MenuItem[]
  size?: ButtonSize
  icon?: ReactNode
  className?: string
}

export function Menu({ label, items, size = 'md', icon = <MoreIcon />, className = '' }: MenuProps) {
  const id = useId()
  const [open, setOpen] = useState(false)
  const [start, setStart] = useState<'first' | 'last'>('first')
  const button = useRef<HTMLButtonElement>(null)
  const listRef = useRef<HTMLDivElement | null>(null)
  const { setReference, setFloating, floatingStyles } = useAnchored(open, { placement: 'bottom-end', gap: 4 })

  const close = useCallback((refocus: boolean) => {
    setOpen(false)
    if (refocus) button.current?.focus()
  }, [])
  useDismiss(open, [listRef, button], (why) => close(why === 'escape'))

  const openWith = (where: 'first' | 'last') => {
    setStart(where)
    setOpen(true)
  }

  return (
    <span className={`k-menu ${className}`.trim()}>
      <Tooltip content={label}>
        <button
          ref={(el) => {
            button.current = el
            setReference(el)
          }}
          type="button"
          className={`k-icon-btn k-icon-btn--standard k-icon-btn--${size}`}
          aria-label={label}
          aria-haspopup="menu"
          aria-expanded={open}
          aria-controls={open ? id : undefined}
          onClick={() => (open ? close(false) : openWith('first'))}
          onKeyDown={(e) => {
            if (e.key === 'ArrowDown') {
              e.preventDefault()
              openWith('first')
            } else if (e.key === 'ArrowUp') {
              e.preventDefault()
              openWith('last')
            }
          }}
        >
          {icon}
        </button>
      </Tooltip>
      {open && (
        <MenuList
          id={id}
          label={label}
          items={items}
          start={start}
          listRef={listRef}
          setFloating={setFloating}
          style={floatingStyles}
          onClose={close}
        />
      )}
    </span>
  )
}

function MenuList({
  id,
  label,
  items,
  start,
  listRef,
  setFloating,
  style,
  onClose,
}: {
  id: string
  label: string
  items: MenuItem[]
  start: 'first' | 'last'
  listRef: { current: HTMLDivElement | null }
  setFloating: (el: HTMLElement | null) => void
  style: CSSProperties
  onClose: (refocus: boolean) => void
}) {
  useTopLayer(listRef)
  // Focus the first (or last) item once the list is in the top layer.
  useLayoutEffect(() => {
    const all = Array.from(listRef.current?.querySelectorAll<HTMLButtonElement>('[role="menuitem"]:not([aria-disabled="true"])') ?? [])
    ;(start === 'first' ? all[0] : all.at(-1))?.focus()
  }, [listRef, start])
  const enabled = () => Array.from(listRef.current?.querySelectorAll<HTMLButtonElement>('[role="menuitem"]:not([aria-disabled="true"])') ?? [])
  const focusAt = (i: number) => {
    const all = enabled()
    if (all.length === 0) return
    all[(i + all.length) % all.length]?.focus()
  }
  const current = () => enabled().indexOf(document.activeElement as HTMLButtonElement)
  const typeahead = useTypeahead(
    () => enabled().map((b) => b.textContent ?? ''),
    (i) => focusAt(i),
  )

  const onKey = (e: KeyboardEvent<HTMLDivElement>) => {
    const i = current()
    switch (e.key) {
      case 'ArrowDown':
        focusAt(i + 1)
        break
      case 'ArrowUp':
        focusAt(i - 1)
        break
      case 'Home':
        focusAt(0)
        break
      case 'End':
        focusAt(-1)
        break
      case 'Escape':
        onClose(true)
        break
      case 'Tab':
        onClose(false)
        return
      default:
        if (!typeahead(e.key, i)) return
    }
    e.preventDefault()
    e.stopPropagation()
  }

  return (
    <div
      ref={(el) => {
        listRef.current = el
        setFloating(el)
      }}
      id={id}
      role="menu"
      aria-label={label}
      popover="manual"
      className="k-menu__list k-pop"
      style={style}
      onKeyDown={onKey}
    >
      {items.map((it) => (
        <button
          key={it.id}
          type="button"
          role="menuitem"
          tabIndex={-1}
          className={`k-menu__item${it.danger ? ' k-menu__item--danger' : ''}`}
          aria-disabled={it.disabled || undefined}
          onClick={() => {
            if (it.disabled) return
            onClose(true)
            it.onSelect()
          }}
        >
          {it.icon && (
            <span className="k-menu__icon" aria-hidden="true">
              {it.icon}
            </span>
          )}
          <span>{it.label}</span>
        </button>
      ))}
    </div>
  )
}
