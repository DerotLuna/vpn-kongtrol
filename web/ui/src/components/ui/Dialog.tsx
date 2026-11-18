// Kongtrol UI kit — Dialog / Sheet · v0.1.0 · 2026-11-18
//
// Native <dialog> + showModal(): the browser makes the rest of the page
// inert, traps focus, closes on Esc and restores focus to the opener (MDN,
// HTML spec), so no focus-trap library is needed. Use it to confirm a
// destructive action (Cancel + one action) or for a short focused task —
// never as the first idea (inline first). `variant="sheet"` docks to the
// bottom on phones (Apple HIG sheets, M3 bottom sheets) and is a centered
// dialog on wide screens.
import { useEffect, useId, useRef, type ReactNode } from 'react'
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
  /** alertdialog for confirmations that interrupt (WAI-ARIA APG). */
  alert?: boolean
}

export function Dialog({ open, onClose, title, description, children, actions, variant = 'dialog', alert = false }: DialogProps) {
  const ref = useRef<HTMLDialogElement>(null)
  const id = useId()
  const { t } = usePreferences()

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
      className={`k-dialog k-dialog--${variant}`}
      role={alert ? 'alertdialog' : undefined}
      aria-labelledby={`${id}-t`}
      aria-describedby={description ? `${id}-d` : undefined}
      // Esc (cancel) and form-less close both report back so state stays in sync.
      onCancel={(e) => {
        e.preventDefault()
        onClose()
      }}
      onClick={(e) => {
        // Click on the backdrop (the dialog element itself) closes a sheet.
        if (variant === 'sheet' && e.target === e.currentTarget) onClose()
      }}
    >
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
    </dialog>
  )
}
