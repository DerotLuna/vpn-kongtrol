// Kongtrol UI — one tunnel's status and actions (Inicio) · v1.3.0 · 2026-11-19
//
// The status cell says, in this order: the state (Four States Rule: word +
// color + dot shape), how a provider app's tunnel came up or what the user
// must do in that app (attach), whether the server answers (health), why
// the last connect failed (with the raw error on demand, for developers) and
// what goes through the tunnel. The actions cell offers exactly the actions
// the state allows and shows the one in progress in place until the daemon
// reports that it ended (data/transition.ts).
import { memo, type ReactNode } from 'react'
import { Link } from 'react-router'
import type { TunnelMetrics } from '../../api/types'
import { useAddSheet } from '../../app/addLink'
import { usePreferences } from '../../app/preferences-context'
import { usePendingAction, useTunnelActions, useTunnelReload, type TunnelAction } from '../../data/home'
import { tunnelState, type TunnelState } from '../../data/resources'
import type { MessageKey } from '../../i18n/messages'
import { PowerIcon } from '../profiles/icons'
import { IconButton } from '../ui/Button'
import { StatusPill, type PillTone } from '../ui/StatusPill'
import { CloseIcon, PlugIcon, PlusIcon, RefreshIcon } from '../ui/icons'
import { formatClock } from './format'
import { ERROR_KEYS, attachKey, attachWaiting } from './model'
import { TunnelRoutesToggle } from './TunnelRoutes'

const STATE: Record<TunnelState, { key: MessageKey; tone: PillTone }> = {
  connected: { key: 'tunnels.status.connected', tone: 'ok' },
  connecting: { key: 'tunnels.status.connecting', tone: 'connecting' },
  no_response: { key: 'tunnels.status.noResponse', tone: 'caution' },
  error: { key: 'tunnels.status.error', tone: 'bad' },
  disconnected: { key: 'tunnels.status.disconnected', tone: 'idle' },
}

/**
 * The action in progress on a row: our own request, or a provider app that
 * answered 202 waiting_user and waits for the user to disconnect in the app
 * (attach phase waiting_disconnect; the act line says what to do).
 */
function busyAction(tn: TunnelMetrics, pending: TunnelAction | undefined): TunnelAction | undefined {
  return pending ?? (tn.attach?.phase === 'waiting_disconnect' ? 'disconnect' : undefined)
}

/** Transitional pill while an action runs (Four States: steel "connecting" tone). */
const BUSY_PILL: Partial<Record<TunnelAction, MessageKey>> = {
  connect: 'tunnels.status.connecting',
  disconnect: 'home.status.disconnecting',
  reload: 'home.status.reconnecting',
}

export const TunnelStatusCell = memo(function TunnelStatusCell({ tn }: { tn: TunnelMetrics }) {
  const { t, lang } = usePreferences()
  const busy = busyAction(tn, usePendingAction(tn.Name))
  const busyPill = busy ? BUSY_PILL[busy] : undefined
  const state = tunnelState(tn)
  const s = busyPill ? { key: busyPill, tone: 'connecting' as const } : STATE[state]
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
            {' '}
            <Link to="/settings?section=diagnostics">{t('diag.open')}</Link>
          </p>
          {tn.last_error && (
            <details className="home-detail">
              <summary>{t('home.lastError.detail')}</summary>
              <code>{tn.last_error}</code>
            </details>
          )}
        </div>
      )}
      {connected && <TunnelRoutesToggle name={tn.Name} />}
    </div>
  )
})

const PENDING: Record<TunnelAction, MessageKey> = {
  connect: 'home.action.connecting',
  cancel: 'home.action.cancelling',
  disconnect: 'home.action.disconnecting',
  reload: 'home.action.reloading',
}

const LABEL: Record<TunnelAction, MessageKey> = {
  connect: 'home.action.connect',
  cancel: 'home.action.cancel',
  disconnect: 'home.action.disconnect',
  reload: 'home.action.reload',
}

/**
 * The row's main action: what the state allows (Conectar / Cancelar /
 * Desconectar), or the action in progress, which keeps its button until
 * the daemon reports the end state (data/transition.ts).
 */
function mainAction(tn: TunnelMetrics, busy: TunnelAction | undefined): TunnelAction {
  if (busy && busy !== 'reload') return busy
  return tn.Status === 'connected' ? 'disconnect' : tn.Status === 'connecting' ? 'cancel' : 'connect'
}

const MAIN_ICON: Record<Exclude<TunnelAction, 'reload'>, ReactNode> = {
  connect: <PlugIcon />,
  disconnect: <PowerIcon />,
  cancel: <CloseIcon />,
}

// One layout for every row (owner feedback 2026-11-19): only 36px square
// icon buttons, each with a tooltip and an accessible name that carries the
// tunnel ("Conectar trabajo"), with the same icons as the card's "Conectar
// todas" (plug) / "Desconectar todas" (power). The state's own action comes
// first and alone carries weight: Conectar filled gold when it is down,
// Desconectar with the ink outline when it is up (never gold, DESIGN.md).
// Fixed slots so the main square never moves (owner feedback 2026-11-20):
// [Cancelar while connecting | Reconectar while connected | empty] [main]
// ["＋" send a site through this VPN]. While
// an action runs the squares stay in place but inert (aria-disabled keeps
// focus where the user pressed); the pressed one shows a spinner inside
// the same square until the daemon reports the end state, and a connect in
// progress adds a Cancelar square.
export const TunnelActionsCell = memo(function TunnelActionsCell({ tn }: { tn: TunnelMetrics }) {
  const { t } = usePreferences()
  const run = useTunnelActions()
  const reload = useTunnelReload()
  const { open: openAdd } = useAddSheet()
  const busy = busyAction(tn, usePendingAction(tn.Name))
  const name = tn.Name
  const main = mainAction(tn, busy) as Exclude<TunnelAction, 'reload'>
  const pressed = busy === main
  const word = t(pressed ? PENDING[main] : LABEL[main])
  const canReload = tn.Status === 'connected' || busy === 'reload'
  const reloadWord = t(busy === 'reload' ? PENDING.reload : LABEL.reload)

  return (
    <div className="home-actions">
      {busy === 'connect' ? (
        <IconButton size="sm" variant="outlined" label={`${t('home.action.cancel')} ${name}`} icon={<CloseIcon />} onClick={() => run(name, 'cancel')} />
      ) : canReload ? (
        <IconButton
          size="sm"
          variant="outlined"
          label={`${reloadWord} ${name}`}
          tip={t('home.action.reload.tip')}
          icon={<RefreshIcon />}
          loading={busy === 'reload'}
          aria-disabled={busy ? true : undefined}
          onClick={busy ? undefined : () => reload(name)}
        />
      ) : (
        <span className="home-act-slot" aria-hidden="true" />
      )}
      <IconButton
        size="sm"
        variant={main === 'connect' ? 'filled' : main === 'disconnect' ? 'strong' : 'outlined'}
        className="home-main-act"
        label={`${word} ${name}`}
        icon={MAIN_ICON[main]}
        loading={pressed}
        aria-disabled={busy ? true : undefined}
        onClick={busy ? undefined : () => run(name, main)}
      />
      <IconButton
        size="sm"
        variant="outlined"
        label={t('add.siteVia', { vpn: name })}
        icon={<PlusIcon />}
        aria-haspopup="dialog"
        onClick={() => openAdd({ mode: 'rule', via: name })}
      />
    </div>
  )
})
