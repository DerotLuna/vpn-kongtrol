// Kongtrol UI kit — Tooltip · v2.0.0 · 2026-11-19
//
// The one tooltip of the panel (every IconButton and the collapsed menu
// use it), built on React Aria Components' TooltipTrigger + Tooltip
// (ADR-0068): opens on hover after a short delay and at once on keyboard
// focus, closes with Esc without moving focus, flips and shifts to stay
// inside the window. React Aria positions it with element.style (CSSOM,
// allowed by the page's CSP) and portals it to <body> — or into the open
// kit Dialog (UNSAFE_PortalProvider in Dialog.tsx), so it stays above a
// modal sheet. Screen readers get the text through aria-describedby (a
// hidden node that is always there) only when it adds something to the
// accessible name: a tooltip that repeats the name (the usual icon button)
// is not linked again, so nothing is read twice.
import { cloneElement, useId, type ReactElement } from 'react'
import { Focusable, Tooltip as AriaTooltip, TooltipTrigger } from 'react-aria-components'

type FocusableChild = Parameters<typeof Focusable>[0]['children']

/** Hover delay before showing (Material: 500 ms; a bit faster for a dense panel). */
export const TOOLTIP_DELAY_MS = 350
const CLOSE_MS = 100

interface TooltipProps {
  content: string
  children: ReactElement<{ 'aria-describedby'?: string; 'aria-label'?: string }>
  /** Describe the trigger (true) or just label it visually (default: when the text differs from aria-label). */
  describe?: boolean
  placement?: 'top' | 'bottom' | 'right'
  /** Off: the trigger keeps its name and no bubble opens (e.g. the expanded menu shows its labels). */
  disabled?: boolean
}

export function Tooltip({ content, children, describe, placement = 'top', disabled = false }: TooltipProps) {
  const id = useId()
  const labelled = children.props['aria-label'] === content
  const linkDescription = describe ?? !labelled
  // React Aria merges its aria-describedby under the child's own props, so
  // the child's value wins: a describing tooltip points at a hidden node that
  // is always there (read by a screen reader's virtual cursor too, not only
  // while the bubble shows); a name-repeating one gets an empty value (no
  // reference), so it is never announced twice.
  const own = children.props['aria-describedby']
  const trigger = cloneElement(children, {
    'aria-describedby': linkDescription ? [own, id].filter(Boolean).join(' ') : (own ?? ''),
  })
  return (
    <TooltipTrigger delay={TOOLTIP_DELAY_MS} closeDelay={CLOSE_MS} isDisabled={disabled}>
      {/* Any element that forwards its ref to a focusable DOM node (button, link). */}
      <Focusable>{trigger as FocusableChild}</Focusable>
      {linkDescription && (
        <span id={id} hidden>
          {content}
        </span>
      )}
      <AriaTooltip className="k-tooltip" placement={placement} offset={6} containerPadding={8}>
        {content}
      </AriaTooltip>
    </TooltipTrigger>
  )
}
