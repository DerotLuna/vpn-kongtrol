// Kongtrol UI kit — Tooltip · v1.0.0 · 2026-11-19
//
// The one tooltip of the panel (every IconButton uses it). WCAG 1.4.13
// (content on hover or focus): opens on hover after a short delay and at
// once on keyboard focus, stays while the pointer moves onto it (hoverable),
// closes with Esc without moving focus (dismissible) and stays until the
// trigger loses hover/focus (persistent). Touch has no hover: a long press
// (500 ms) shows it for a moment and does not press the button (Material's
// touch tooltip). Positioned with Floating UI (flip/shift inside the
// viewport) in the top layer (floating.ts, ADR-0067), so it never clips at
// a screen edge, inside a table or over a dialog. Screen readers get the
// text through aria-describedby on a hidden node that is always there; the
// floating bubble is visual only (aria-hidden), so nothing is read twice.
import { cloneElement, useEffect, useId, useRef, useState, type CSSProperties, type ReactElement } from 'react'
import { useAnchored, useTopLayer } from './floating'

/** Hover delay before showing (Material: 500 ms; a bit faster for a dense panel). */
export const TOOLTIP_DELAY_MS = 350
const HIDE_MS = 100
const LONG_PRESS_MS = 500
const TOUCH_SHOW_MS = 1500

interface TooltipProps {
  content: string
  children: ReactElement<{ 'aria-describedby'?: string; 'aria-label'?: string }>
  /** Describe the trigger (true) or just label it visually (default: when the text differs from aria-label). */
  describe?: boolean
  placement?: 'top' | 'bottom'
}

function keyboardFocus(el: Element): boolean {
  try {
    return el.matches(':focus-visible')
  } catch {
    return true
  }
}

export function Tooltip({ content, children, describe, placement = 'top' }: TooltipProps) {
  const id = useId()
  const [open, setOpen] = useState(false)
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)
  const pressTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)
  const longPressed = useRef(false)
  const bubbleRef = useRef<HTMLSpanElement | null>(null)
  const { setReference, setFloating, floatingStyles } = useAnchored(open, { placement, gap: 6 })

  const later = (fn: () => void, ms: number) => {
    clearTimeout(timer.current)
    timer.current = setTimeout(fn, ms)
  }
  const showNow = () => {
    clearTimeout(timer.current)
    setOpen(true)
  }
  const hide = () => later(() => setOpen(false), HIDE_MS)

  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [open])
  useEffect(
    () => () => {
      clearTimeout(timer.current)
      clearTimeout(pressTimer.current)
    },
    [],
  )

  const labelled = children.props['aria-label'] === content
  const linkDescription = describe ?? !labelled
  const trigger = cloneElement(children, linkDescription ? { 'aria-describedby': id } : {})

  return (
    <span
      ref={setReference}
      className="k-tooltip-anchor"
      onPointerEnter={(e) => {
        if (e.pointerType !== 'touch') later(() => setOpen(true), TOOLTIP_DELAY_MS)
      }}
      onPointerLeave={(e) => {
        if (e.pointerType !== 'touch') hide()
      }}
      onPointerDown={(e) => {
        if (e.pointerType !== 'touch') {
          // A click answers the hover: the tooltip gets out of the way.
          clearTimeout(timer.current)
          setOpen(false)
          return
        }
        longPressed.current = false
        clearTimeout(pressTimer.current)
        pressTimer.current = setTimeout(() => {
          longPressed.current = true
          showNow()
          later(() => setOpen(false), TOUCH_SHOW_MS)
        }, LONG_PRESS_MS)
      }}
      onPointerUp={() => clearTimeout(pressTimer.current)}
      onPointerCancel={() => clearTimeout(pressTimer.current)}
      onContextMenu={(e) => {
        if (longPressed.current) e.preventDefault()
      }}
      onClickCapture={(e) => {
        // The long press showed the tooltip; it does not also press the button.
        if (longPressed.current) {
          longPressed.current = false
          e.preventDefault()
          e.stopPropagation()
        }
      }}
      onFocus={(e) => {
        if (keyboardFocus(e.target)) showNow()
      }}
      onBlur={hide}
    >
      {trigger}
      {linkDescription && (
        <span id={id} role="tooltip" hidden>
          {content}
        </span>
      )}
      {open && <TooltipBubble text={content} bubbleRef={bubbleRef} setFloating={setFloating} style={floatingStyles} onEnter={showNow} onLeave={hide} />}
    </span>
  )
}

function TooltipBubble({
  text,
  bubbleRef,
  setFloating,
  style,
  onEnter,
  onLeave,
}: {
  text: string
  bubbleRef: { current: HTMLSpanElement | null }
  setFloating: (el: HTMLElement | null) => void
  style: CSSProperties
  onEnter: () => void
  onLeave: () => void
}) {
  useTopLayer(bubbleRef)
  return (
    <span
      ref={(el) => {
        bubbleRef.current = el
        setFloating(el)
      }}
      popover="manual"
      className="k-tooltip"
      aria-hidden="true"
      data-open=""
      style={style}
      onPointerEnter={onEnter}
      onPointerLeave={onLeave}
    >
      {text}
    </span>
  )
}
