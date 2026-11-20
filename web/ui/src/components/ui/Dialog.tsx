// Kongtrol UI kit — Dialog / Sheet · v0.3.0 · 2026-11-19
//
// Native <dialog> + showModal(): the browser makes the rest of the page
// inert, traps focus, closes on Esc and restores focus to the opener (MDN,
// HTML spec), so no focus-trap library is needed. Use it to confirm a
// destructive action (Cancel + one action) or for a short focused task —
// never as the first idea (inline first). `variant="sheet"` docks to the
// bottom on phones (Apple HIG sheets, M3 bottom sheets) and is a centered
// dialog on wide screens. Kit layers opened inside (select lists, calendar,
// tooltips: React Aria overlays, ADR-0068) portal into this <dialog>
// instead of <body>, so they stay in its top layer and are not made inert.
import { useCallback, useEffect, useId, useRef, type ReactNode } from 'react'
import { UNSAFE_PortalProvider } from 'react-aria/PortalProvider'
import { usePreferences } from '../../app/preferences-context'
import { IconButton } from './Button'
import { CloseIcon } from './icons'

interface DialogProps {
  open: boolean
  onClose: () => void
  title: string
  description?: ReactNode
  children?: ReactNode
  /** Buttons; the first in DOM order is the safe one (Cancel). */
  actions?: ReactNode
  variant?: 'dialog' | 'sheet'
  /** Extra class on the <dialog> (e.g. a wider sheet). */
  className?: string
  /** alertdialog for confirmations that interrupt (WAI-ARIA APG). */
  alert?: boolean
}

export function Dialog({ open, onClose, title, description, children, actions, variant = 'dialog', alert = false, className = '' }: DialogProps) {
  const ref = useRef<HTMLDialogElement>(null)
  const id = useId()
  const { t } = usePreferences()
  const container = useCallback(() => ref.current, [])
  const downOnBackdrop = useRef(false)

  useEffect(() => {
    const el = ref.current
    if (!el) return
    if (open && !el.open) {
      if (typeof el.showModal === 'function') el.showModal()
      else el.setAttribute('open', '')
    } else if (!open && el.open) {
      if (typeof el.close === 'function') el.close()
      else el.removeAttribute('open')
    }
  }, [open])

  return (
    <dialog
      ref={ref}
      className={`k-dialog k-dialog--${variant} ${className}`.trim()}
      role={alert ? 'alertdialog' : undefined}
      aria-labelledby={`${id}-t`}
      aria-describedby={description ? `${id}-d` : undefined}
      // Esc (cancel) and form-less close both report back so state stays in sync.
      onCancel={(e) => {
        e.preventDefault()
        onClose()
      }}
      onPointerDown={(e) => {
        downOnBackdrop.current = e.target === e.currentTarget
      }}
      onClick={(e) => {
        // Click on the backdrop (the dialog element itself) closes a sheet:
        // pressed and released there. A press on a control that opens a list
        // ends on the list's underlay, and the browser then targets the
        // click at the <dialog> (their common ancestor): that is no backdrop
        // click.
        if (variant === 'sheet' && e.target === e.currentTarget && downOnBackdrop.current) onClose()
      }}
    >
      <UNSAFE_PortalProvider getContainer={container}>
      <div className="k-dialog__body">
        <header className="k-dialog__head">
          {/* showModal() focuses the autofocus element: the title, so screen
              readers start there and the close button's tooltip stays shut. */}
          <h2 id={`${id}-t`} tabIndex={-1} autoFocus>
            {title}
          </h2>
          <IconButton label={t('kit.close')} icon={<CloseIcon />} onClick={onClose} />
        </header>
        {description && (
          <div id={`${id}-d`} className="k-dialog__desc">
            {description}
          </div>
        )}
        {children}
        {actions && <footer className="k-dialog__actions">{actions}</footer>}
      </div>
      </UNSAFE_PortalProvider>
    </dialog>
  )
}
