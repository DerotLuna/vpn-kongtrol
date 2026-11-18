import { KeyboardEvent, useEffect, useId, useRef, useState } from 'react'

interface JumpOption {
  id: string
  num: string
  label: string
}

interface Props {
  label: string
  value: string
  options: JumpOption[]
  onChange: (id: string) => void
  className?: string
}

// "Jump to section" on narrow screens. Native <select> can't be restyled once
// open, so this is a disclosure navigation (WAI-ARIA APG "Disclosure
// Navigation Menu": https://www.w3.org/WAI/ARIA/apg/patterns/disclosure/):
// a button that shows a list of in-page links. Plain links keep it working
// without JavaScript and let the browser move focus to the section.
// Keyboard: Enter/Space/↓ opens and focuses the current item, ↑/↓/Home/End
// move between items, Escape closes and returns focus to the button.
export default function SectionJumpMenu({ label, value, options, onChange, className }: Props) {
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLElement>(null)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const linkRefs = useRef<(HTMLAnchorElement | null)[]>([])
  const listId = useId()
  const current = options.find(o => o.id === value) ?? options[0]
  const currentIndex = Math.max(0, options.findIndex(o => o.id === current.id))

  useEffect(() => {
    if (!open) return
    const onPointerDown = (e: PointerEvent) => {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) setOpen(false)
    }
    const onKey = (e: globalThis.KeyboardEvent) => {
      if (e.key !== 'Escape') return
      setOpen(false)
      triggerRef.current?.focus()
    }
    document.addEventListener('pointerdown', onPointerDown)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('pointerdown', onPointerDown)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  const focusItem = (i: number) => {
    const n = options.length
    linkRefs.current[(i + n) % n]?.focus()
  }

  const openAndFocus = (i: number) => {
    setOpen(true)
    // the list becomes visible on the next frame (visibility transition)
    requestAnimationFrame(() => focusItem(i))
  }

  const onTriggerKey = (e: KeyboardEvent<HTMLButtonElement>) => {
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault()
      openAndFocus(currentIndex)
    }
  }

  const onItemKey = (e: KeyboardEvent<HTMLAnchorElement>, i: number) => {
    const to =
      e.key === 'ArrowDown' ? i + 1
      : e.key === 'ArrowUp' ? i - 1
      : e.key === 'Home' ? 0
      : e.key === 'End' ? options.length - 1
      : null
    if (to == null) return
    e.preventDefault()
    focusItem(to)
  }

  return (
    <nav className={`guide-mobile-nav${className ? ` ${className}` : ''}`} ref={rootRef} aria-label={label}>
      <span className="guide-mobile-nav-label" aria-hidden="true">{label}</span>
      <div className={`jump-select${open ? ' open' : ''}`}>
        <button
          ref={triggerRef}
          type="button"
          className="jump-select-trigger"
          aria-expanded={open}
          aria-controls={listId}
          onClick={() => setOpen(o => !o)}
          onKeyDown={onTriggerKey}
        >
          <span className="jump-select-value">{current.num} — {current.label}</span>
          <svg className="jump-select-arrow" viewBox="0 0 10 6" aria-hidden="true">
            <path d="M1 1l4 4 4-4" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
        </button>
        <ul className="jump-select-list" id={listId}>
          {options.map((o, i) => (
            <li key={o.id}>
              <a
                ref={el => { linkRefs.current[i] = el }}
                href={`#${o.id}`}
                aria-current={o.id === value ? 'true' : undefined}
                onKeyDown={e => onItemKey(e, i)}
                onClick={() => { onChange(o.id); setOpen(false) }}
              >
                <span className="guide-num">{o.num}</span>{o.label}
              </a>
            </li>
          ))}
        </ul>
      </div>
    </nav>
  )
}
