// Kongtrol UI kit — Skeleton, EmptyState, ErrorState, Banner, Kbd · v0.4.0 · 2026-11-25
//
// Loading: skeletons only on the first load (they reserve the final space,
// no layout shift); later refreshes keep the last data on screen
// (stale-while-revalidate) — NN/g: skeletons for full-page loads under ~10s.
// Empty states teach the next step. Errors = what happened + what to do.
// Banners sit inline above content for page- or app-wide conditions (daemon
// stopped, helper missing), never as a modal.
import type { ReactNode } from 'react'
import { ApiError } from '../../api/client'
import { describeError } from '../../api/errors'
import { guideLink } from '../../links'
import { usePreferences } from '../../app/preferences-context'
import { Button } from './Button'
import { ExternalLink } from './ExternalLink'
import { AlertIcon, CheckIcon, InfoIcon, RefreshIcon } from './icons'

export function Skeleton({ lines = 3, label }: { lines?: number; label: string }) {
  return (
    <div className="k-skeleton" role="status" aria-label={label}>
      {Array.from({ length: lines }, (_, i) => (
        <span key={i} className="k-skeleton__line" aria-hidden="true" />
      ))}
    </div>
  )
}

/**
 * One shimmer shape for a skeleton shaped like the final content (a chart,
 * a card): the caller's class sets its size, so nothing jumps when the real
 * content arrives. Decorative only: the region around it carries aria-busy
 * and the words for screen readers.
 */
export function SkeletonBlock({ className = '' }: { className?: string }) {
  return <span className={`k-skeleton__block ${className}`.trim()} aria-hidden="true" />
}

interface EmptyStateProps {
  title: string
  body: ReactNode
  action?: ReactNode
  icon?: ReactNode
  /** Centred, icon in a round tile: a whole list that is empty on purpose (e.g. Avisos). */
  center?: boolean
}

export function EmptyState({ title, body, action, icon, center = false }: EmptyStateProps) {
  return (
    <div className={center ? 'k-empty k-empty--center' : 'k-empty'}>
      {icon && <span className="k-empty__icon">{icon}</span>}
      <h3>{title}</h3>
      <p>{body}</p>
      {action}
    </div>
  )
}

/**
 * The one error block: what happened + what to do (+ Retry). `help` adds the
 * troubleshooting guide link for errors the user may need to dig into.
 */
export function ErrorState({ error, onRetry, help = false }: { error: unknown; onRetry?: () => void; help?: boolean }) {
  const { t } = usePreferences()
  if (error instanceof ApiError && error.kind === 'unauthorized') return null
  const { message, fix } = describeError(error, t)
  return (
    <div className="k-error-state" role="alert">
      <AlertIcon />
      <div>
        <strong>{message}</strong>
        <p>{fix}</p>
        {help && (
          <p>
            <ExternalLink href={guideLink('trouble')}>{t('help.trouble')}</ExternalLink>
          </p>
        )}
      </div>
      {onRetry && (
        <Button variant="outlined" size="sm" icon={<RefreshIcon />} onClick={onRetry}>
          {t('error.retry')}
        </Button>
      )}
    </div>
  )
}

export type BannerTone = 'info' | 'ok' | 'caution' | 'bad'

interface BannerProps {
  tone: BannerTone
  title: string
  children?: ReactNode
  actions?: ReactNode
  /** alert for urgent conditions (daemon stopped); status otherwise. */
  urgent?: boolean
}

export function Banner({ tone, title, children, actions, urgent = false }: BannerProps) {
  const icon = tone === 'ok' ? <CheckIcon /> : tone === 'info' ? <InfoIcon /> : <AlertIcon />
  return (
    <div className={`k-banner k-banner--${tone}`} role={urgent ? 'alert' : 'status'}>
      <span className="k-banner__icon">{icon}</span>
      <div className="k-banner__text">
        <strong>{title}</strong>
        {children}
      </div>
      {actions && <div className="k-banner__actions">{actions}</div>}
    </div>
  )
}

/** A key or shortcut, e.g. <Kbd>Esc</Kbd>. */
export function Kbd({ children }: { children: ReactNode }) {
  return <kbd className="k-kbd">{children}</kbd>
}
