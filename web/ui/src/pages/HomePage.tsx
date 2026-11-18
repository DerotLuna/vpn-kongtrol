// Kongtrol UI — Inicio (/) · v1.0.1 · 2026-11-18
//
// The full Home of docs/dashboard-parity.md (section "/ Inicio"): helper
// banner, every VPN with its state, traffic and actions (live), notices
// from the daemon, protection summary, the normal way to the internet and
// groups quick launch. Data only through src/data (shared cache + live
// feed); this page fetches nothing on its own.
import { useTunnels } from '../api/hooks'
import { useLatencyRecorder } from '../data/home'
import { usePreferences } from '../app/preferences-context'
import { PageHeader } from '../components/layout/PageHeader'
import { GroupsCard } from '../components/home/GroupsCard'
import { HelperBanner } from '../components/home/HelperBanner'
import { NetworkCard } from '../components/home/NetworkCard'
import { NoticesCard } from '../components/home/NoticesCard'
import { SecurityCard } from '../components/home/SecurityCard'
import { TunnelsCard } from '../components/home/TunnelsCard'
import '../components/home/home.css'

export function HomePage() {
  const { t } = usePreferences()
  // REST snapshot then the live feed; "Kongtrol no responde" at the first
  // failed request when nothing is cached (api/hooks.ts useTunnels).
  const tunnels = useTunnels()
  useLatencyRecorder()
  // Daemon notices toast from the shell, on every page (AppShell).

  return (
    <>
      <PageHeader title={t('nav.home')} lead={t('home.lead')} />
      <HelperBanner />
      <TunnelsCard
        tunnels={tunnels.tunnels ?? undefined}
        error={tunnels.error}
        onRefresh={tunnels.reload}
      />
      <div className="home-grid">
        <NoticesCard />
        <SecurityCard />
        <NetworkCard />
        <GroupsCard />
      </div>
    </>
  )
}
