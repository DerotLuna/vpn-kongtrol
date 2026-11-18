import type { HelperState } from '../../api/hooks'
import { usePreferences } from '../../app/preferences-context'
import { helperUiState } from '../../data/resources'
import { guideLink } from '../../links'
import { ErrorNotice } from '../ui/ErrorNotice'
import { ExternalLink } from '../ui/ExternalLink'
import { StatusPill } from '../ui/StatusPill'
import { HelperInstallButton, HelperInstallProgress } from '../ui/HelperInstall'

/** Step 1: is the privileged helper ready? Offers the one-click install (or update). */
export function HelperCard({ helper }: { helper: HelperState }) {
  const { t } = usePreferences()
  const { status, error, reload } = helper
  const state = helperUiState(status ?? undefined)

  const view = {
    unknown: { pill: null, line: t('helper.loading') },
    not_needed: { pill: <StatusPill tone="ok">{t('helper.pill.notNeeded')}</StatusPill>, line: t('helper.elevated') },
    ready: { pill: <StatusPill tone="ok">{t('helper.pill.ready')}</StatusPill>, line: t('helper.ready') },
    missing: { pill: <StatusPill tone="caution">{t('helper.pill.missing')}</StatusPill>, line: t('helper.needsInstall') },
    outdated: { pill: <StatusPill tone="caution">{t('helper.pill.outdated')}</StatusPill>, line: t('helper.needsUpdate') },
  }[state]

  return (
    <section className="card" aria-labelledby="helper-title" aria-busy={!status && !error}>
      <div className="card-head">
        <h2 id="helper-title">
          <span className="step-num" aria-hidden="true">
            1
          </span>
          {t('helper.title')}
        </h2>
        {view.pill}
      </div>
      <p className="muted small">{t('helper.what')}</p>

      {error ? <ErrorNotice error={error} onRetry={reload} /> : <p role="status">{view.line}</p>}

      {status?.helper_version && state === 'ready' && (
        <p className="muted small">{t('helper.version', { version: status.helper_version })}</p>
      )}

      {(state === 'missing' || state === 'outdated') && <HelperInstallButton helper={helper} outdated={state === 'outdated'} />}
      <HelperInstallProgress helper={helper} />

      <p className="help">
        <ExternalLink href={guideLink('install')}>{t('help.helper')}</ExternalLink>
      </p>
    </section>
  )
}
