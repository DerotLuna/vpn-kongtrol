// Kongtrol UI — system routes + base network, for developers · v1.0.0 · 2026-11-18
//
// Collapsed by default (technical detail on demand); the OS route table is
// read only while it is open (slow on Windows). Classic panel: active
// routes card (index.html:142, app.js:803-890) and base network card
// (index.html:97, app.js:852). Rows are memoized DataList rows keyed by
// interface + destination, so a refresh re-renders only routes that changed.
import { useId, useMemo, useState } from 'react'
import { usePreferences } from '../../app/preferences-context'
import {
  effectiveFilter,
  filterRoutes,
  routeFilterOptions,
  routeInfo,
  routeOriginIndex,
  trafficKeys,
  useNetworkOverviewQuery,
  useRoutesQuery,
  type FilterOption,
  type Policy,
  type RouteInfo,
  type RouteRow,
  type TunnelRoutesResponse,
} from '../../data/traffic'
import { useQueryClient } from '@tanstack/react-query'
import { Button } from '../ui/Button'
import { DataList, type Column } from '../ui/DataList'
import { SelectField } from '../ui/Field'
import { RefreshIcon } from '../ui/icons'
import { Skeleton } from '../ui/States'
import { SectionError } from './SectionError'

interface Props {
  policies: readonly Policy[]
  tunnelRoutes: TunnelRoutesResponse | undefined
}

export function SystemRoutes({ policies, tunnelRoutes }: Props) {
  const { t } = usePreferences()
  const [open, setOpen] = useState(false)
  const bodyId = useId()
  return (
    <section className="tr-system k-card" aria-labelledby="tr-system-title">
      <header className="k-card__head">
        <div>
          <h2 id="tr-system-title" className="k-card__title">
            {t('traffic.system.title')}
          </h2>
          <p className="muted small tr-system__lead">{t('traffic.system.lead')}</p>
        </div>
        <button
          type="button"
          className="tr-disclosure tr-disclosure--btn"
          aria-expanded={open}
          aria-controls={bodyId}
          onClick={() => setOpen((o) => !o)}
        >
          <span className="tr-disclosure__chev" aria-hidden="true" />
          {open ? t('traffic.system.hide') : t('traffic.system.show')}
        </button>
      </header>
      {open && (
        <div id={bodyId} className="tr-system__body">
          <NetworkOverviewPanel />
          <RoutesTable policies={policies} tunnelRoutes={tunnelRoutes} />
        </div>
      )}
    </section>
  )
}

function NetworkOverviewPanel() {
  const { t } = usePreferences()
  const q = useNetworkOverviewQuery(true)
  const o = q.data
  return (
    <div className="tr-net" aria-busy={q.isFetching || undefined}>
      <h3 className="tr-subtitle">{t('traffic.net.title')}</h3>
      {q.isPending && <Skeleton lines={3} label={t('traffic.net.loading')} />}
      {q.isError && !o && <SectionError error={q.error} onRetry={() => void q.refetch()} />}
      {o && (
        <>
          <dl className="tr-kv">
            <div>
              <dt>{t('traffic.net.egress')}</dt>
              <dd className="mono">
                {o.default_routes?.[0]
                  ? t('traffic.net.egressValue', { iface: o.default_routes[0].interface || '—', metric: o.default_routes[0].metric })
                  : '—'}
              </dd>
            </div>
            <div>
              <dt>{t('traffic.net.gateway')}</dt>
              <dd className="mono">{o.default_routes?.[0] ? o.default_routes[0].gateway || t('traffic.system.onLink') : '—'}</dd>
            </div>
            <div>
              <dt>{t('traffic.net.localIps')}</dt>
              <dd className="mono">{o.local_ips?.length ? o.local_ips.join(', ') : '—'}</dd>
            </div>
            <div>
              <dt>{t('traffic.net.connected')}</dt>
              <dd className="mono">{o.connected_tunnels}</dd>
            </div>
            <div>
              <dt>{t('traffic.net.publicIp')}</dt>
              <dd className="mono">{o.public_ip_check_enabled ? o.public_ip || '—' : t('traffic.net.publicIpOff')}</dd>
            </div>
          </dl>
          <p className="small muted">{t('traffic.net.note')}</p>
        </>
      )}
    </div>
  )
}

interface RouteView {
  key: string
  route: RouteRow
  info: RouteInfo | null
}

