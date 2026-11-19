// Kongtrol UI — what goes through one tunnel, grouped by origin · v1.0.0 · 2026-11-18
//
// Classic panel's per-tunnel drawer (app.js:770-800): server / rule / VPN
// groups with counts and chips, "all your traffic", "nothing yet" and the
// "everything else →" line.
import { usePreferences } from '../../app/preferences-context'
import { AddButton } from '../add/entries'
import { groupTunnelRoutes, type TunnelRoutes, type TunnelRoutesResponse } from '../../data/traffic'

function restLine(t: ReturnType<typeof usePreferences>['t'], tr: TunnelRoutesResponse | undefined): string {
  if (tr?.rest_via_tunnel) return t('traffic.routes.restTunnel', { name: tr.rest_via_tunnel })
  if (tr?.rest_via) return t('traffic.routes.rest', { iface: tr.rest_via })
  return t('traffic.routes.restUnknown')
}

export function TunnelRoutesPanel({ routes, all, id }: { routes: TunnelRoutes; all: TunnelRoutesResponse | undefined; id: string }) {
  const { t } = usePreferences()
  const groups = groupTunnelRoutes(routes.routes ?? [])
  return (
    <div className="tr-routes" id={id}>
      {routes.all_traffic && <p className="tr-routes__all">{t('traffic.routes.allTraffic')}</p>}
      {groups.length === 0 && !routes.all_traffic && <p className="muted">{t('traffic.routes.none')}</p>}
      {groups.map((g) => {
        const n = g.destinations.length
        const title =
          g.origin === 'server'
            ? t('traffic.routes.server', { n })
            : g.origin === 'policy'
              ? t('traffic.routes.policy', { name: g.policy, n })
              : t('traffic.routes.vpn', { n })
        return (
          <div key={`${g.origin}|${g.policy}|${g.destinations[0]}`} className="tr-routes__group">
            <p className="tr-routes__title">
              <span className={`tr-tag tr-tag--${g.origin}`}>{t(`traffic.origin.${g.origin}`)}</span> {title}
            </p>
            {g.origin === 'server' && <p className="tr-routes__explain">{t('traffic.routes.serverExplain')}</p>}
            <ul className="tr-chips" aria-label={title}>
              {g.destinations.map((d) => (
                <li key={d} className="mono">
                  {d}
                </li>
              ))}
            </ul>
          </div>
        )
      })}
      <p className="tr-routes__rest">{restLine(t, all)}</p>
      <AddButton req={{ mode: 'rule', via: routes.name }} variant="text">
        {t('add.siteVia', { vpn: routes.name })}
      </AddButton>
    </div>
  )
}
