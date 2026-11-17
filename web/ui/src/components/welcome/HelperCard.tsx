import type { HelperState } from '../../api/hooks'
import { usePreferences } from '../../app/preferences-context'
import { guideLink } from '../../links'
import { ErrorNotice } from '../ui/ErrorNotice'
import { ExternalLink } from '../ui/ExternalLink'
import { StatusPill } from '../ui/StatusPill'

/** Step 1: is the privileged helper ready? Offers the one-click install. */
export function HelperCard({ helper }: { helper: HelperState }) {
  const { t } = usePreferences()
  const { status, error, installing, installError, installed, install, reload } = helper

  let pill = <StatusPill tone="idle">{t('helper.loading')}</StatusPill>
  let line = t('helper.loading')
  if (status) {
    if (status.elevated) {
      pill = <StatusPill tone="ok">{t('helper.pill.notNeeded')}</StatusPill>
      line = t('helper.elevated')
    } else if (status.needs_install) {
      pill = <StatusPill tone="warn">{t('helper.pill.missing')}</StatusPill>
      line = t('helper.needsInstall')
    } else {
      pill = <StatusPill tone="ok">{t('helper.pill.ready')}</StatusPill>
      line = t('helper.ready')
    }
  }

  return (
    <section className="card" aria-labelledby="helper-title" aria-busy={!status && !error}>
      <div className="card-head">
        <h2 id="helper-title">
          <span className="step-num" aria-hidden="true">
            1
          </span>
          {t('helper.title')}
        </h2>
        {status && pill}
      </div>
      <p className="muted small">{t('helper.what')}</p>

      {error ? (
        <ErrorNotice error={error} onRetry={reload} />
      ) : (
        <p role="status">{line}</p>
      )}

      {status?.helper_version && !status.needs_install && (
        <p className="muted small">{t('helper.version', { version: status.helper_version })}</p>
      )}

      {status?.needs_install && (
        <button type="button" className="btn" onClick={install} disabled={installing}>
          {t('helper.install')}
        </button>
      )}
      <div aria-live="polite">
        {installing && <p className="notice">{t('helper.installing')}</p>}
        {installed && !installing && <p className="notice ok">{t('helper.installed')}</p>}
      </div>
      {installError && <ErrorNotice error={installError} />}

      <p className="help">
        <ExternalLink href={guideLink('install')}>{t('help.helper')}</ExternalLink>
      </p>
    </section>
  )
}
