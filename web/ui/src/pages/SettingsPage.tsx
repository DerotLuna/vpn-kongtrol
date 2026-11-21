// Ajustes (/settings) — docs/dashboard-parity.md §3.4 (General) + the items
// the classic panel lacked: language/theme, Kongtrol on this computer
// (version, helper, config file, panel address, restart, quit), health
// check, privacy (update notice opt-in, public IP → Seguridad), diagnostics
// + export, help links and "Borrar todos mis datos". Security tuning lives
// on /security and the scheduler on /rules/schedule (ADR-0038 §6).
import { useEffect } from 'react'
import { useConfigExport, useDoctor, useHelper } from '../api/hooks'
import { usePurgeData, useUpdates } from '../api/lifecycle-hooks'
import { usePreferences } from '../app/preferences-context'
import { PageHeader } from '../components/layout/PageHeader'
import { DataCard } from '../components/settings/DataCard'
import { HealthSection } from '../components/settings/HealthSection'
import { AboutSection, AppearanceSection, PrivacySection } from '../components/settings/SettingsGroups'
import { SettingsSection } from '../components/settings/SettingsSection'
import { SystemSection } from '../components/settings/SystemSection'
import { UpdatesCard } from '../components/settings/UpdatesCard'
import { ErrorState, Skeleton } from '../components/ui/States'
import { DiagnosticsCard } from '../components/welcome/DiagnosticsCard'
import { useSettingsQuery } from '../data/settings'
import type { MessageKey } from '../i18n/messages'

const TOC: { id: string; label: MessageKey }[] = [
  { id: 'look', label: 'settings.look.title' },
  { id: 'system', label: 'settings.system.title' },
  { id: 'health', label: 'settings.health.title' },
  { id: 'privacy', label: 'settings.privacy.title' },
  { id: 'diagnostics', label: 'diag.title' },
  { id: 'about', label: 'settings.about.title' },
  { id: 'data', label: 'data.title' },
]

export function SettingsPage() {
  const { t } = usePreferences()
  const settings = useSettingsQuery()
  const helper = useHelper()
  const doctor = useDoctor()
  const exporter = useConfigExport()
  const updates = useUpdates()
  const purge = usePurgeData()
  useEffect(() => {
    if (new URLSearchParams(window.location.search).get('section') !== 'diagnostics') return
    const section = document.getElementById('diagnostics')
    section?.focus({ preventScroll: true })
    section?.scrollIntoView?.({ block: 'start' })
  }, [])

  return (
    <div className="k-set-page">
      <PageHeader title={t('nav.settings')} lead={t('page.settings.lead')} />
      <div className="k-set-layout">
        <nav className="k-set-toc" aria-label={t('settings.toc')}>
          <p className="k-set-toc__title">{t('settings.toc')}</p>
          <ul>
            {TOC.map((s) => (
              <li key={s.id}>
                <a href={`#${s.id}`}>{t(s.label)}</a>
              </li>
            ))}
          </ul>
        </nav>
        <div className="k-set-col">
          <AppearanceSection />
          <SystemSection settings={settings.data} helper={helper} />
          {settings.data ? (
            <HealthSection settings={settings.data} />
          ) : (
            <SettingsSection id="health" title={t('settings.health.title')} what={t('settings.health.what')} busy={!settings.error}>
              {settings.error ? (
                <ErrorState error={settings.error} onRetry={() => void settings.refetch()} />
              ) : (
                <Skeleton lines={2} label={t('settings.loading')} />
              )}
            </SettingsSection>
          )}
          <PrivacySection />
          <UpdatesCard state={updates} />
          <div id="diagnostics" className="k-set-anchor" tabIndex={-1}>
            <DiagnosticsCard doctor={doctor} helper={helper} exporter={exporter} />
          </div>
          <AboutSection />
          <div id="data" className="k-set-anchor">
            <DataCard state={purge} />
          </div>
        </div>
      </div>
    </div>
  )
}
