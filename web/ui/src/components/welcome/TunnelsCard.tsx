import type { TunnelsState } from '../../api/hooks'
import type { TunnelMetrics, TunnelStatus } from '../../api/types'
import { usePreferences } from '../../app/preferences-context'
import type { MessageKey } from '../../i18n/messages'
import { guideLink } from '../../links'
import { ErrorNotice } from '../ui/ErrorNotice'
import { ExternalLink } from '../ui/ExternalLink'
import { StatusPill, type PillTone } from '../ui/StatusPill'

const STATUS: Record<TunnelStatus, { key: MessageKey; tone: PillTone }> = {
  connected: { key: 'tunnels.status.connected', tone: 'ok' },
  connecting: { key: 'tunnels.status.connecting', tone: 'warn' },
  disconnected: { key: 'tunnels.status.disconnected', tone: 'idle' },
  error: { key: 'tunnels.status.error', tone: 'bad' },
}

/**
 * Whether a tunnel's server answers (WireGuard handshake, ADR-0030), as a
 * one-line "what to do", or null when there is nothing to flag. Mirrors
 * healthLine in cmd/kongtrol/attach_render.go.
 */
function healthNote(tn: TunnelMetrics): MessageKey | null {
  switch (tn.health?.state) {
    case 'no_response':
      return 'tunnels.health.noResponse'
    case 'unknown':
      return tn.health.reason_key === 'service.error.outdated' ||
        tn.health.reason_key === 'service.error.not_installed'
        ? 'tunnels.health.outdated'
        : 'tunnels.health.unknown'
    default:
      return null
  }
}

/** Step 2: read-only summary of the configured tunnels and their state. */
export function TunnelsCard({ state }: { state: TunnelsState }) {
  const { t } = usePreferences()
  const { tunnels, error, live, reload } = state
  const sorted = tunnels ? [...tunnels].sort((a, b) => a.Name.localeCompare(b.Name)) : []
  const connected = sorted.filter((tn) => tn.Status === 'connected').length

  return (
    <section className="card" aria-labelledby="tunnels-title" aria-busy={!tunnels && !error}>
      <div className="card-head">
        <h2 id="tunnels-title">
          <span className="step-num" aria-hidden="true">
            2
          </span>
          {t('tunnels.title')}
        </h2>
        {live && <span className="muted small">{t('tunnels.live')}</span>}
      </div>

      {error && !tunnels && <ErrorNotice error={error} onRetry={reload} />}
      {!error && !tunnels && <p role="status">{t('tunnels.loading')}</p>}

      {tunnels && sorted.length === 0 && <p>{t('tunnels.none')}</p>}
      {tunnels && sorted.length > 0 && (
        <>
          <p className="summary" role="status">
            {t('tunnels.summary', { connected, total: sorted.length })}
          </p>
          {connected === 0 && <p className="muted small">{t('tunnels.allOff')}</p>}
          <ul className="tunnel-list">
            {sorted.map((tn) => {
              const noResponse = tn.health?.state === 'no_response'
              const s = noResponse
                ? { key: 'tunnels.status.noResponse' as MessageKey, tone: 'caution' as PillTone }
                : (STATUS[tn.Status] ?? STATUS.disconnected)
              const note = healthNote(tn)
              return (
                <li key={tn.Name}>
                  <span>
                    <span className="tunnel-name">{tn.Name}</span>
                    {note && <span className="tunnel-note small">{t(note)}</span>}
                  </span>
                  <StatusPill tone={s.tone}>{t(s.key)}</StatusPill>
                </li>
              )
            })}
          </ul>
        </>
      )}

      <p className="help">
        <ExternalLink href={guideLink('connect')}>{t('help.connect')}</ExternalLink>
      </p>
    </section>
  )
}
