// Kongtrol UI — Tráfico (/traffic) · v1.0.0 · 2026-11-18
//
// Where each destination goes out and how much each VPN carries
// (docs/dashboard-parity.md §3.1 traffic rows). Three blocks, plain words
// first and technical detail on demand:
//  1. "¿Por dónde sale?" lookup + the rules it filters (with resolved IPs
//     and their expiry).
//  2. One card per connected VPN: live received / sent / latency charts,
//     stability (metrics history) and what goes through that tunnel.
//  3. System routes + base network for developers, collapsed by default.
// Live data comes from the shared tunnels feed; route tables poll every
// 30 s and refresh right after a tunnel goes up or down.
import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useLocation } from 'react-router'
import { usePreferences } from '../app/preferences-context'
import { PageHeader } from '../components/layout/PageHeader'
import { LookupCard } from '../components/traffic/LookupCard'
import { RecentCard } from '../components/traffic/RecentCard'
import { SystemRoutes } from '../components/traffic/SystemRoutes'
import { TunnelTrafficCard } from '../components/traffic/TunnelTrafficCard'
import { SectionError } from '../components/traffic/SectionError'
import { EmptyState, Skeleton } from '../components/ui/States'
import { tunnelState, useTunnelsQuery } from '../data/resources'
import {
  useMetricsHistoryQuery,
  usePoliciesQuery,
  useRefreshRoutesOnTunnelChange,
  useTrafficRecorder,
  useTunnelRoutesQuery,
} from '../data/traffic'
import '../components/traffic/traffic.css'

export interface TrafficPageProps {
  /** Test hook: called whenever a tunnel card renders. */
  onCardRender?: (name: string) => void
}

export function TrafficPage({ onCardRender }: TrafficPageProps = {}) {
  const { t } = usePreferences()
  useTrafficRecorder()
  useRefreshRoutesOnTunnelChange()
  const tunnels = useTunnelsQuery()
  const tunnelRoutes = useTunnelRoutesQuery()
  const policies = usePoliciesQuery()
  const history = useMetricsHistoryQuery()
  const list = useMemo(() => tunnels.data ?? [], [tunnels.data])

  // Routes panels start open; the set holds the ones the user closed, so the
  // choice survives live re-renders and tunnels that come and go.
  const [closed, setClosed] = useState<ReadonlySet<string>>(() => new Set())
  const toggle = useCallback((name: string) => {
    setClosed((prev) => {
      const next = new Set(prev)
      if (next.has(name)) next.delete(name)
      else next.add(name)
      return next
    })
  }, [])

  const up = list.filter((tn) => {
    const s = tunnelState(tn)
    return s === 'connected' || s === 'no_response'
  })
  const down = list.filter((tn) => !up.includes(tn))

  // Deep link from Inicio: /traffic#<tunnelAnchor(name)> scrolls to that card.
  const { hash } = useLocation()
  const ready = up.length > 0
  useEffect(() => {
    if (!hash || !ready) return
    document.getElementById(decodeURIComponent(hash.slice(1)))?.scrollIntoView({ block: 'start' })
  }, [hash, ready])

  return (
    <>
      <PageHeader title={t('nav.traffic')} lead={t('page.traffic.lead')} />
      <div className="tr-page">
        <LookupCard tunnels={list} routes={tunnelRoutes.data} />
        <RecentCard />

        <section className="tr-tunnels" aria-labelledby="tr-tunnels-title" aria-busy={tunnels.isPending || undefined}>
          <h2 id="tr-tunnels-title" className="tr-section-title">
            {t('traffic.tunnels.title')}
          </h2>
          {tunnels.isPending && <Skeleton lines={4} label={t('traffic.tunnels.loading')} />}
          {tunnels.isError && !tunnels.data && <SectionError error={tunnels.error} onRetry={() => void tunnels.refetch()} />}
          {tunnels.data && up.length === 0 && (
            <EmptyState
              title={t('traffic.tunnels.empty.title')}
              body={list.length === 0 ? t('traffic.tunnels.empty.noVpns') : t('traffic.tunnels.empty.body')}
              action={
                <Link className="k-btn k-btn--outlined k-btn--md" to={list.length === 0 ? '/profiles' : '/'}>
                  <span className="k-btn__label">{list.length === 0 ? t('nav.profiles') : t('traffic.tunnels.empty.action')}</span>
                </Link>
              }
            />
          )}
          {tunnelRoutes.isError && !tunnelRoutes.data && up.length > 0 && (
            <SectionError error={tunnelRoutes.error} onRetry={() => void tunnelRoutes.refetch()} />
          )}
          {up.length > 0 && (
            <div className="tr-tunnels__grid">
              {up.map((tn) => (
                <TunnelTrafficCard
                  key={tn.Name}
                  tunnel={tn}
                  routes={tunnelRoutes.data?.tunnels.find((r) => r.name === tn.Name)}
                  allRoutes={tunnelRoutes.data}
                  history={history.data?.[tn.Name]}
                  open={!closed.has(tn.Name)}
                  onToggle={toggle}
                  onRender={onCardRender}
                />
              ))}
            </div>
          )}
          {up.length > 0 && down.length > 0 && (
            <p className="tr-note">{t('traffic.tunnels.off', { names: down.map((tn) => tn.Name).join(', ') })}</p>
          )}
        </section>

        <SystemRoutes policies={policies.data ?? []} tunnelRoutes={tunnelRoutes.data} />
      </div>
    </>
  )
}
