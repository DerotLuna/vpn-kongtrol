// Kongtrol UI — Seguridad: kill switch per profile · v1.0.0 · 2026-11-18
//
// Decision in docs/dashboard-parity.md §2: the editable override lives here
// (/security#profile-overrides); /profiles shows it read-only. Each change
// is optimistic and rolls back with a toast if the daemon refuses. A
// native <select> (keyboard and screen readers for free; the classic
// panel's custom select leaked listeners: parity §5.1 #10).
import { useMemo } from 'react'
import { Link } from 'react-router'
import { usePreferences } from '../../app/preferences-context'
import { useOverride, useProfiles, type KillSwitchOverride, type ProfileSummary } from '../../data/security'
import { Card } from '../ui/Card'
import { DataList, type Column } from '../ui/DataList'
import { EmptyState, ErrorState, Skeleton } from '../ui/States'
import { StatusPill } from '../ui/StatusPill'

export function OverridesCard({ globalOn }: { globalOn: boolean }) {
  const { t } = usePreferences()
  const profiles = useProfiles()
  const override = useOverride()
  const { mutate } = override

  const columns = useMemo<Column<ProfileSummary>[]>(
    () => [
      {
        id: 'name',
        header: t('security.override.col.profile'),
        cell: (p) => <span className="mono">{p.name}</span>,
      },
      {
        id: 'setting',
        header: t('security.override.col.setting'),
        cell: (p) => (
          <select
            className="k-input k-select sec-select"
            aria-label={t('security.override.label', { name: p.name })}
            value={p.kill_switch_override || 'inherit'}
            onChange={(e) => mutate({ name: p.name, override: e.target.value as KillSwitchOverride })}
          >
            <option value="inherit">
              {t('security.override.inherit', { state: globalOn ? t('security.on') : t('security.off') })}
            </option>
            <option value="on">{t('security.override.on')}</option>
            <option value="off">{t('security.override.off')}</option>
          </select>
        ),
      },
      {
        id: 'effect',
        header: t('security.override.col.effect'),
        cell: (p) => {
          const o = p.kill_switch_override || 'inherit'
          const protectedNow = o === 'on' || (o === 'inherit' && globalOn)
          return (
            <StatusPill tone={protectedNow ? 'on' : 'idle'}>
              {t(protectedNow ? 'security.override.effect.on' : 'security.override.effect.off')}
            </StatusPill>
          )
        },
      },
    ],
    [t, mutate, globalOn],
  )

  return (
    <Card title={t('security.override.title')} className="sec-card sec-card--wide" busy={profiles.isPending}>
      <span id="profile-overrides" className="sec-anchor" />
      <p className="sec-lead">{t('security.override.lead')}</p>
      {profiles.isPending ? (
        <Skeleton lines={3} label={t('security.loading')} />
      ) : profiles.error ? (
        <ErrorState error={profiles.error} onRetry={() => void profiles.refetch()} />
      ) : (
        <DataList
          caption={t('security.override.caption')}
          columns={columns}
          rows={profiles.data ?? []}
          rowKey={(p) => p.name}
          empty={
            <EmptyState
              title={t('security.override.empty.title')}
              body={t('security.override.empty.body')}
              action={
                <Link className="k-btn k-btn--outlined k-btn--md" to="/profiles">
                  <span className="k-btn__label">{t('security.override.empty.action')}</span>
                </Link>
              }
            />
          }
        />
      )}
    </Card>
  )
}
