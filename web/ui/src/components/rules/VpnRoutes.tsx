// Reglas — "Rutas que trae cada VPN" (read-only) · v1.0.0 · 2026-11-24
//
// The routes each VPN brings by itself (.ovpn route lines, server-pushed
// routes, WireGuard AllowedIPs; provider apps only as "la app decide"),
// grouped by profile in native disclosures. Kongtrol does not own them, so
// there is no edit or delete; the one action per route opens the shell's
// "＋ Agregar" rule sheet (?add=rule&target=…, ADR-0055). The daemon says
// who wins when a rule overlaps a route (ADR-0107); this only words it.
import { useCallback } from 'react'
import { useAddSheet } from '../../app/addLink'
import { usePreferences } from '../../app/preferences-context'
import { useVpnRoutesQuery } from '../../data/rules'
import type { MessageKey } from '../../i18n/messages'
import { IconButton } from '../ui/Button'
import { Card } from '../ui/Card'
import { ErrorState, Skeleton } from '../ui/States'
import { LockIcon, RouteIcon } from './icons'
import { conflictText, profileSummary, routeRows, rowLabel, ruleTarget, type ProfileOwnRoutes } from './vpnRoutesModel'

export function VpnRoutes() {
  const { t } = usePreferences()
  const q = useVpnRoutesQuery()
  const list = q.data ?? []
  return (
    // feature:vpn_routes
    <Card className="r-vpnroutes" title={t('rules.vpnRoutes.title')} busy={q.isFetching}>
      <p className="r-vpnroutes__lead">
        <LockIcon />
        <span>{t('rules.vpnRoutes.lead')}</span>
      </p>
      {q.isPending ? (
        <Skeleton lines={3} label={t('rules.vpnRoutes.loading')} />
      ) : q.isError ? (
        <ErrorState error={q.error} onRetry={() => void q.refetch()} />
      ) : list.length === 0 ? (
        <p className="muted">{t('rules.vpnRoutes.empty')}</p>
      ) : (
        <div className="r-vpnroutes__list">
          {list.map((p) => (
            <ProfileRoutes key={p.name} p={p} />
          ))}
        </div>
      )}
      <Precedence />
    </Card>
  )
}

function ProfileRoutes({ p }: { p: ProfileOwnRoutes }) {
  const { t } = usePreferences()
  const { open } = useAddSheet()
  const moveElsewhere = useCallback((target: string) => open({ mode: 'rule', target }), [open])
  const rows = routeRows(p)
  const notes: MessageKey[] = []
  if (p.source === 'app') notes.push(p.all_traffic ? 'rules.vpnRoutes.app.all' : 'rules.vpnRoutes.app.other')
  if (p.source === 'unreadable') notes.push('rules.vpnRoutes.unreadable')
  if (p.source === 'config' && rows.length === 0) notes.push('rules.vpnRoutes.noFileRoutes')
  if (p.rules_only) notes.push('rules.vpnRoutes.rulesOnly')
  if (p.server_pending) notes.push('rules.vpnRoutes.serverPending')
  if (p.all_traffic && p.source === 'config' && !p.rules_only) notes.push('rules.vpnRoutes.allTrafficWins')
  return (
    <details className="r-disclosure r-vpnroute">
      <summary>
        <span className="r-vpnroute__name">{p.name}</span>
        <span className="r-vpnroute__meta">· {profileSummary(p, t)}</span>
      </summary>
      {rows.length > 0 && (
        <ul className="r-vpnroute__rows" aria-label={t('rules.vpnRoutes.listLabel', { name: p.name })}>
          {rows.map((row) => {
            const label = rowLabel(row, t)
            return (
              <li key={`${row.kind}|${row.destination}`} className="r-vpnroute__row">
                <span className={row.kind === 'route' ? 'mono r-vpnroute__dest' : 'r-vpnroute__dest'}>{label}</span>
                <span className="r-chip r-chip--static r-vpnroute__origin" data-origin={row.origin}>
                  {t(`rules.vpnRoutes.origin.${row.origin}` as MessageKey)}
                </span>
                {row.kind === 'route' && (
                  <IconButton
                    label={t('rules.vpnRoutes.moveNamed', { target: ruleTarget(row.destination) })}
                    tip={t('rules.vpnRoutes.move')}
                    icon={<RouteIcon />}
                    aria-haspopup="dialog"
                    onClick={() => moveElsewhere(ruleTarget(row.destination))}
                  />
                )}
                {row.conflicts.length > 0 && (
                  <ul className="r-vpnroute__conflicts">
                    {row.conflicts.map((c) => (
                      <li key={`${c.rule}|${c.destination}`}>{conflictText(c, label, t)}</li>
                    ))}
                  </ul>
                )}
              </li>
            )
          })}
        </ul>
      )}
      {notes.map((k) => (
        <p key={k} className="r-note">
          {t(k)}
        </p>
      ))}
    </details>
  )
}

/** "Qué gana": how a destination is decided between rules and VPN routes. */
function Precedence() {
  const { t } = usePreferences()
  return (
    <section className="r-precedence" aria-labelledby="r-precedence-title">
      <h3 id="r-precedence-title">{t('rules.vpnRoutes.precedence.title')}</h3>
      <p>{t('rules.vpnRoutes.precedence.body')}</p>
      <p className="muted">{t('rules.vpnRoutes.precedence.sites')}</p>
      <p className="muted">{t('rules.vpnRoutes.precedence.rules')}</p>
    </section>
  )
}
