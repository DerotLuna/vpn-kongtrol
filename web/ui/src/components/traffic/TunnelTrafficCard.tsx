// Kongtrol UI — one connected tunnel: live throughput, stability, routes · v1.1.0 · 2026-11-24
//
// Memoized: a live tick re-renders only the cards whose tunnel object,
// routes or samples changed (structural sharing + per-tunnel sample store).
// Received and sent are two small charts on one shared scale (small
// multiples), latency a third; the open/closed state of the routes panel
// lives in the page, so it survives every re-render.
// Until two samples exist the three charts are a skeleton of the same size
// (aria-busy, "Cargando gráficos" for screen readers). Latency comes from the
// daemon's health probe (TCP to the VPN server, else the tunnel's DNS): a
// loader only while a first reading is still expected (LATENCY_WAIT_MS of
// samples), then a static "Sin medición para esta VPN" with why — never an
// endless spinner (ADR-0114).
import { memo } from 'react'
import type { TunnelMetrics } from '../../api/types'
import { usePreferences } from '../../app/preferences-context'
import {
  formatBytes,
  formatClock,
  formatDuration,
  formatRate,
  tunnelAnchor,
  useTrafficSeries,
  type ProfileHistory,
  type TrafficSeries,
  type TunnelRoutes,
  type TunnelRoutesResponse,
} from '../../data/traffic'
import { tunnelState } from '../../data/resources'
import { IconButton } from '../ui/Button'
import { InfoIcon } from '../ui/icons'
import { SkeletonBlock } from '../ui/States'
import { StatusPill } from '../ui/StatusPill'
import { MiniChart } from './MiniChart'
import { TunnelRoutesPanel } from './TunnelRoutesPanel'

interface Props {
  tunnel: TunnelMetrics
  routes: TunnelRoutes | undefined
  allRoutes: TunnelRoutesResponse | undefined
  history: ProfileHistory | undefined
  open: boolean
  onToggle: (name: string) => void
  /** Test hook: called on every render of this card. */
  onRender?: ((name: string) => void) | undefined
}

const TABLE_ROWS = 10

function TunnelTrafficCardImpl({ tunnel, routes, allRoutes, history, open, onToggle, onRender }: Props) {
  onRender?.(tunnel.Name)
  const { t, lang } = usePreferences()
  const series = useTrafficSeries(tunnel.Name)
  const state = tunnelState(tunnel)
  const anchor = tunnelAnchor(tunnel.Name)
  const panelId = `${anchor}-routes`
  const headId = `${anchor}-title`

  return (
    <section className="tr-tunnel" id={anchor} aria-labelledby={headId}>
      <header className="tr-tunnel__head">
        <h3 id={headId} className="tr-tunnel__name mono">
          {tunnel.Name}
        </h3>
        <StatusPill tone={state === 'no_response' ? 'caution' : 'ok'}>
          {t(state === 'no_response' ? 'tunnels.status.noResponse' : 'tunnels.status.connected')}
        </StatusPill>
        {(routes?.interface || tunnel.InterfaceName) && (
          <span className="tr-tunnel__iface">{t('traffic.tunnel.iface', { iface: routes?.interface || tunnel.InterfaceName })}</span>
        )}
      </header>

      <Charts series={series} name={tunnel.Name} />

      <p className="tr-tunnel__totals">
        {t('traffic.tunnel.totals', { rx: formatBytes(tunnel.BytesReceived, lang), tx: formatBytes(tunnel.BytesSent, lang) })}
      </p>

      {history && (
        <dl className="tr-stats" aria-label={t('traffic.stats.label', { name: tunnel.Name })}>
          <div>
            <dt>{t('traffic.stats.uptime')}</dt>
            <dd>{formatDuration(history.uptime_seconds)}</dd>
          </div>
          <div>
            <dt>{t('traffic.stats.reconnects')}</dt>
            <dd>{history.reconnects}</dd>
          </div>
          <div>
            <dt>{t('traffic.stats.jitter')}</dt>
            <dd>{t('traffic.stats.ms', { n: history.jitter_ms })}</dd>
          </div>
        </dl>
      )}

      {series.points.length > 0 && <DataTable series={series} name={tunnel.Name} />}

      {routes && (
        <div className="tr-tunnel__routes">
          <button
            type="button"
            className="tr-disclosure"
            aria-expanded={open}
            aria-controls={panelId}
            onClick={() => onToggle(tunnel.Name)}
          >
            <span className="tr-disclosure__chev" aria-hidden="true" />
            {routes.all_traffic ? t('traffic.routes.toggleAll') : t('traffic.routes.toggle', { n: routes.total })}
          </button>
          {open && <TunnelRoutesPanel routes={routes} all={allRoutes} id={panelId} />}
        </div>
      )}
    </section>
  )
}

function spanText(points: readonly { at: number }[]): string {
  const ms = (points.at(-1)?.at ?? 0) - (points[0]?.at ?? 0)
  return formatDuration(Math.max(1, Math.round(ms / 1000)))
}

/**
 * How long (in samples' time) a first latency reading is still expected.
 * The daemon probes each connected tunnel within one watchdog pass (5 s)
 * and gives a probe up to `monitor.health_check.timeout` (10 s by default);
 * 45 s covers that with room. A tunnel already connected when the page
 * opened has its reading in the first frame.
 */
