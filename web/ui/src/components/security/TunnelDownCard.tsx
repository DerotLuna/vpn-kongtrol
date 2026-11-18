// Kongtrol UI — Seguridad: what each rule does when its VPN is off · v1.0.0 · 2026-11-18
//
// AGENTS.md: "a policy whose tunnel is off uses the normal internet unless
// on_tunnel_down says otherwise". Read-only overview (rules are edited in
// /rules): action per rule, which ones the kill switch blocks right now
// (kill_switch_status.policy_blocking), and the latest policy-watch
// warnings (GET /api/v1/events, kind policy_tunnel_down).
import { useMemo } from 'react'
import { Link } from 'react-router'
import { usePreferences } from '../../app/preferences-context'
import { usePolicies, usePolicyEvents, type PolicySummary } from '../../data/security'
import { Card } from '../ui/Card'
import { DataList, type Column } from '../ui/DataList'
import { EmptyState, ErrorState, Skeleton } from '../ui/States'
import { StatusPill } from '../ui/StatusPill'
import { downAction, watchKey } from './model'

export function TunnelDownCard({ blocking }: { blocking: readonly string[] }) {
  const { t, lang } = usePreferences()
  const policies = usePolicies()
  const events = usePolicyEvents()
  const blockingKey = blocking.join('\n')

  const columns = useMemo<Column<PolicySummary>[]>(() => {
    const now = new Set(blockingKey ? blockingKey.split('\n') : [])
    return [
      { id: 'rule', header: t('security.down.col.rule'), cell: (p) => <span className="mono">{p.name}</span> },
      { id: 'via', header: t('security.down.col.via'), cell: (p) => <span className="mono">{p.via}</span> },
      {
        id: 'action',
        header: t('security.down.col.action'),
        cell: (p) => {
          const a = downAction(p.on_tunnel_down)
          return a === 'connect' ? t('security.down.connect', { via: p.via }) : t(a === 'block' ? 'security.down.block' : 'security.down.allow')
        },
      },
      {
        id: 'now',
        header: t('security.down.col.now'),
        cell: (p) =>
          now.has(p.name) ? (
            <StatusPill tone="caution">{t('security.down.blockingNow')}</StatusPill>
          ) : (
            <span className="muted">{t('security.down.fine')}</span>
          ),
      },
    ]
  }, [t, blockingKey])

  const warnings = (events.data ?? []).flatMap((e) => {
    const key = watchKey(e.key)
    if (!key) return []
    const vars = Object.fromEntries((e.args ?? []).map((a, i) => [String(i), a]))
    return [{ seq: e.seq, warn: e.level === 'WARN', time: e.time, text: t(key, vars) }]
  })

  return (
    <Card
      title={t('security.down.title')}
      className="sec-card sec-card--wide"
      busy={policies.isPending}
      aside={
        <Link className="k-btn k-btn--text k-btn--sm" to="/rules">
          <span className="k-btn__label">{t('security.down.edit')}</span>
        </Link>
      }
    >
      <p className="sec-lead">{t('security.down.lead')}</p>
      {policies.isPending ? (
        <Skeleton lines={3} label={t('security.loading')} />
      ) : policies.error ? (
        <ErrorState error={policies.error} onRetry={() => void policies.refetch()} />
      ) : (
        <DataList
          caption={t('security.down.caption')}
          columns={columns}
          rows={policies.data ?? []}
          rowKey={(p) => p.name}
          empty={
            <EmptyState
              title={t('security.down.empty.title')}
              body={t('security.down.empty.body')}
              action={
                <Link className="k-btn k-btn--outlined k-btn--md" to="/rules">
                  <span className="k-btn__label">{t('security.down.empty.action')}</span>
                </Link>
              }
            />
          }
        />
      )}

      <div className="sec-watch">
        <h3>{t('security.watch.title')}</h3>
        {events.isPending ? (
          <Skeleton lines={1} label={t('security.loading')} />
        ) : warnings.length === 0 ? (
          <p className="sec-meta">{t('security.watch.none')}</p>
        ) : (
          <ul className="sec-watch__list">
            {warnings.map((w) => (
              <li key={w.seq} data-warn={w.warn || undefined}>
                <time className="mono" dateTime={w.time}>
                  {new Date(w.time).toLocaleTimeString(lang, { hour: '2-digit', minute: '2-digit' })}
                </time>
                <span>{w.text}</span>
              </li>
            ))}
          </ul>
        )}
      </div>
    </Card>
  )
}
