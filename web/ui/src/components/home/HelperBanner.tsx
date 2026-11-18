// Kongtrol UI — helper banner on Inicio · v1.0.0 · 2026-11-18
//
// The app-wide notice (AppNotices) skips Inicio so the helper is asked for
// once per page; here it sits above the tunnels, with the shared install
// flow (one mutation key: every install button reacts at once).
import { useHelper } from '../../api/hooks'
import { usePreferences } from '../../app/preferences-context'
import { helperUiState } from '../../data/resources'
import { guideLink } from '../../links'
import { ExternalLink } from '../ui/ExternalLink'
import { HelperInstallButton, HelperInstallProgress } from '../ui/HelperInstall'
import { Banner } from '../ui/States'

export function HelperBanner() {
  const { t } = usePreferences()
  const helper = useHelper()
  const state = helperUiState(helper.status ?? undefined)
  if (state !== 'missing' && state !== 'outdated') {
    // Keep the confirmation visible right after an install from here.
    return helper.installed ? <HelperInstallProgress helper={helper} /> : null
  }
  const outdated = state === 'outdated'
  return (
    <div className="home-banner">
      <Banner
        tone="caution"
        title={outdated ? t('helper.notice.outdated') : t('helper.notice.missing')}
        actions={<HelperInstallButton helper={helper} outdated={outdated} />}
      >
        <p>{outdated ? t('helper.needsUpdate') : t('helper.needsInstall')}</p>
        <p>
          <ExternalLink href={guideLink('install')}>{t('help.helper')}</ExternalLink>
        </p>
        <HelperInstallProgress helper={helper} />
      </Banner>
    </div>
  )
}
