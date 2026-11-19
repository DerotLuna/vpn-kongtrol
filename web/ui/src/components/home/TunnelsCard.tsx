// Kongtrol UI — "Tus VPN" table (Inicio) · v1.1.0 · 2026-11-19
//
// One row per tunnel (DataList: a real table, stacked cards on narrow
// screens). Rows are memoized and the cache keeps unchanged tunnels
// identical, so a live tick re-renders only the rows that changed; the
// cells that read other state (pending action, routes, latency samples)
// subscribe on their own. Four States: skeleton on first load, the error
// with what to do, the first-run step when there are no VPNs, the table.
import { useMemo } from 'react'
import type { TunnelMetrics } from '../../api/types'
import { usePreferences } from '../../app/preferences-context'
import { bulkTargets, useTunnelActions } from '../../data/home'
import { summarizeTunnels } from '../../data/resources'
import { guideLink } from '../../links'
import { Button, IconButton } from '../ui/Button'
import { Card } from '../ui/Card'
import { DataList, type Column } from '../ui/DataList'
import { ExternalLink } from '../ui/ExternalLink'
import { PlugIcon, RefreshIcon } from '../ui/icons'
import { EmptyState, ErrorState, Skeleton } from '../ui/States'
import { formatBytes, formatDuration } from './format'
import { LatencySparkline } from './Sparkline'
import { TunnelRoutesRow } from './TunnelRoutes'
import { TunnelActionsCell, TunnelStatusCell } from './TunnelRow'
import { AddButton, AddRuleIcon } from '../add/entries'

interface Props {
  tunnels: TunnelMetrics[] | undefined
  error: unknown
  onRefresh: () => void
  /** Test hook (DataList.onRowRender). */
  onRowRender?: (key: string) => void
}

const Dash = () => <span className="home-dim">—</span>

// Module function: stable, so memoized rows stay skipped (DataList.rowDetail).
const routesDetail = (tn: TunnelMetrics, colSpan: number) => <TunnelRoutesRow name={tn.Name} colSpan={colSpan} />

export function TunnelsCard({ tunnels, error, onRefresh, onRowRender }: Props) {
  const { t, lang } = usePreferences()
  const run = useTunnelActions()

  const columns = useMemo<Column<TunnelMetrics>[]>(
    () => [
      { id: 'name', header: t('home.col.name'), cell: (tn) => <span className="home-name">{tn.Name}</span> },
      { id: 'status', header: t('home.col.status'), cell: (tn) => <TunnelStatusCell tn={tn} /> },
      {
        id: 'ip',
        header: t('home.col.ip'),
        cell: (tn) => (tn.AssignedIP ? <span className="mono">{tn.AssignedIP}</span> : <Dash />),
      },
      {
        id: 'uptime',
        header: t('home.col.uptime'),
        numeric: true,
        cell: (tn) => (tn.Status === 'connected' && tn.UptimeSeconds > 0 ? formatDuration(tn.UptimeSeconds) : <Dash />),
      },
      { id: 'sent', header: t('home.col.sent'), numeric: true, cell: (tn) => formatBytes(tn.BytesSent, lang) ?? <Dash /> },
      {
        id: 'received',
        header: t('home.col.received'),
        numeric: true,
        cell: (tn) => formatBytes(tn.BytesReceived, lang) ?? <Dash />,
      },
      {
        id: 'latency',
        header: t('home.col.latency'),
        numeric: true,
        cell: (tn) =>
          tn.Status === 'connected' ? (
            <span className="home-latency">
              {tn.LatencyMS > 0 ? <span>{tn.LatencyMS}&#8239;ms</span> : <Dash />}
              <LatencySparkline name={tn.Name} />
            </span>
          ) : (
            <Dash />
          ),
      },
      {
        id: 'actions',
        header: t('home.col.actions'),
        hideHeader: true,
        cell: (tn) => (
          <span className="home-row-actions">
            <TunnelActionsCell tn={tn} />
            <AddRuleIcon via={tn.Name} />
          </span>
        ),
      },
    ],
    [t, lang],
  )

  const list = tunnels ?? []
  const sum = summarizeTunnels(list)
  const toConnect = bulkTargets(list, 'connect')
  const toDisconnect = bulkTargets(list, 'disconnect')

  const head = (
    <div className="home-toolbar">
      {list.length > 1 && (
        <>
          <Button variant="tonal" size="sm" disabled={toConnect.length === 0} onClick={() => toConnect.forEach((n) => run(n, 'connect'))}>
            {t('home.tunnels.connectAll')}
          </Button>
          <Button
            variant="outlined"
            size="sm"
            disabled={toDisconnect.length === 0}
            onClick={() => toDisconnect.forEach((n) => run(n, 'disconnect'))}
          >
            {t('home.tunnels.disconnectAll')}
          </Button>
        </>
      )}
      <IconButton label={t('home.tunnels.refresh')} icon={<RefreshIcon />} onClick={onRefresh} />
    </div>
  )

  return (
    <Card
      className="home-tunnels"
      title={
        <>
          {t('tunnels.title')}
          {tunnels && list.length > 0 && (
            <span className="home-count" role="status">
              {t('tunnels.summary', { connected: sum.connected, total: sum.total })}
            </span>
          )}
        </>
      }
      aside={head}
      busy={!tunnels && !error}
    >
      {!tunnels && !error && <Skeleton lines={4} label={t('home.tunnels.loading')} />}
      {!tunnels && !!error && <ErrorState error={error} onRetry={onRefresh} />}
      {tunnels && list.length === 0 && (
        <EmptyState
          icon={<PlugIcon />}
          title={t('home.firstRun.title')}
          body={t('home.firstRun.body')}
          action={
            <div className="home-empty-actions">
              <AddButton req={{ mode: 'menu' }}>{t('home.firstRun.action')}</AddButton>
              <ExternalLink href={guideLink('quickstart')}>{t('help.quickstart')}</ExternalLink>
            </div>
          }
        />
      )}
      {tunnels && list.length > 0 && (
        <>
          {sum.connected === 0 && <p className="muted">{t('tunnels.allOff')}</p>}
          <DataList
            caption={t('home.tunnels.caption')}
            columns={columns}
            rows={list}
            rowKey={(tn) => tn.Name}
            rowDetail={routesDetail}
            {...(onRowRender ? { onRowRender } : {})}
          />
          <p className="home-help">
            <ExternalLink href={guideLink('connect')}>{t('help.connect')}</ExternalLink>
          </p>
        </>
      )}
    </Card>
  )
}
