import type { TunnelsState } from '../../api/hooks'
import type { TunnelStatus } from '../../api/types'
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
              const s = STATUS[tn.Status] ?? STATUS.disconnected
              return (
                <li key={tn.Name}>
                  <span className="tunnel-name">{tn.Name}</span>
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
