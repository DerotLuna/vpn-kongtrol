// Actividad (/activity) — docs/dashboard-parity.md §3.5 (audit.html) + the
// notices feed the classic panel only showed as passing toasts. Two in-page
// tabs (APG); the open one is in the URL (?tab=audit) so links and Back
// work. The log's own settings sit under the log (IA decision for the row
// "Audit log path, max size MB, HMAC sign switch").
import { useSearchParams } from 'react-router'
import { usePreferences } from '../app/preferences-context'
import { AuditConfig } from '../components/activity/AuditConfig'
import { AuditPanel } from '../components/activity/AuditPanel'
import { EventsPanel } from '../components/activity/EventsPanel'
import { PageHeader } from '../components/layout/PageHeader'
import { Tabs } from '../components/ui/Tabs'
import { useSettingsQuery } from '../data/settings'

type Tab = 'events' | 'audit'

export function ActivityPage() {
  const { t } = usePreferences()
  const [params, setParams] = useSearchParams()
  const tab: Tab = params.get('tab') === 'audit' ? 'audit' : 'events'
  const settings = useSettingsQuery()

  return (
    <div className="k-act-page">
      <PageHeader title={t('nav.activity')} lead={t('page.activity.lead')} />
      <Tabs
        label={t('activity.tabs')}
        value={tab}
        onChange={(id) => setParams(id === 'audit' ? { tab: 'audit' } : {}, { replace: true })}
        items={[
          { id: 'events', label: t('activity.tab.events'), panel: <EventsPanel /> },
          {
            id: 'audit',
            label: t('activity.tab.audit'),
            panel: (
              <div className="k-act-stack">
                <AuditPanel settings={settings.data} settingsError={settings.error} />
                {settings.data && <AuditConfig settings={settings.data} />}
              </div>
            ),
          },
        ]}
      />
    </div>
  )
}
