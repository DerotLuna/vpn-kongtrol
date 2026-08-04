import { useEffect, useRef, useState } from 'react'

export interface JumpOption {
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

// Native <select> can't be restyled once open (its option list is drawn by
// the OS, not the page) — so the "jump to section" control is a custom
// listbox instead, styled like the rest of the site and with its own
// open/close animation.
export default function SectionJumpMenu({ label, value, options, onChange, className }: Props) {
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)
  const current = options.find(o => o.id === value) ?? options[0]

  useEffect(() => {
    if (!open) return
    const onPointerDown = (e: MouseEvent) => {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) setOpen(false)
    }
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') setOpen(false) }
    document.addEventListener('mousedown', onPointerDown)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onPointerDown)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  const select = (id: string) => {
    onChange(id)
    setOpen(false)
  }

  return (
    <nav className={`guide-mobile-nav${className ? ` ${className}` : ''}`} ref={rootRef}>
      <span className="guide-mobile-nav-label">{label}</span>
      <div className={`jump-select${open ? ' open' : ''}`}>
        <button
          type="button"
          className="jump-select-trigger"
          aria-haspopup="listbox"
          aria-expanded={open}
          onClick={() => setOpen(o => !o)}
        >
          <span className="jump-select-value">{current.num} — {current.label}</span>
          <svg className="jump-select-arrow" viewBox="0 0 10 6" aria-hidden="true">
            <path d="M1 1l4 4 4-4" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
        </button>
        <ul className="jump-select-list" role="listbox">
          {options.map(o => (
            <li key={o.id} role="option" aria-selected={o.id === value}>
              <button type="button" className={o.id === value ? 'active' : ''} onClick={() => select(o.id)}>
                <span className="guide-num">{o.num}</span>{o.label}
              </button>
            </li>
          ))}
        </ul>
      </div>
    </nav>
  )
}
