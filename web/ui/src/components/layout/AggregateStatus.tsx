// Kongtrol UI — header status: all VPNs at a glance + version · v0.1.0 · 2026-11-18
//
// The worst state wins the color (Four States Rule) and the words always
// say it: "2 de 3 conectadas", "Kongtrol no responde"… It links to Inicio.
// Numbers use tabular figures so a live tick never shifts the header.
import { memo } from 'react'
import { Link } from 'react-router'
import { usePreferences } from '../../app/preferences-context'
import { useDaemonState, useLiveState } from '../../data/connection'
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
  return (
    <Link to="/" className="k-aggregate" aria-label={`${t('status.label')}: ${text}`}>
      <StatusPill tone={tone}>{text}</StatusPill>
      {daemon !== 'offline' && live === 'down' && data && (
        <span className="k-aggregate__live">{t('status.reconnecting')}</span>
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
