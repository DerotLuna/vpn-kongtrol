// Kongtrol UI — small pieces shared by Perfiles and Grupos · v1.0.0 · 2026-11-18
//
// Monogram tiles (our own marks, never a provider logo: ADR-0023), the
// "works on this system" chip, the profile state pill and the page header
// with its Perfiles · Grupos tabs.
import type { ReactNode } from 'react'
import type { TunnelMetrics } from '../../api/types'
import { usePreferences } from '../../app/preferences-context'
import { tunnelState } from '../../data/resources'
import type { MessageKey } from '../../i18n/messages'
import { detectPlatform } from '../../lib/platform'
import { PageHeader } from '../layout/PageHeader'
import { StatusPill, type PillTone } from '../ui/StatusPill'
import { TabLinks } from '../ui/Tabs'
import { supportOn, type AdapterSpec, type Support } from './catalog'

export function Monogram({ spec, size = 'md' }: { spec: AdapterSpec; size?: 'md' | 'lg' }) {
  return (
    <span className={`kp-mono kp-mono--${spec.group} kp-mono--${size}`} aria-hidden="true">
      {spec.mark}
    </span>
  )
}

const SUPPORT: Record<Support | 'none', MessageKey> = {
  yes: 'profiles.support.yes',
  auto: 'profiles.support.auto',
  manual: 'profiles.support.manual',
  partial: 'profiles.support.partial',
  none: 'profiles.support.none',
}

const SUPPORT_LONG: Record<Support | 'none', MessageKey> = {
  yes: 'profiles.support.yes.long',
  auto: 'profiles.support.auto.long',
  manual: 'profiles.support.manual.long',
  partial: 'profiles.support.partial.long',
  none: 'profiles.support.none.long',
}

const platform = detectPlatform()

/** One chip: how this VPN works on this computer's OS (nothing when the OS is unknown). */
export function SupportChip({ spec, long = false }: { spec: AdapterSpec; long?: boolean }) {
  const { t } = usePreferences()
  const s = supportOn(spec, platform)
  if (!s) return null
  const label = t(SUPPORT[s])
  return (
    <span className={`kp-chip kp-chip--${s}${long ? ' kp-chip--long' : ''}`} title={t(SUPPORT_LONG[s])}>
      {long ? `${label} · ${t(SUPPORT_LONG[s])}` : label}
    </span>
  )
}

/** Four States Rule; a profile the running daemon does not know yet waits for a restart. */
export function ProfileState({ tunnel, pendingRestart }: { tunnel: TunnelMetrics | undefined; pendingRestart: boolean }) {
  const { t } = usePreferences()
  if (!tunnel) {
    return pendingRestart ? (
      <StatusPill tone="unknown">{t('profiles.state.restart')}</StatusPill>
    ) : (
      <StatusPill tone="idle">{t('tunnels.status.disconnected')}</StatusPill>
    )
  }
  const s = tunnelState(tunnel)
  const map: Record<typeof s, [PillTone, MessageKey]> = {
    connected: ['ok', 'tunnels.status.connected'],
    connecting: ['connecting', 'tunnels.status.connecting'],
    no_response: ['caution', 'tunnels.status.noResponse'],
    error: ['bad', 'tunnels.status.error'],
    disconnected: ['idle', 'tunnels.status.disconnected'],
  }
  const [tone, key] = map[s]
  return <StatusPill tone={tone}>{t(key)}</StatusPill>
}

export function ProfilesHeader({ children }: { children?: ReactNode }) {
  const { t } = usePreferences()
  return (
    <PageHeader title={t('nav.profiles')} lead={t('page.profiles.lead')}>
      <TabLinks
        label={t('nav.profiles')}
        items={[
          { to: '/profiles', label: t('page.profiles.tab.vpns') },
          { to: '/profiles/groups', label: t('page.profiles.tab.groups') },
        ]}
      />
      {children}
    </PageHeader>
  )
}
