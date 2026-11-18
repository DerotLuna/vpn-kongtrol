// Kongtrol UI kit — Tooltip · v0.1.0 · 2026-11-18
//
// WCAG 1.4.13 (content on hover or focus): shows on hover AND keyboard
// focus, stays while the pointer moves onto it (hoverable), closes with Esc
// without moving focus (dismissible), and stays until the trigger loses
// hover/focus (persistent). The trigger keeps its own accessible name; the
// tooltip is linked with aria-describedby only when it adds something.
import { cloneElement, useEffect, useId, useRef, useState, type ReactElement } from 'react'

interface TooltipProps {
  content: string
  children: ReactElement<{ 'aria-describedby'?: string; 'aria-label'?: string }>
  /** Describe the trigger (true) or just label it visually (default: when the text differs from aria-label). */
  describe?: boolean
}

export function Tooltip({ content, children, describe }: TooltipProps) {
  const id = useId()
  const [open, setOpen] = useState(false)
  const hideTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)

  const show = () => {
    clearTimeout(hideTimer.current)
    setOpen(true)
  }
  // Small delay so the pointer can travel onto the tooltip (hoverable).
  const hide = () => {
    clearTimeout(hideTimer.current)
    hideTimer.current = setTimeout(() => setOpen(false), 120)
  }

  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [open])
  useEffect(() => () => clearTimeout(hideTimer.current), [])

  const labelled = children.props['aria-label'] === content
  const linkDescription = describe ?? !labelled
  const trigger = cloneElement(children, linkDescription ? { 'aria-describedby': id } : {})

  return (
    <span className="k-tooltip-anchor" onPointerEnter={show} onPointerLeave={hide} onFocus={show} onBlur={hide}>
      {trigger}
      <span
        id={id}
        role="tooltip"
        className="k-tooltip"
        data-open={open || undefined}
        onPointerEnter={show}
        onPointerLeave={hide}
      >
        {content}
      </span>
    </span>
  )
}
