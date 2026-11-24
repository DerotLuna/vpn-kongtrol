// Kongtrol UI — the list of VPN profiles · v1.1.0 · 2026-11-24
//
// One row per profile: which VPN (with "Prioridad N" when set, ADR-0108), its live state (Four States Rule), how it
// works on this computer, sign-in stored, backups, kill switch (read-only:
// it is changed on Seguridad, docs/dashboard-parity.md §2), and Edit /
// Delete. A table on wide screens, cards on phones (DataList). Rows are
// memoized: a live tunnel tick re-renders only the row whose state changed.
import { useMemo } from 'react'
import { Link } from 'react-router'
import type { TunnelMetrics } from '../../api/types'
import { usePreferences } from '../../app/preferences-context'
import type { VPNProfile } from '../../data/profiles'
import type { MessageKey } from '../../i18n/messages'
import { IconButton } from '../ui/Button'
import { DataList, type Column } from '../ui/DataList'
import { StatusPill } from '../ui/StatusPill'
import { Monogram, ProfileState, SupportChip } from './Bits'
import { specOf } from './catalog'
import { EditIcon, TrashIcon } from './icons'

export interface ProfileRow {
  profile: VPNProfile
  tunnel: TunnelMetrics | undefined
  pendingRestart: boolean
}

function credentials(p: VPNProfile): MessageKey {
  if (p.has_username_credential && p.has_password_credential) return 'profiles.cred.userPass'
  if (p.has_username_credential) return 'profiles.cred.userOnly'
  if (p.has_password_credential) return 'profiles.cred.passOnly'
  return 'profiles.cred.none'
}

const OVERRIDE: Record<VPNProfile['kill_switch_override'], MessageKey> = {
  inherit: 'profiles.ks.inheritShort',
  on: 'profiles.ks.onShort',
  off: 'profiles.ks.offShort',
}

interface Props {
  rows: readonly ProfileRow[]
  onEdit: (p: VPNProfile) => void
  onDelete: (p: VPNProfile) => void
}

export function ProfileList({ rows, onEdit, onDelete }: Props) {
  const { t } = usePreferences()
  // Stable columns (DataList memoizes rows on row identity + columns).
  const columns = useMemo<Column<ProfileRow>[]>(
    () => [
      {
        id: 'vpn',
        header: t('profiles.col.vpn'),
        cell: ({ profile }) => {
          const spec = specOf(profile.type)
          // A file shows by its name only; the full path is in the editor.
          const where = profile.server || profile.host || (profile.config ?? '').split(/[\\/]/).pop() || ''
          return (
            <span className="kp-vpn">
              <Monogram spec={spec} />
              <span className="kp-vpn__text">
                <span className="kp-vpn__name mono">{profile.name}</span>
                <span className="kp-vpn__type">
                  {spec.label}
                  {where && <span className="mono kp-vpn__where"> · {where}</span>}
                  {profile.priority > 0 && (
                    <span className="kp-vpn__priority"> · {t('profiles.list.priority', { n: profile.priority })}</span>
                  )}
                </span>
                <SupportChip spec={spec} />
              </span>
            </span>
          )
        },
      },
      {
        id: 'state',
        header: t('profiles.col.state'),
        cell: ({ tunnel, pendingRestart }) => <ProfileState tunnel={tunnel} pendingRestart={pendingRestart} />,
      },
      {
        id: 'cred',
        header: t('profiles.col.signin'),
        cell: ({ profile }) =>
          specOf(profile.type).auth === 'none' ? (
            <span className="kp-muted">{t('profiles.signin.inApp')}</span>
          ) : (
            <span>{t(credentials(profile))}</span>
          ),
      },
      {
        id: 'drops',
        header: t('profiles.col.drops'),
        cell: ({ profile }) => (
          <span className="kp-drops">
            {profile.failover && profile.failover.length > 0 ? (
              <span className="kp-drops__backup">
                {t('profiles.backup.label')} <span className="mono">{profile.failover.join(' → ')}</span>
              </span>
            ) : (
              <span className="kp-drops__backup kp-muted">{t('profiles.backup.none')}</span>
            )}
            <span className="kp-ks">
              <span className="kp-muted kp-small">{t('profiles.col.killSwitch')}</span>
              <StatusPill tone={profile.kill_switch_override === 'on' ? 'on' : 'idle'}>
                {t(OVERRIDE[profile.kill_switch_override])}
              </StatusPill>
              <Link to="/security#profile-overrides" className="kp-small" aria-label={t('profiles.ks.changeFor', { name: profile.name })}>
                {t('profiles.ks.changeShort')}
              </Link>
            </span>
          </span>
        ),
      },
      {
        id: 'actions',
        header: t('profiles.col.actions'),
        hideHeader: true,
        cell: ({ profile }) => (
          <span className="kp-actions">
            <IconButton label={t('profiles.action.editName', { name: profile.name })} icon={<EditIcon />} onClick={() => onEdit(profile)} />
            <IconButton
              label={t('profiles.action.deleteName', { name: profile.name })}
              icon={<TrashIcon />}
              className="kp-danger-icon"
              onClick={() => onDelete(profile)}
            />
          </span>
        ),
      },
    ],
    [t, onEdit, onDelete],
  )

  return (
    <div className="kp-list">
      <DataList
        caption={t('profiles.list.caption')}
        columns={columns}
        rows={rows}
        rowKey={(r) => r.profile.name}
      />
    </div>
  )
}
