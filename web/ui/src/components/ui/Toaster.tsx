// Kongtrol UI kit — Toaster (snackbar) · v0.1.0 · 2026-11-18
//
// Renders the head of data/toast.ts's queue: one at a time (M3 snackbars),
// bottom center above the mobile bar, in a live region that is always in the
// DOM so screen readers hear each new toast (WCAG 4.1.3 Status Messages).
// Errors go to the assertive region and stay until dismissed; others close by
// themselves, with the timer paused on hover/focus (WCAG 2.2.1).
import { useEffect, useRef, useState } from 'react'
import { usePreferences } from '../../app/preferences-context'
import { toasts, useToasts, type Toast } from '../../data/toast'
import { IconButton } from './Button'
import { AlertIcon, CheckIcon, CloseIcon, InfoIcon } from './icons'

function ToastView({ toast }: { toast: Toast }) {
  const { t } = usePreferences()
  const [paused, setPaused] = useState(false)
  const remaining = useRef(toast.duration)
  const started = useRef(0)

  useEffect(() => {
    if (!toast.duration || paused) return
    started.current = Date.now()
    const timer = setTimeout(() => toasts.dismiss(toast.id), remaining.current)
    return () => {
      clearTimeout(timer)
      remaining.current -= Date.now() - started.current
    }
  }, [toast.id, toast.duration, paused])

  const icon = toast.tone === 'bad' ? <AlertIcon /> : toast.tone === 'ok' ? <CheckIcon /> : <InfoIcon />
  return (
    <div
      className={`k-toast k-toast--${toast.tone}`}
      onPointerEnter={() => setPaused(true)}
      onPointerLeave={() => setPaused(false)}
      onFocus={() => setPaused(true)}
      onBlur={() => setPaused(false)}
    >
      <span className="k-toast__icon">{icon}</span>
      <div className="k-toast__text">
        <strong>{toast.message}</strong>
        {toast.detail && <span>{toast.detail}</span>}
      </div>
      {toast.action && (
        <button
          type="button"
          className="k-btn k-btn--text k-btn--sm"
          onClick={() => {
            toasts.dismiss(toast.id)
            toast.action?.run()
          }}
        >
          <span className="k-btn__label">{toast.action.label}</span>
        </button>
      )}
      <IconButton label={t('kit.dismiss')} icon={<CloseIcon />} onClick={() => toasts.dismiss(toast.id)} />
    </div>
  )
}

export function Toaster() {
  const list = useToasts()
  const head = list[0]
  const error = head?.tone === 'bad'
  // Both live regions stay mounted so a new toast is a content change the
  // screen reader announces; errors go to the assertive one.
  return (
    <div className="k-toaster">
      <div role="status" aria-live="polite">
        {head && !error && <ToastView key={head.id} toast={head} />}
      </div>
      <div aria-live="assertive">{head && error && <ToastView key={head.id} toast={head} />}</div>
    </div>
  )
}
