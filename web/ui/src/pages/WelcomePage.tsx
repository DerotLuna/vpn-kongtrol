// "Bienvenida / Primeros pasos" — first page migrated to the new UI (phase 1).
// Read-only except the helper install button (existing POST /api/v1/helper/install).
import { useHelper, usePublicIP, useTunnels } from '../api/hooks'
import { usePreferences } from '../app/preferences-context'
import { AppShell } from '../components/layout/AppShell'
import { ExternalLink } from '../components/ui/ExternalLink'
import { ActionTiles } from '../components/welcome/ActionTiles'
import { HelperCard } from '../components/welcome/HelperCard'
import { PublicIPCard } from '../components/welcome/PublicIPCard'
import { TunnelsCard } from '../components/welcome/TunnelsCard'
import { guideLink } from '../links'

export function WelcomePage() {
  const { t } = usePreferences()
  const helper = useHelper()
  const tunnels = useTunnels()
  const publicIP = usePublicIP()
  const noVPNs = tunnels.tunnels !== null && tunnels.tunnels.length === 0

  return (
    <AppShell {...(helper.status?.cli_version ? { version: helper.status.cli_version } : {})}>
      <section className="hero" aria-labelledby="welcome-title">
        <p className="eyebrow">{t('welcome.eyebrow')}</p>
        <h1 id="welcome-title">{t('welcome.title')}</h1>
        <p className="lead">{t('welcome.lead')}</p>
        <ExternalLink href={guideLink('quickstart')}>{t('help.quickstart')}</ExternalLink>
      </section>

      <div className="grid-2">
        <HelperCard helper={helper} />
        <TunnelsCard state={tunnels} />
        <PublicIPCard state={publicIP} />
      </div>

      <ActionTiles emphasizeAdd={noVPNs} />
    </AppShell>
  )
}
