// Inicio — first page migrated to the new UI (phase 1 of issue #19).
// Read-only except the helper install button (existing POST /api/v1/helper/install),
// the public IP opt-in, the diagnostics card (doctor + export, GUI parity) and
// Ajustes (update notice opt-in, "Borrar todos mis datos"). Phase 2 turns it
// into the full Home of docs/dashboard-parity.md.
import { useEffect } from 'react'
import { useConfigExport, useDoctor, useHelper, usePublicIP, useTunnels } from '../api/hooks'
import { usePurgeData, useUpdates } from '../api/lifecycle-hooks'
import { usePreferences } from '../app/preferences-context'
import { ExternalLink } from '../components/ui/ExternalLink'
import { ActionTiles } from '../components/welcome/ActionTiles'
import { DiagnosticsCard } from '../components/welcome/DiagnosticsCard'
import { HelperCard } from '../components/welcome/HelperCard'
import { PublicIPCard } from '../components/welcome/PublicIPCard'
import { TunnelsCard } from '../components/welcome/TunnelsCard'
import { DataCard } from '../components/settings/DataCard'
import { UpdatesCard } from '../components/settings/UpdatesCard'
import { guideLink } from '../links'

export function WelcomePage() {
  const { t } = usePreferences()
  const helper = useHelper()
  const tunnels = useTunnels()
  const publicIP = usePublicIP()
  const doctor = useDoctor()
  const exporter = useConfigExport()
  const updates = useUpdates()
  const purge = usePurgeData()
  const noVPNs = tunnels.tunnels !== null && tunnels.tunnels.length === 0

  useEffect(() => {
    document.title = `${t('nav.home')} · ${t('app.name')}`
  }, [t])

  return (
    <>
      <section className="hero" aria-labelledby="welcome-title">
        <h1 id="welcome-title" tabIndex={-1}>
          {t('welcome.title')}
        </h1>
        <p className="lead">{t('welcome.lead')}</p>
        <ExternalLink href={guideLink('quickstart')}>{t('help.quickstart')}</ExternalLink>
      </section>

      <div className="grid-2">
        <HelperCard helper={helper} />
        <TunnelsCard state={tunnels} />
        <PublicIPCard state={publicIP} />
      </div>

      <DiagnosticsCard doctor={doctor} helper={helper} exporter={exporter} />

      <ActionTiles emphasizeAdd={noVPNs} />

      <section className="settings" aria-labelledby="settings-title">
        <h2 id="settings-title">{t('settings.title')}</h2>
        <div className="grid-2">
          <UpdatesCard state={updates} />
          <DataCard state={purge} />
        </div>
      </section>
    </>
  )
}
