// Seguridad (/security) · v1.0.0 · 2026-11-18
//
// Parity with the classic security.html + the security part of
// settings.html (docs/dashboard-parity.md §3.3, §3.4). Everything reads the
// shared cache (src/data/security.ts); the security status refreshes every
// 5 s, so states follow the daemon without a reload.
import { useState } from 'react'
import { usePreferences } from '../app/preferences-context'
import { PageHeader } from '../components/layout/PageHeader'
import { DNSCard } from '../components/security/DNSCard'
import { KillSwitchCard } from '../components/security/KillSwitchCard'
import { OverridesCard } from '../components/security/OverridesCard'
import { PublicIPCheckCard } from '../components/security/PublicIPCheckCard'
import { TunnelDownCard } from '../components/security/TunnelDownCard'
import '../components/security/security.css'
import { Button } from '../components/ui/Button'
import { RefreshIcon } from '../components/ui/icons'
import { ErrorState, Skeleton } from '../components/ui/States'
import { useRefreshSecurity, useSecurityState } from '../data/security'

export function SecurityPage() {
  const { t } = usePreferences()
  const status = useSecurityState()
  const refresh = useRefreshSecurity()
  const [refreshing, setRefreshing] = useState(false)
  const st = status.data

  return (
    <>
      <PageHeader title={t('nav.security')} lead={t('page.security.lead')}>
        <div className="sec-head-actions">
          <Button
            variant="outlined"
            size="sm"
            icon={<RefreshIcon />}
            loading={refreshing}
            onClick={() => {
              setRefreshing(true)
              void refresh().finally(() => setRefreshing(false))
            }}
          >
            {t('security.refresh')}
          </Button>
        </div>
      </PageHeader>

      {st ? (
        <div className="sec-grid">
          <KillSwitchCard status={st} />
          <OverridesCard globalOn={st.kill_switch_enabled} />
          <DNSCard status={st} />
          <PublicIPCheckCard status={st} />
          <TunnelDownCard blocking={st.kill_switch_status?.policy_blocking ?? []} />
        </div>
      ) : status.error ? (
        <ErrorState error={status.error} onRetry={() => void status.refetch()} />
      ) : (
        <div className="sec-grid" aria-busy="true">
          {[0, 1, 2].map((i) => (
            <section key={i} className={`k-card sec-card${i === 0 ? ' sec-card--wide' : ''}`}>
              <Skeleton lines={4} label={t('security.loading')} />
            </section>
          ))}
        </div>
      )}
    </>
  )
}
