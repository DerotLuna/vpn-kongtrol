// Kongtrol UI — Seguridad: what the kill switch blocks right now · v1.0.0 · 2026-11-18
//
// GET /api/v1/security/blocked (ADR-0042): the destinations (IP ranges)
// blocked now and why — a VPN that dropped, one that carried all traffic,
// or a rule set to block while its VPN is off. Collapsed by default: the
// count is the answer most people need; the list is there to check one
// address. Long lists show the first LIST_MAX and say how many more.
import { usePreferences } from '../../app/preferences-context'
import { useBlockedDestinations } from '../../data/security'
import { Skeleton } from '../ui/States'

const LIST_MAX = 200

export function BlockedList({ enabled }: { enabled: boolean }) {
  const { t } = usePreferences()
  const blocked = useBlockedDestinations(enabled)
  const b = blocked.data
  if (!enabled) return null
  if (!b) return blocked.isPending ? <Skeleton lines={1} label={t('security.ks.blocked.loading')} /> : null
  if (b.count === 0) return <p className="sec-meta">{t('security.ks.blocked.none')}</p>

  const shown = b.prefixes.slice(0, LIST_MAX)
  return (
    <details className="sec-tech sec-blocked">
      <summary>{t('security.ks.blocked.summary', { n: b.count })}</summary>
      <div className="sec-tech__body">
        <p className="sec-meta">{t('security.ks.blocked.what')}</p>
        {b.blocking.length > 0 && <p className="sec-meta">{t('security.ks.blocked.byTunnel', { names: b.blocking.join(', ') })}</p>}
        {b.full_tunnel.length > 0 && <p className="sec-meta">{t('security.ks.fullTunnel', { names: b.full_tunnel.join(', ') })}</p>}
        {b.policy_blocking.length > 0 && (
          <p className="sec-meta">{t('security.ks.policyBlocking', { names: b.policy_blocking.join(', ') })}</p>
        )}
        <ul className="sec-blocked__list" aria-label={t('security.ks.blocked.listLabel')}>
          {shown.map((p) => (
            <li key={p} className="mono">
              {p}
            </li>
          ))}
        </ul>
        {b.count > shown.length && <p className="sec-meta">{t('security.ks.blocked.more', { n: b.count - shown.length })}</p>}
      </div>
    </details>
  )
}
