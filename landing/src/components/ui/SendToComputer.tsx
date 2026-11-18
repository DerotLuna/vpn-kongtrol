import { ReactNode, useEffect, useState } from 'react'
import { useCanShare } from '../../hooks/useVisitor'
import { Lang } from '../../i18n'
import { CTA_COPY, SendIcon } from './downloadCta'
import './download-cta.css'

interface Props {
  lang: Lang
  /** full: notice + button + status; compact: icon button for the header */
  variant?: 'full' | 'compact'
  /** extra actions next to the button (e.g. the hero's "Ver en GitHub") */
  children?: ReactNode
  className?: string
}

type Status = 'idle' | 'copied' | 'failed'

// Phones and tablets can't run Kongtrol, so instead of a download they get a
// way to move this page to their computer: the system share sheet (Web Share
// API — AirDrop, Nearby Share, mail, chat…) or, where it doesn't exist, a
// "copy link" with a visible confirmation that is also announced (role=status).
// https://developer.mozilla.org/en-US/docs/Web/API/Navigator/share
// https://developer.mozilla.org/en-US/docs/Web/API/Clipboard/writeText
export default function SendToComputer({ lang, variant = 'full', children, className = '' }: Props) {
  const t = CTA_COPY[lang]
  const canShare = useCanShare()
  const [status, setStatus] = useState<Status>('idle')

  // the confirmation stays long enough to read, then clears
  useEffect(() => {
    if (status === 'idle') return
    const id = window.setTimeout(() => setStatus('idle'), 6000)
    return () => window.clearTimeout(id)
  }, [status])

  const send = async () => {
    const url = `${location.origin}${location.pathname}`
    if (canShare) {
      try {
        await navigator.share({ title: t.shareTitle, url })
        return
      } catch (err) {
        // the visitor closed the share sheet: nothing to do
        if (err instanceof DOMException && err.name === 'AbortError') return
        // anything else (no share target, not allowed): fall back to copying
      }
    }
    try {
      await navigator.clipboard.writeText(url)
      setStatus('copied')
    } catch {
      setStatus('failed')
    }
  }

  const label = canShare ? t.send : t.copy
  const message = status === 'copied' ? t.copied : status === 'failed' ? t.copyFailed : ''

  if (variant === 'compact') {
    return (
      <span className={`stc-compact ${className}`}>
        <button type="button" className="stc-icon-btn" onClick={send} aria-label={label} title={label}>
          <SendIcon size={18} />
        </button>
        <span className={`stc-bubble${message ? ' show' : ''}`} role="status">{message}</span>
      </span>
    )
  }

  return (
    <div className={`stc ${className}`} data-send-to-computer="">
      <p className="stc-note">{t.mobileNotice}</p>
      <div className="stc-actions">
        <button type="button" className="btn-download stc-btn" onClick={send}>
          <SendIcon size={18} />
          {/* both labels share one grid cell, so the button keeps its width
              when canShare flips after hydration (no layout shift); the
              unused one is visibility:hidden, never read */}
          <span className="stc-label">
            <span className={canShare ? undefined : 'is-off'}>{t.send}</span>
            <span className={canShare ? 'is-off' : undefined}>{t.copy}</span>
          </span>
        </button>
        {children}
      </div>
      <p className={`stc-status${status === 'failed' ? ' failed' : ''}`} role="status"><span key={status} className="copy-label is-fb">{message}</span></p>
    </div>
  )
}
