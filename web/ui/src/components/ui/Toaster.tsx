// Kongtrol UI kit — Toaster (snackbar) · v0.2.0 · 2026-11-18
//
// Renders the head of data/toast.ts's queue: one at a time (M3 snackbars),
// bottom center above the mobile bar, in a live region that is always in the
// DOM so screen readers hear each new toast (WCAG 4.1.3 Status Messages).
// Errors go to the assertive region and stay until dismissed; others close by
// themselves, with the timer paused on hover/focus (WCAG 2.2.1).
// A toast rises in and fades out (--dur-exit) before it leaves the queue;
// with reduced motion it simply appears and disappears.
import { useCallback, useEffect, useRef, useState } from 'react'
import { usePreferences } from '../../app/preferences-context'
import { toasts, useToasts, type Toast } from '../../data/toast'
import { motionMs } from '../../theme/motion'
import { IconButton } from './Button'
import { AlertIcon, CheckIcon, CloseIcon, InfoIcon } from './icons'

function ToastView({ toast }: { toast: Toast }) {
  const { t } = usePreferences()
  const [paused, setPaused] = useState(false)
  const remaining = useRef(toast.duration)
  const started = useRef(0)
  const [leaving, setLeaving] = useState(false)
  // Play the exit, then drop the toast from the queue.
  const leave = useCallback(() => {
    const ms = motionMs('--dur-exit')
    if (ms === 0) {
      toasts.dismiss(toast.id)
      return
    }
    setLeaving(true)
    setTimeout(() => toasts.dismiss(toast.id), ms)
  }, [toast.id])

  useEffect(() => {
    if (!toast.duration || paused) return
    started.current = Date.now()
    const timer = setTimeout(leave, remaining.current)
    return () => {
      clearTimeout(timer)
      remaining.current -= Date.now() - started.current
    }
  }, [toast.duration, paused, leave])

  const icon = toast.tone === 'bad' ? <AlertIcon /> : toast.tone === 'ok' ? <CheckIcon /> : <InfoIcon />
  return (
    <div
      className={`k-toast k-toast--${toast.tone}`}
      data-leaving={leaving || undefined}
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
            leave()
            toast.action?.run()
          }}
        >
          <span className="k-btn__label">{toast.action.label}</span>
        </button>
      )}
      <IconButton label={t('kit.dismiss')} icon={<CloseIcon />} onClick={leave} />
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
