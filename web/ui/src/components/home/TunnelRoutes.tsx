// Kongtrol UI — "Qué sale por este túnel" disclosure (Inicio) · v1.0.0 · 2026-11-18
//
// Inline disclosure (APG pattern: a button with aria-expanded controlling a
// region), not a modal. The open state lives in the component, so live ticks
// that re-render the row keep it open. One shared request for every row
// (GET /api/v1/tunnels/routes); the full view lives in /traffic.
import { useId, useState } from 'react'
import { Link } from 'react-router'
import { usePreferences } from '../../app/preferences-context'
import { useTunnelRoutes, useTunnelRoutesFor, type TunnelRoute } from '../../data/home'

/** Destinations shown per group before "y N más". */
const CHIPS = 8

interface Group {
  id: string
  label: string
  routes: TunnelRoute[]
}

export function TunnelRoutesDisclosure({ name }: { name: string }) {
  const { t } = usePreferences()
  const tr = useTunnelRoutesFor(name)
  const { data } = useTunnelRoutes()
  const [open, setOpen] = useState(false)
  const id = useId()
  if (!tr || !tr.connected) return null

  const groups: Group[] = []
  for (const r of tr.routes ?? []) {
    const gid = r.origin === 'policy' ? `policy:${r.policy ?? ''}` : r.origin
    let g = groups.find((x) => x.id === gid)
    if (!g) {
      const label =
        r.origin === 'policy'
          ? t('home.routes.origin.policy', { policy: r.policy ?? '' })
          : r.origin === 'server'
            ? t('home.routes.origin.server')
            : t('home.routes.origin.vpn')
      g = { id: gid, label, routes: [] }
      groups.push(g)
    }
    g.routes.push(r)
  }

  const rest = data?.rest_via_tunnel
    ? data.rest_via_tunnel === name
      ? null
      : t('home.routes.restTunnel', { name: data.rest_via_tunnel })
    : data?.rest_via
      ? t('home.routes.rest', { via: data.rest_via })
      : null

  return (
    <div className="home-routes">
      <button
        type="button"
        className="home-routes__toggle"
        aria-expanded={open}
        aria-controls={id}
        onClick={() => setOpen((v) => !v)}
      >
        <svg viewBox="0 0 12 12" aria-hidden="true" className="home-routes__chevron">
          <path d="M4 2.5 7.5 6 4 9.5" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
        {tr.all_traffic ? t('home.routes.toggleAll') : t('home.routes.toggle', { n: tr.total })}
      </button>
      <div id={id} className="home-routes__panel" hidden={!open}>
        {tr.all_traffic && <p>{t('home.routes.all')}</p>}
        {!tr.all_traffic && groups.length === 0 && <p className="muted">{t('home.routes.none')}</p>}
        {groups.map((g) => (
          <div key={g.id} className="home-routes__group">
            <p className="home-routes__origin">
              {g.label} <span className="home-routes__count">({g.routes.length})</span>
            </p>
            <ul className="home-routes__chips">
              {g.routes.slice(0, CHIPS).map((r) => (
                <li key={r.destination} className="mono">
                  {r.destination}
                </li>
              ))}
              {g.routes.length > CHIPS && <li className="home-routes__more">{t('home.routes.more', { n: g.routes.length - CHIPS })}</li>}
            </ul>
          </div>
        ))}
        {!tr.all_traffic && rest && <p className="muted">{rest}</p>}
        <Link to="/traffic">{t('home.routes.seeAll')}</Link>
      </div>
    </div>
  )
}
