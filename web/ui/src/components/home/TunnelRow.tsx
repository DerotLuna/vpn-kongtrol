// Kongtrol UI — one tunnel's status and actions (Inicio) · v1.2.0 · 2026-11-19
//
// The status cell says, in this order: the state (Four States Rule: word +
// color + dot shape), how a provider app's tunnel came up or what the user
// must do in that app (attach), whether the server answers (health), why
// the last connect failed (with the raw error on demand, for developers) and
// what goes through the tunnel. The actions cell offers exactly the actions
// the state allows and shows the one in progress in place until the daemon
// reports that it ended (data/transition.ts).
import { memo } from 'react'
import type { TunnelMetrics } from '../../api/types'
import { useAddSheet } from '../../app/addLink'
import { usePreferences } from '../../app/preferences-context'
import { usePendingAction, useTunnelActions, useTunnelReload, type TunnelAction } from '../../data/home'
import { tunnelState, type TunnelState } from '../../data/resources'
import type { MessageKey } from '../../i18n/messages'
import { Button, IconButton } from '../ui/Button'
import { Menu, type MenuItem } from '../ui/Menu'
import { StatusPill, type PillTone } from '../ui/StatusPill'
import { PlusIcon, RefreshIcon } from '../ui/icons'
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

// One layout for every row (DESIGN.md: one primary per row, owner feedback
// 2026-11-19): the main action as a small button — Conectar in amber (the
// next step), Desconectar / Cancelar outlined (neutral; never amber) — then
// the secondary actions as 36px icon buttons with tooltips: Reconectar
// (connected only) and "＋" (send a site through this VPN). When the table
// is narrow the secondary ones fold into a "⋯" menu (CSS container query in
// home.css: the inline set and the menu are both rendered, one is hidden),
// so the row never wraps. Accessible names carry the tunnel ("Conectar
// trabajo"); the visible word stays first (WCAG 2.5.3). While an action runs
// the buttons stay in place but inert (aria-disabled keeps focus where the
// user pressed) and the pressed one shows a spinner and "…ando"; a connect
// in progress also offers Cancelar.
export const TunnelActionsCell = memo(function TunnelActionsCell({ tn }: { tn: TunnelMetrics }) {
  const { t } = usePreferences()
  const run = useTunnelActions()
  const reload = useTunnelReload()
  const { open: openAdd } = useAddSheet()
  const busy = busyAction(tn, usePendingAction(tn.Name))
  const name = tn.Name
  const main = mainAction(tn, busy)
  const pressed = busy === main
  const word = t(pressed ? PENDING[main] : LABEL[main])
  const canReload = tn.Status === 'connected' || busy === 'reload'
  const reloadWord = t(busy === 'reload' ? PENDING.reload : LABEL.reload)
  const addLabel = t('add.siteVia', { vpn: name })
  const doReload = () => reload(name)
  const doAdd = () => openAdd({ mode: 'rule', via: name })

  const menu: MenuItem[] = [
    ...(canReload ? [{ id: 'reload', label: reloadWord, icon: <RefreshIcon />, onSelect: doReload, disabled: !!busy }] : []),
    { id: 'add', label: addLabel, icon: <PlusIcon />, onSelect: doAdd },
  ]

  return (
    <div className="home-actions">
      <Button
        variant={main === 'connect' ? 'filled' : 'outlined'}
        size="sm"
        className="home-main-act"
        icon={pressed ? <span className="k-spinner" aria-hidden="true" /> : undefined}
        aria-label={`${word} ${name}`}
        aria-disabled={busy ? true : undefined}
        aria-busy={pressed || undefined}
        onClick={busy ? undefined : () => run(name, main as Exclude<TunnelAction, 'reload'>)}
      >
        {word}
      </Button>
      {busy === 'connect' && (
        <Button variant="text" size="sm" aria-label={`${t('home.action.cancel')} ${name}`} onClick={() => run(name, 'cancel')}>
          {t('home.action.cancel')}
        </Button>
      )}
      <span className="home-actions__inline">
        {canReload && (
          <IconButton
            size="sm"
            label={`${reloadWord} ${name}`}
            tip={t('home.action.reload.tip')}
            icon={<RefreshIcon />}
            loading={busy === 'reload'}
            aria-disabled={busy ? true : undefined}
            onClick={busy ? undefined : doReload}
          />
        )}
        <IconButton size="sm" label={addLabel} icon={<PlusIcon />} aria-haspopup="dialog" onClick={doAdd} />
      </span>
      <Menu className="home-actions__more" size="sm" label={t('kit.more', { name })} items={menu} />
    </div>
  )
})
