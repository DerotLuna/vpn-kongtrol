// Kongtrol UI — header status: all VPNs at a glance + version · v0.2.0 · 2026-11-18
//
// The worst state wins the color (Four States Rule) and the words always
// say it: "2 de 3 conectadas", "Kongtrol no responde"… It links to Inicio.
// Numbers use tabular figures so a live tick never shifts the header.
// The link's name is its visible text after a hidden "Estado de tus VPN:"
// prefix, so the name contains what is on screen (WCAG 2.5.3 Label in Name).
// While the live feed is down, "Reconectando…" shrinks to a steel icon on
// phones (the words stay for screen readers) and truncates elsewhere, so
// the header never grows wider than the screen.
import { memo } from 'react'
import { RefreshIcon } from '../ui/icons'
import { Link } from 'react-router'
import { usePreferences } from '../../app/preferences-context'
import { useDaemonState, useLiveState } from '../../data/connection'
import { useAuthState } from '../../api/auth'
import { summarizeTunnels, useAppVersion, useTunnelsQuery, type TunnelState } from '../../data/resources'
import { StatusPill, type PillTone } from '../ui/StatusPill'

const TONE: Record<TunnelState | 'none', PillTone> = {
  connected: 'ok',
  connecting: 'connecting',
  no_response: 'caution',
  error: 'bad',
  disconnected: 'idle',
  none: 'idle',
}

export const AggregateStatus = memo(function AggregateStatus() {
  const { t } = usePreferences()
  const daemon = useDaemonState()
  const live = useLiveState()
  const auth = useAuthState()
  const { data } = useTunnelsQuery()

  let tone: PillTone = 'unknown'
  let text = t('status.loading')
  if (daemon === 'offline') {
    tone = 'bad'
    text = t('status.stopped')
  } else if (data) {
    const sum = summarizeTunnels(data)
    tone = TONE[sum.worst]
    text = sum.total === 0 ? t('status.none') : t('status.summary', { connected: sum.connected, total: sum.total })
  }
  const reconnecting = auth === 'ok' && daemon !== 'offline' && live === 'down' && !!data
  // The {' '} text nodes separate words in the accessible name; the flex
  // layout ignores them.
  return (
    <Link to="/" className="k-aggregate">
      <span className="visually-hidden">{t('status.label')}:</span>{' '}
      <StatusPill tone={tone}>{text}</StatusPill>
      {reconnecting && (
        <>
          {' '}
          <span className="k-aggregate__live">
            <RefreshIcon />
            <span className="k-aggregate__live-text">{t('status.reconnecting')}</span>
          </span>
        </>
      )}
    </Link>
  )
})

export function VersionTag({ className = '' }: { className?: string }) {
  const { t } = usePreferences()
  const version = useAppVersion()
  if (!version) return null
  return <span className={`k-version ${className}`.trim()}>{t('header.version', { version })}</span>
}
