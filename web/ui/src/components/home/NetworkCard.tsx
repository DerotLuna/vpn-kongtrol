// Kongtrol UI — base network egress (Inicio) · v1.0.0 · 2026-11-18
//
// Where traffic that matches no rule goes out: the default route not on a
// tunnel, its gateway, the local IPs and the public IP from the opt-in
// lookup (GET /api/v1/network/overview never contacts an external service).
// Read-only; the full route table lives in /traffic.
import { usePreferences } from '../../app/preferences-context'
import { useNetworkOverview } from '../../data/home'
import { IconButton } from '../ui/Button'
import { Card } from '../ui/Card'
import { RefreshIcon } from '../ui/icons'
import { ErrorState, Skeleton } from '../ui/States'

export function NetworkCard() {
  const { t } = usePreferences()
  const query = useNetworkOverview()
  const ov = query.data
  const route = ov?.default_routes?.[0]
  const ips = ov?.local_ips ?? []

  return (
    <Card
      className="home-net"
      title={t('home.net.title')}
      aside={<IconButton label={t('home.net.refresh')} icon={<RefreshIcon />} onClick={() => void query.refetch()} />}
      busy={query.isPending}
    >
      <p className="muted home-lead">{t('home.net.lead')}</p>
      {query.isPending && <Skeleton lines={4} label={t('home.net.loading')} />}
      {query.isError && !ov && <ErrorState error={query.error} onRetry={() => void query.refetch()} />}
      {ov && (
        <>
          <dl className="home-facts">
            <div>
              <dt>{t('home.net.tunnels')}</dt>
              <dd className="num">{ov.connected_tunnels}</dd>
            </div>
            <div>
              <dt>{t('home.net.egress')}</dt>
              <dd className="mono">{route ? t('home.net.egressValue', { iface: route.interface, metric: route.metric }) : '—'}</dd>
            </div>
            <div>
              <dt>{t('home.net.gateway')}</dt>
              <dd className="mono">{route ? route.gateway || t('home.net.onLink') : '—'}</dd>
            </div>
            <div>
              <dt>{t('home.net.localIps')}</dt>
              <dd className="mono">{ips.length > 0 ? ips.join(', ') : '—'}</dd>
            </div>
            <div>
              <dt>{t('home.net.publicIp')}</dt>
              <dd className="mono">
                {ov.public_ip_check_enabled ? (ov.public_ip || '—') : <span className="home-dim">{t('home.net.publicIpOff')}</span>}
              </dd>
            </div>
          </dl>
          <p className="muted small">{t('home.net.note')}</p>
        </>
      )}
    </Card>
  )
}
