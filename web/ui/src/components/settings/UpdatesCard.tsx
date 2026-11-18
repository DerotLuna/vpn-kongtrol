import type { UpdatesState } from '../../api/lifecycle-hooks'
import { usePreferences } from '../../app/preferences-context'
import { ErrorNotice } from '../ui/ErrorNotice'
import { ExternalLink } from '../ui/ExternalLink'
import { StatusPill } from '../ui/StatusPill'

// feature:update_check
/**
 * Opt-in new version notice (ADR-0032). Off by default; while off nothing
 * leaves the machine. It only tells the user how to update, never installs.
 */
export function UpdatesCard({ state }: { state: UpdatesState }) {
  const { t } = usePreferences()
  const { status, error, busy, actionError, setEnabled, reload } = state
  const on = !!status?.check_enabled
  const host = status?.host || 'api.github.com'

  return (
    <section className="card" aria-labelledby="updates-title" aria-busy={!status && !error}>
      <div className="card-head">
        <h2 id="updates-title">{t('updates.title')}</h2>
        {status && <StatusPill tone={on ? 'on' : 'idle'}>{on ? t('updates.pill.on') : t('updates.pill.off')}</StatusPill>}
      </div>
      {error ? (
        <ErrorNotice error={error} onRetry={reload} />
      ) : (
        <>
          <p>{t('updates.what')}</p>
          <p className="muted small">{t('updates.privacy', { host })}</p>
          <div aria-live="polite">
            {on && status?.newer && status.latest ? (
              <p className="notice ok">
                {t('updates.newer', { latest: status.latest, current: status.current })}
                {status.url && <ExternalLink href={status.url}>{t('updates.releaseNotes')}</ExternalLink>}
              </p>
            ) : on && status?.checked_at ? (
              <p className="muted small">{t('updates.upToDate', { current: status.current })}</p>
            ) : on ? (
              <p className="muted small">{t('updates.notChecked')}</p>
            ) : null}
          </div>
          {status && (
            <div className="notice-actions">
              <button type="button" className="btn ghost" onClick={() => setEnabled(!on)} disabled={busy}>
                {on ? t('updates.turnOff') : t('updates.turnOn')}
              </button>
            </div>
          )}
        </>
      )}
      {actionError && <ErrorNotice error={actionError} />}
    </section>
  )
}
