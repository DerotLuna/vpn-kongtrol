// Kongtrol UI kit — anchored layers (tooltip, menu, listbox, calendar) · v1.0.0 · 2026-11-19
//
// One way to float something next to its trigger (ADR-0067):
// - Position: Floating UI (`@floating-ui/react-dom`, MIT) with flip + shift +
//   size inside an 8px viewport margin, so a layer never leaves the screen
//   (the clipped tooltip at the right edge, owner feedback 2026-11-19), and
//   autoUpdate while it is open (scroll, resize, the trigger moving).
// - Layer: the native Popover API (`popover="manual"` + showPopover()), so
//   the layer sits in the browser's top layer: above a modal <dialog> (the
//   "＋ Agregar" sheet), never clipped by a table or a card, no z-index or
//   portals. Dismissal (Esc, outside press) stays ours, the same in every
//   browser and in tests (jsdom has no Popover API: the calls are guarded).
// - CSP: Floating UI writes left/top through React's style prop and this
//   file writes CSS custom properties with element.style.setProperty. Both
//   are CSSOM, which `style-src 'self'` does not block; only style
//   attributes in markup and <style> elements are (CSP3 §6.1.11; checked in
//   Chromium under the daemon's exact header, docs/research/2026-11-19-floating-layers.md).
import { autoUpdate, flip, offset, shift, size, useFloating, type Placement } from '@floating-ui/react-dom'
import { useEffect, useLayoutEffect, useRef, type RefObject } from 'react'

/** Distance kept from the viewport edges. */
export const VIEWPORT_MARGIN = 8

interface AnchorOptions {
  placement?: Placement
  /** Gap between the trigger and the layer. */
  gap?: number
  /** The layer is at least as wide as the trigger (listboxes). */
  matchWidth?: boolean
  /**
   * Cap the height to the room on the chosen side and scroll inside
   * (lists). False for a panel that changes size while open (the calendar):
   * a capped panel never looks too tall, so flip would never move it; it is
   * flipped and shifted on both axes instead, to stay whole on screen.
   */
  fitHeight?: boolean
}

export function useAnchored(
  open: boolean,
  { placement = 'bottom-start', gap = 6, matchWidth = false, fitHeight = true }: AnchorOptions = {},
) {
  const f = useFloating({
    open,
    placement,
    strategy: 'fixed',
    whileElementsMounted: autoUpdate,
    middleware: [
      offset(gap),
      flip({ padding: VIEWPORT_MARGIN }),
      shift({ padding: VIEWPORT_MARGIN, crossAxis: !fitHeight }),
      size({
        padding: VIEWPORT_MARGIN,
        apply({ availableHeight, availableWidth, rects, elements }) {
          const s = elements.floating.style
          if (fitHeight) s.setProperty('--k-pop-max-h', `${Math.max(140, Math.floor(availableHeight))}px`)
          else s.setProperty('--k-pop-max-h', `calc(100dvh - ${2 * VIEWPORT_MARGIN}px)`)
          s.setProperty('--k-pop-max-w', `${Math.max(160, Math.floor(availableWidth))}px`)
          if (matchWidth) s.setProperty('--k-pop-min-w', `${Math.round(rects.reference.width)}px`)
        },
      }),
    ],
  })
  // Flat callbacks (stable, from Floating UI) so callers never read a refs object during render.
  return { setReference: f.refs.setReference, setFloating: f.refs.setFloating, floatingStyles: f.floatingStyles }
}

/** Puts a mounted `popover` element in the top layer; removes it on unmount. */
export function useTopLayer(ref: RefObject<HTMLElement | null>): void {
  useLayoutEffect(() => {
    const el = ref.current as (HTMLElement & { showPopover?: () => void; hidePopover?: () => void }) | null
    try {
      el?.showPopover?.()
    } catch {
      /* already shown, or not connected yet */
    }
    return () => {
      try {
        el?.hidePopover?.()
      } catch {
        /* removed */
      }
    }
  }, [ref])
}

/** Calls onDismiss on Escape or a press outside every given element while open. */
export function useDismiss(
  open: boolean,
  inside: Array<RefObject<HTMLElement | null>>,
  onDismiss: (reason: 'escape' | 'outside') => void,
): void {
  const cb = useRef(onDismiss)
  useEffect(() => {
    cb.current = onDismiss
  }, [onDismiss])
  const insideRef = useRef(inside)
  useEffect(() => {
    insideRef.current = inside
  })
  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault() // a modal <dialog> under the layer stays open
        e.stopPropagation()
        cb.current('escape')
      }
    }
    const onDown = (e: PointerEvent) => {
      const target = e.target as Node | null
      if (target && insideRef.current.some((r) => r.current?.contains(target))) return
      cb.current('outside')
    }
    document.addEventListener('keydown', onKey, true)
    document.addEventListener('pointerdown', onDown, true)
    return () => {
      document.removeEventListener('keydown', onKey, true)
      document.removeEventListener('pointerdown', onDown, true)
    }
  }, [open])
}

/** Type-ahead: the first item whose label starts with what was typed in the last 600 ms. */
export function useTypeahead(labels: () => string[], onMatch: (index: number) => void) {
  const buf = useRef('')
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)
  useEffect(() => () => clearTimeout(timer.current), [])
  return (key: string, from: number): boolean => {
    if (key.length !== 1 || (key === ' ' && buf.current === '')) return false
    clearTimeout(timer.current)
    buf.current += key.toLocaleLowerCase()
    timer.current = setTimeout(() => (buf.current = ''), 600)
    const list = labels().map((l) => l.toLocaleLowerCase())
    const n = list.length
    // Repeating one letter cycles through the items that start with it.
    const same = buf.current.split('').every((c) => c === buf.current[0])
    const needle = same ? buf.current[0]! : buf.current
    const start = same ? from + 1 : from
    for (let i = 0; i < n; i++) {
      const idx = (start + i) % n
      if (list[idx]!.startsWith(needle)) {
        onMatch(idx)
        return true
      }
    }
    return true
  }
}
