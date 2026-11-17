import type { PublicIPState } from '../../api/hooks'
import { usePreferences } from '../../app/preferences-context'
import { ErrorNotice } from '../ui/ErrorNotice'
import { StatusPill } from '../ui/StatusPill'

/**
 * Network egress: the public IP from the opt-in lookup. Off by default; while
 * off it shows "—" and a hint, and nothing leaves the machine.
 */
export function PublicIPCard({ state }: { state: PublicIPState }) {
  const { t } = usePreferences()
  const { status, error, busy, actionError, justEnabled, setEnabled, check, reload } = state
  const on = !!status?.public_ip_check_enabled
  const host = status?.public_ip_check_host || 'api.ipify.org'
  const ip = (on && status?.leak_check?.public_ip) || '—'

  return (
    <section className="card" aria-labelledby="publicip-title" aria-busy={!status && !error}>
      <div className="card-head">
        <h2 id="publicip-title">{t('publicip.title')}</h2>
        {status && (
          <StatusPill tone={on ? 'ok' : 'idle'}>{on ? t('publicip.pill.on') : t('publicip.pill.off')}</StatusPill>
        )}
      </div>
      {error ? (
        <ErrorNotice error={error} onRetry={reload} />
      ) : (
        <>
          <p role="status">
            {t('publicip.label')} <code>{ip}</code>
          </p>
          {status && !on && <p className="muted small">{t('publicip.hint')}</p>}
          {on && <p className="muted small">{t('publicip.notice', { host })}</p>}
          {status && (
            <div className="notice-actions">
              {on && (
                <button type="button" className="btn" onClick={check} disabled={busy}>
                  {busy ? t('publicip.checking') : t('publicip.check')}
                </button>
              )}
              <button type="button" className="btn ghost" onClick={() => setEnabled(!on)} disabled={busy}>
                {on ? t('publicip.turnOff') : t('publicip.turnOn')}
              </button>
            </div>
          )}
        </>
      )}
      <div aria-live="polite">{justEnabled && on && <p className="notice ok">{t('publicip.enabled')}</p>}</div>
      {actionError && <ErrorNotice error={actionError} />}
    </section>
  )
}
