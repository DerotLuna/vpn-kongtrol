import type { ApiError } from '../../api/client'
import { usePreferences } from '../../app/preferences-context'
import { guideLink } from '../../links'
import { ExternalLink } from './ExternalLink'

interface Props {
  error: ApiError
  onRetry?: () => void
  className?: string
}

/**
 * One error = what happened + one "what to do" line (CLAUDE.md user
 * guidance). Uses the daemon's translated message and hint when it sent
 * them; otherwise our own text. Never shows a raw transport error alone.
 */
export function ErrorNotice({ error, onRetry, className = '' }: Props) {
  const { t } = usePreferences()
  let message: string
  let fix: string
  switch (error.kind) {
    case 'offline':
      message = t('error.offline')
      fix = t('error.offline.fix')
      break
    case 'unauthorized':
      message = t('error.unauthorized')
      fix = t('error.unauthorized.fix')
      break
    default:
      message = error.message
      fix = error.hint || t('error.generic.fix')
  }
  return (
    <div className={`notice bad ${className}`.trim()} role="alert">
      <strong>{message}</strong>
      <span>{fix}</span>
      <div className="notice-actions">
        {onRetry && (
          <button type="button" className="btn ghost" onClick={onRetry}>
            {t('error.retry')}
          </button>
        )}
        <ExternalLink href={guideLink('trouble')}>{t('help.trouble')}</ExternalLink>
      </div>
    </div>
  )
}
