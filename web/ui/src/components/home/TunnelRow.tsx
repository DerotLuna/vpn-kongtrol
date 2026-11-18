// Kongtrol UI — one tunnel's status and actions (Inicio) · v1.0.0 · 2026-11-18
//
// The status cell says, in this order: the state (Four States Rule: word +
// color + dot shape), how a provider app's tunnel came up or what the user
// must do in that app (attach), whether the server answers (health), why
// the last connect failed (with the raw error on demand, for developers) and
// what goes through the tunnel. The actions cell offers exactly the actions
// the state allows and shows the one in progress in place.
import { memo } from 'react'
import type { TunnelMetrics } from '../../api/types'
import { usePreferences } from '../../app/preferences-context'
import { usePendingAction, useTunnelActions, useTunnelReload, type TunnelAction } from '../../data/home'
import { tunnelState, type TunnelState } from '../../data/resources'
import type { MessageKey } from '../../i18n/messages'
import { Button } from '../ui/Button'
import { StatusPill, type PillTone } from '../ui/StatusPill'
import { Tooltip } from '../ui/Tooltip'
import { formatClock } from './format'
import { ERROR_KEYS, attachKey, attachWaiting } from './model'
import { TunnelRoutesDisclosure } from './TunnelRoutes'

const STATE: Record<TunnelState, { key: MessageKey; tone: PillTone }> = {
  connected: { key: 'tunnels.status.connected', tone: 'ok' },
  connecting: { key: 'tunnels.status.connecting', tone: 'connecting' },
  no_response: { key: 'tunnels.status.noResponse', tone: 'caution' },
  error: { key: 'tunnels.status.error', tone: 'bad' },
  disconnected: { key: 'tunnels.status.disconnected', tone: 'idle' },
}

export const TunnelStatusCell = memo(function TunnelStatusCell({ tn }: { tn: TunnelMetrics }) {
  const { t, lang } = usePreferences()
  const state = tunnelState(tn)
  const s = STATE[state]
  const connected = tn.Status === 'connected'
  const health = tn.health?.state
  const unknown = health === 'unknown'
  const attach = attachKey(tn.attach)
  const hasError = !connected && !!(tn.last_error || tn.last_error_key)
  const errorFix = tn.last_error_key ? ERROR_KEYS[tn.last_error_key] : undefined
  const reasonFix = tn.health?.reason_key ? ERROR_KEYS[tn.health.reason_key] : undefined

  return (
    <div className="home-status">
      <span className="home-status__pill">
        <StatusPill tone={s.tone}>{t(s.key)}</StatusPill>
        {connected && unknown && <span className="home-status__tag">· {t('home.health.unverified')}</span>}
      </span>
      {attach && tn.attach && (
        <p className={attachWaiting(tn.attach) ? 'home-line home-line--act' : 'home-line'}>
          {t(attach, { app: tn.attach.app, location: tn.attach.location ?? '', reason: tn.attach.reason ?? '' })}
        </p>
      )}
      {health === 'idle' && connected && <p className="home-line home-line--muted">{t('home.health.idle')}</p>}
      {health === 'no_response' && <p className="home-line home-line--caution">{t('tunnels.health.noResponse')}</p>}
      {unknown && connected && (
        <p className="home-line home-line--muted">
          {t('home.health.unknown')} {reasonFix ? t(reasonFix) : t('home.health.unknown.fix')}
        </p>
      )}
      {hasError && (
        <div className="home-line home-line--bad">
          <p>
            <strong>{t('home.lastError', { time: formatClock(tn.last_error_at, lang) })}</strong>{' '}
            {errorFix ? t(errorFix) : t('home.lastError.generic')}
          </p>
          {tn.last_error && (
            <details className="home-detail">
              <summary>{t('home.lastError.detail')}</summary>
              <code>{tn.last_error}</code>
            </details>
          )}
        </div>
      )}
      {connected && <TunnelRoutesDisclosure name={tn.Name} />}
    </div>
  )
})

const PENDING: Record<TunnelAction, MessageKey> = {
  connect: 'home.action.connecting',
  cancel: 'home.action.cancelling',
  disconnect: 'home.action.disconnecting',
  reload: 'home.action.reloading',
}

// Accessible names carry the tunnel ("Conectar trabajo"), not five identical "Conectar";
// the visible word stays first (WCAG 2.5.3 label in name).
export const TunnelActionsCell = memo(function TunnelActionsCell({ tn }: { tn: TunnelMetrics }) {
  const { t } = usePreferences()
  const run = useTunnelActions()
  const reload = useTunnelReload()
  const pending = usePendingAction(tn.Name)
  const name = tn.Name

  if (pending) {
    return (
      <div className="home-actions">
        <Button variant="outlined" size="sm" loading aria-label={`${t(PENDING[pending])} ${name}`}>
          {t(PENDING[pending])}
        </Button>
      </div>
    )
  }
  if (tn.Status === 'connected') {
    return (
      <div className="home-actions">
        <Button variant="outlined" size="sm" onClick={() => run(name, 'disconnect')} aria-label={`${t('home.action.disconnect')} ${name}`}>
          {t('home.action.disconnect')}
        </Button>
        <Tooltip content={t('home.action.reload.tip')} describe>
          <Button variant="text" size="sm" onClick={() => reload(name)} aria-label={`${t('home.action.reload')} ${name}`}>
            {t('home.action.reload')}
          </Button>
        </Tooltip>
      </div>
    )
  }
  if (tn.Status === 'connecting') {
    return (
      <div className="home-actions">
        <Button variant="outlined" size="sm" onClick={() => run(name, 'cancel')} aria-label={`${t('home.action.cancel')} ${name}`}>
          {t('home.action.cancel')}
        </Button>
      </div>
    )
  }
  return (
    <div className="home-actions">
      <Button variant="tonal" size="sm" onClick={() => run(name, 'connect')} aria-label={`${t('home.action.connect')} ${name}`}>
        {t('home.action.connect')}
      </Button>
    </div>
  )
})