function optionLabel(o: FilterOption, t: ReturnType<typeof usePreferences>['t']): string {
  switch (o.kind) {
    case 'all':
      return t('traffic.system.filter.all')
    case 'default':
      return t('traffic.system.filter.default')
    case 'profile':
      return t('traffic.system.filter.profile', { name: o.name })
    case 'origin':
      return t('traffic.system.filter.origin', { name: t(`traffic.origin.${o.name as 'server' | 'policy' | 'vpn'}`) })
    case 'policy':
      return t('traffic.system.filter.policy', { name: o.name })
  }
}

// Identity cache: same route + same info → same view object, so memoized
// rows skip re-rendering when a refresh brings the same route back.
const viewCache = new WeakMap<RouteRow, RouteView>()
function viewOf(route: RouteRow, info: RouteInfo | null): RouteView {
  const cached = viewCache.get(route)
  if (cached && cached.info?.profile === info?.profile && cached.info?.origin === info?.origin && cached.info?.policy === info?.policy) {
    return cached
  }
  const v = { key: `${route.interface}|${route.destination}|${route.gateway}|${route.metric}`, route, info }
  viewCache.set(route, v)
  return v
}

function RoutesTable({ policies, tunnelRoutes }: Props) {
  const { t } = usePreferences()
  const client = useQueryClient()
  const q = useRoutesQuery(true)
  const [filter, setFilter] = useState('all')
  const options = useMemo(() => routeFilterOptions(policies, tunnelRoutes), [policies, tunnelRoutes])
  const current = effectiveFilter(filter, options)
  const idx = useMemo(() => routeOriginIndex(tunnelRoutes), [tunnelRoutes])
  const all = q.data
  const rows = useMemo(
    () => (all ? filterRoutes(all, current, idx).map((r) => viewOf(r, routeInfo(r, idx))) : []),
    [all, current, idx],
  )
  const columns = useRouteColumns()

  const refresh = () => {
    void q.refetch()
    void client.invalidateQueries({ queryKey: trafficKeys.tunnelRoutes })
  }

  return (
    <div className="tr-sysroutes" aria-busy={q.isFetching || undefined}>
      <div className="tr-sysroutes__bar">
        <SelectField
          label={t('traffic.system.filter')}
          value={current}
          onChange={setFilter}
          options={options.map((o) => ({ value: o.value, label: optionLabel(o, t) }))}
        />
        <Button variant="outlined" icon={<RefreshIcon />} onClick={refresh} loading={q.isRefetching}>
          {t('traffic.system.refresh')}
        </Button>
      </div>
      {q.isPending && <Skeleton lines={5} label={t('traffic.system.loading')} />}
      {q.isError && !all && <SectionError error={q.error} onRetry={() => void q.refetch()} />}
      {all && (
        <>
          <p className="tr-note" role="status">
            {t('traffic.system.count', { shown: rows.length, total: all.length })}
          </p>
          <DataList
            caption={t('traffic.system.caption')}
            columns={columns}
            rows={rows}
            rowKey={(r) => r.key}
            empty={<p className="muted">{t('traffic.system.empty')}</p>}
          />
        </>
      )}
    </div>
  )
}

function useRouteColumns(): Column<RouteView>[] {
  const { t } = usePreferences()
  return useMemo(
    () => [
      { id: 'dest', header: t('traffic.system.col.dest'), cell: (v) => <span className="mono">{v.route.destination}</span> },
      {
        id: 'tag',
        header: t('traffic.system.col.tag'),
        cell: (v) => {
          if (v.route.is_default) return <span className="tr-tag">{t('traffic.system.defaultTag')}</span>
          if (v.info)
            return (
              <span className="tr-tags">
                <span className={`tr-tag tr-tag--${v.info.origin}`}>{t(`traffic.origin.${v.info.origin}`)}</span>
                {v.info.policy && <span className="tr-tag mono">{v.info.policy}</span>}
              </span>
            )
          if (v.route.policy_name) return <span className="tr-tag mono">{v.route.policy_name}</span>
          return <span className="muted">—</span>
        },
      },
      {
        id: 'via',
        header: t('traffic.system.col.via'),
        cell: (v) => (
          <span className="mono">
            {v.info ? `${v.info.profile} · ${v.route.interface || v.route.gateway || '—'}` : v.route.interface || v.route.gateway || '—'}
          </span>
        ),
      },
      {
        id: 'gw',
        header: t('traffic.system.col.gateway'),
        cell: (v) => <span className="mono">{v.route.gateway || t('traffic.system.onLink')}</span>,
      },
      { id: 'metric', header: t('traffic.system.col.metric'), numeric: true, cell: (v) => v.route.metric },
    ],
    [t],
  )
}