export const LATENCY_WAIT_MS = 45_000

/** Same size as the three charts, so nothing jumps when they arrive. */
function ChartsSkeleton({ name }: { name: string }) {
  const { t } = usePreferences()
  return (
    <div className="tr-charts tr-charts--loading" data-tunnel={name} aria-busy="true">
      <span className="visually-hidden" role="status">
        {t('traffic.tunnel.loadingCharts')}
      </span>
      {[0, 1, 2].map((i) => (
        <div key={i} className="tr-chart tr-chart--skeleton" aria-hidden="true">
          <div className="tr-chart__head">
            <SkeletonBlock className="tr-skel tr-skel--title" />
            <SkeletonBlock className="tr-skel tr-skel--now" />
          </div>
          <SkeletonBlock className="tr-skel tr-skel--plot" />
          <SkeletonBlock className="tr-skel tr-skel--x" />
          <SkeletonBlock className="tr-skel tr-skel--line" />
          <SkeletonBlock className="tr-skel tr-skel--line tr-skel--short" />
        </div>
      ))}
    </div>
  )
}

function Charts({ series, name }: { series: TrafficSeries; name: string }) {
  const { t, lang } = usePreferences()
  const pts = series.points
  if (pts.length < 2) return <ChartsSkeleton name={name} />
  const waited = (pts.at(-1)?.at ?? 0) - (pts[0]?.at ?? 0) >= LATENCY_WAIT_MS
  const times = pts.map((p) => p.at)
  const rx = pts.map((p) => p.rx)
  const tx = pts.map((p) => p.tx)
  const max = Math.max(1, ...rx, ...tx)
  const rate = (v: number) => formatRate(v, lang)
  const span = spanText(pts)
  const lat = series.latency
  const latMs = lat.map((l) => l.ms)
  const ms = (v: number) => t('traffic.stats.ms', { n: Math.round(v) })
  return (
    <div className="tr-charts" data-tunnel={name}>
      <MiniChart
        title={t('traffic.tunnel.received')}
        times={times}
        values={rx}
        max={max}
        format={rate}
        summary={t('traffic.chart.summary', {
          series: t('traffic.tunnel.received'),
          now: rate(rx.at(-1) ?? 0),
          max: rate(Math.max(...rx)),
          span,
        })}
      />
      <MiniChart
        title={t('traffic.tunnel.sent')}
        times={times}
        values={tx}
        max={max}
        format={rate}
        summary={t('traffic.chart.summary', {
          series: t('traffic.tunnel.sent'),
          now: rate(tx.at(-1) ?? 0),
          max: rate(Math.max(...tx)),
          span,
        })}
      />
      {lat.length >= 2 ? (
        <MiniChart
          title={t('traffic.tunnel.latency')}
          times={lat.map((l) => l.at)}
          values={latMs}
          max={Math.max(...latMs) * 1.1}
          format={ms}
          summary={t('traffic.chart.summaryLatency', {
            now: ms(latMs.at(-1) ?? 0),
            min: ms(Math.min(...latMs)),
            max: ms(Math.max(...latMs)),
            span: spanText(lat),
          })}
        />
      ) : (
        <div className="tr-chart tr-chart--empty">
          <span className="tr-chart__title">{t('traffic.tunnel.latency')}</span>
          <span className="tr-chart__now">{lat.length === 1 ? ms(latMs[0]!) : '—'}</span>
          {lat.length === 1 ? (
            // One reading so far (it is recorded only when it changes).
            <span className="tr-chart__note">{t('traffic.tunnel.latencyLast')}</span>
          ) : waited ? (
            // The probe got no round trip from this VPN: say so, statically.
            <span className="tr-chart__note tr-chart__none">
              {t('traffic.tunnel.latencyNone')}
              <IconButton size="sm" label={t('traffic.tunnel.latencyWhy')} tip={t('traffic.tunnel.latencyWhy.tip')} icon={<InfoIcon />} />
            </span>
          ) : (
            // A first reading is still expected: a loader, with the words.
            <span className="tr-chart__wait" role="status">
              <span className="k-spinner" aria-hidden="true" />
              <span className="visually-hidden">{t('traffic.tunnel.noLatency')}</span>
            </span>
          )}
        </div>
      )}
    </div>
  )
}

function DataTable({ series, name }: { series: TrafficSeries; name: string }) {
  const { t, lang } = usePreferences()
  const rows = series.points.slice(-TABLE_ROWS).reverse()
  return (
    <details className="tr-table">
      <summary>{t('traffic.chart.table')}</summary>
      <table>
        <caption className="visually-hidden">{t('traffic.chart.tableCaption', { name })}</caption>
        <thead>
          <tr>
            <th scope="col">{t('traffic.chart.col.time')}</th>
            <th scope="col" className="num">{t('traffic.tunnel.received')}</th>
            <th scope="col" className="num">{t('traffic.tunnel.sent')}</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((p) => (
            <tr key={p.at}>
              <th scope="row" className="mono">{formatClock(p.at, lang)}</th>
              <td className="num">{formatRate(p.rx, lang)}</td>
              <td className="num">{formatRate(p.tx, lang)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </details>
  )
}

export const TunnelTrafficCard = memo(TunnelTrafficCardImpl)
