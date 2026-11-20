// Kongtrol UI — tray flyout page (/tray) · v1.4.0 · 2026-11-20 (ADR-0036, ADR-0038, ADR-0047, ADR-0056, ADR-0069)
//
// The compact window that opens next to the tray icon on a left click
// (cmd/kongtrol-tray/flyout_windows.go). Same data as the dashboard, from the
// same daemon API and cookie. Window-only controls (Salir, Escape to close)
// appear only when the tray's bridge is present; in a normal tab the page is
// a compact status view whose "Abrir panel" is a plain link.
// The solid mark (28 px is below the line art's minimum size, ADR-0047) is a
// transparent SVG, so the dark theme's invert filter leaves only the head.
import logoUrl from '@brand/source/logo-kong-solid.svg'
import { useCallback, useEffect, useRef, useState } from 'react'
import { useAuthState } from '../api/auth'
import { api } from '../api/client'
import { useTunnels } from '../api/hooks'
import type { Group, TunnelMetrics } from '../api/types'
import { useQueryClient } from '@tanstack/react-query'
import { usePreferences } from '../app/preferences-context'
import { keys } from '../data/keys'
import { useAppVersion, useGroupsQuery, useSecurityQuery } from '../data/resources'
import { SHOWN_EVENT, announceReady, trayHost } from '../tray/bridge'
import {
  aggregate,
  applyOrder,
  dnsGuardWord,
  groupState,
  killSwitchWord,
  pendingDone,
  rowState,
  trayOrder,
  type Pending,
  type Tone,
} from '../tray/model'
import { SETTLE_TIMEOUT_MS } from '../data/transition'
import '../theme/tray.css'

/** A request the feed never confirmed stops showing as pending after this. */
const PENDING_MS = SETTLE_TIMEOUT_MS.connect

/**
 * Groups, protection state and version from the shared cache (src/data,
 * ADR-0038): the same requests and polling as the panel. Showing the
 * flyout again refreshes them.
 */
function useTrayData(shownTick: number) {
  const client = useQueryClient()
  const groups = useGroupsQuery().data ?? NO_GROUPS
  const security = useSecurityQuery().data ?? null
  const version = useAppVersion()

  useEffect(() => {
    if (shownTick === 0) return
    void client.invalidateQueries({ queryKey: keys.groups })
    void client.invalidateQueries({ queryKey: keys.security })
  }, [client, shownTick])

  return { groups, security, version }
}

const NO_GROUPS: Group[] = []

function Dot({ tone }: { tone: Tone }) {
  return <span className={`tray-dot ${tone}`} aria-hidden="true" />
}

export function TrayPage() {
  const { t, lang } = usePreferences()
  const host = trayHost() // frozen object injected by the tray: stable
  const [shownTick, setShownTick] = useState(0)
  const { tunnels, error, reload } = useTunnels()
  const { groups, security, version } = useTrayData(shownTick)
  const [pending, setPending] = useState<Record<string, { want: Pending; at: number }>>({})
  const [failed, setFailed] = useState<Record<string, boolean>>({})
  const [quitOpen, setQuitOpen] = useState(false)
  // Row order, taken on each show and when profiles come or go (trayOrder).
  const [order, setOrder] = useState<{ tick: number; names: string[] } | null>(null)
  const rootRef = useRef<HTMLDivElement>(null)
  // The daemon restarted (upgrade, restart): this hidden page's session died
  // with it and every request answers 401. Inside the tray, ask it once per
  // show to reload the page with a fresh sign-in (ADR-0072); in a tab, say
  // how to reopen it. Never show the stale states as if they were live.
  const auth = useAuthState()
  const reauthAsked = useRef(false)
  useEffect(() => {
    if (auth !== 'unauthorized' || !host || reauthAsked.current) return
    reauthAsked.current = true
    host.post('reauth')
  }, [auth, host])

  useEffect(() => {
    document.title = t('tray.pageTitle')
  }, [t, lang])

  // A control that asks for it (quit choices, Reintentar, Agregar VPN) gets
  // focus; otherwise the window itself does, so opening the flyout never
  // paints a focus ring on the first switch (#41) and Tab still starts at
  // the first control.
  const focusFirst = useCallback(() => {
    const root = rootRef.current
    const target = root?.querySelector<HTMLElement>('[data-autofocus]') ?? root
    target?.focus({ preventScroll: true })
  }, [])

  // Opening or closing the quit choices moves focus into / back to them.
  useEffect(() => {
    if (quitOpen) focusFirst()
  }, [quitOpen, focusFirst])

  // Tell the tray we are on screen, after the first load and after every
  // show (it falls back to its native menu otherwise, ADR-0056); refresh and
  // refocus each time it shows us again.
  useEffect(() => {
    let cancel = host ? announceReady(host) : () => {}
    const onShown = () => {
      if (host) {
        cancel()
        cancel = announceReady(host)
      }
      setQuitOpen(false)
      reauthAsked.current = false
      reload()
      setShownTick((n) => n + 1)
      requestAnimationFrame(focusFirst)
    }
    window.addEventListener(SHOWN_EVENT, onShown)
    return () => {
      cancel()
      window.removeEventListener(SHOWN_EVENT, onShown)
    }
  }, [host, reload, focusFirst])

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== 'Escape') return
      if (quitOpen) setQuitOpen(false)
      else host?.post('close')
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [host, quitOpen])

  // Drop pending marks once the daemon's feed (an external system) confirms
  // them, or they time out; deriving instead would revive a mark when the
  // state later changes for another reason.
  useEffect(() => {
    if (!tunnels) return
    const byName = new Map(tunnels.map((tn) => [tn.Name, tn]))
    // oxlint-disable-next-line react/set-state-in-effect
    setPending((prev) => {
      const next = { ...prev }
      let changed = false
      for (const [name, p] of Object.entries(prev)) {
        if (pendingDone(byName.get(name), p.want) || Date.now() - p.at > PENDING_MS) {
          delete next[name]
          changed = true
        }
      }
      return changed ? next : prev
    })
    // A "No se pudo" note ends as soon as the feed shows the profile up or
    // on its way up (the request failed but the tunnel came up anyway, e.g.
    // during a daemon restart), so it never sits next to a connected row.
    // oxlint-disable-next-line react/set-state-in-effect
    setFailed((prev) => {
      const stale = Object.keys(prev).filter((name) => {
        const s = byName.get(name)?.Status
        return prev[name] && (s === 'connected' || s === 'connecting')
      })
      if (stale.length === 0) return prev
      const next = { ...prev }
      for (const name of stale) next[name] = false
      return next
    })
  }, [tunnels])

  const toggle = (tn: TunnelMetrics, on: boolean) => {
    const want: Pending = on ? 'disconnect' : 'connect'
    setFailed((f) => ({ ...f, [tn.Name]: false }))
    setPending((p) => ({ ...p, [tn.Name]: { want, at: Date.now() } }))
    const call = want === 'connect' ? api.connect(tn.Name) : api.disconnect(tn.Name)
    call.then(
      () => reload(),
      () => {
        setPending((p) => {
          const next = { ...p }
          delete next[tn.Name]
          return next
        })
        setFailed((f) => ({ ...f, [tn.Name]: true }))
      },
    )
  }

  // A group action shows each member it touches in progress until the feed
  // confirms it (same rule as a single row), not just while the request runs.
  const toggleGroup = (g: Group, allUp: boolean) => {
    const want: Pending = allUp ? 'disconnect' : 'connect'
    const members = (tunnels ?? []).filter(
      (tn) => g.profiles.includes(tn.Name) && (want === 'disconnect' ? tn.Status === 'connected' : tn.Status !== 'connected'),
    )
    const at = Date.now()
    setPending((p) => ({ ...p, ...Object.fromEntries(members.map((tn) => [tn.Name, { want, at }])) }))
    const call = allUp ? api.disconnectGroup(g.name) : api.connectGroup(g.name)
    call.then(reload, () => {
      setPending((p) => {
        const next = { ...p }
        for (const tn of members) delete next[tn.Name]
        return next
      })
      reload()
    })
  }

  useEffect(() => {
    if (!tunnels) return
    const key = tunnels
      .map((tn) => tn.Name)
      .sort()
      .join('\n')
    // oxlint-disable-next-line react/set-state-in-effect
    setOrder((prev) =>
      prev && prev.tick === shownTick && [...prev.names].sort().join('\n') === key
        ? prev
        : { tick: shownTick, names: trayOrder(tunnels) },
    )
  }, [tunnels, shownTick])

  const sorted = tunnels ? applyOrder(tunnels, order?.names ?? trayOrder(tunnels)) : []
  const rows = sorted.map((tn) => ({ tn, st: rowState(tn, pending[tn.Name]?.want) }))
  const agg = aggregate(rows.map((r) => r.st))
  const offline = error && !tunnels
  const unauthorized = auth === 'unauthorized'
  const blocking = security?.kill_switch_state === 'active'
  // one primary per view: with no VPN it is "Agregar VPN", not "Abrir panel"
  const noVPNs = !!tunnels && tunnels.length === 0
  const openAdd = () => (host ? host.post('open-add') : window.location.assign('/?add=vpn'))

  const publicIP = !security?.public_ip_check_enabled
    ? t('tray.publicip.off')
    : security.leak_check?.public_ip || t('tray.publicip.unknown')

  return (
    <div className="tray" ref={rootRef} tabIndex={-1}>
      <header className="tray-head">
        <img className="tray-logo" src={logoUrl} alt="" width="28" height="28" />
        <div className="tray-head-text">
          <h1 className="tray-title">{t('app.name')}</h1>
          <p className={`tray-agg ${offline ? 'danger' : agg.tone}`} role="status">
            <Dot tone={offline ? 'danger' : agg.tone} />
            {offline ? t('tray.offline') : tunnels ? t(agg.key, agg.vars) : t('tray.agg.loading')}
          </p>
        </div>
      </header>

      <main className="tray-body" aria-busy={!tunnels && !error}>
        {offline && (
          <div className="tray-notice" role="alert">
            <p>{t('tray.offline.fix')}</p>
            <button type="button" className="tray-btn ghost small" onClick={reload} data-autofocus>
              {t('tray.retry')}
            </button>
          </div>
        )}

        {unauthorized && (
          <div className="tray-notice" role="alert">
            {host ? (
              <p>{t('tray.reauth')}</p>
            ) : (
              <p>
                <strong>{t('tray.unauth')}</strong> {t('tray.unauth.fix')}
              </p>
            )}
          </div>
        )}

        {blocking && (
          <div className="tray-notice caution" role="status">
            <p>
              <strong>{t('tray.ks.blocking')}</strong> {t('tray.ks.blocking.fix')}
            </p>
          </div>
        )}

        {tunnels && !unauthorized && (
          <section aria-labelledby="tray-vpns">
            <h2 id="tray-vpns" className="tray-label">
              {t('tray.vpns')}
            </h2>
            {rows.length === 0 ? (
              <div className="tray-empty">
                <p>{t('tray.vpns.none')}</p>
                <button type="button" className="tray-btn primary" onClick={openAdd} data-autofocus>
                  {t('tray.vpns.add')}
                </button>
              </div>
            ) : (
              <ul className="tray-list">
                {rows.map(({ tn, st }, i) => {
                  const noteId = `tray-note-${i}`
                  const showFail = failed[tn.Name]
                  const note = showFail ? { key: 'tray.note.failed' as const } : st.note
                  return (
                    <li key={tn.Name} className="tray-row">
                      <div className="tray-row-main">
                        <span className="tray-name">{tn.Name}</span>
                        <span className={`tray-state ${st.tone}`}>
                          <Dot tone={st.tone} />
                          {t(st.word)}
                        </span>
                        {note && (
                          <span id={noteId} className="tray-note">
                            {t(note.key, 'vars' in note ? note.vars : undefined)}
                          </span>
                        )}
                      </div>
                      <button
                        type="button"
                        role="switch"
                        aria-checked={st.on}
                        aria-busy={st.busy}
                        aria-describedby={note ? noteId : undefined}
                        aria-label={t(st.on ? 'tray.toggle.disconnect' : 'tray.toggle.connect', { name: tn.Name })}
                        className={`tray-switch ${st.tone}`}
                        disabled={pending[tn.Name] !== undefined}
                        onClick={() => toggle(tn, st.on)}
                      >
                        <span className="tray-switch-knob" aria-hidden="true" />
                      </button>
                    </li>
                  )
                })}
              </ul>
            )}
          </section>
        )}

        {tunnels && groups.length > 0 && (
          <section aria-labelledby="tray-groups">
            <h2 id="tray-groups" className="tray-label">
              {t('tray.groups')}
            </h2>
            <ul className="tray-list">
              {groups.map((g) => {
                const gs = groupState(g, tunnels)
                return (
                  <li key={g.name} className="tray-row">
                    <div className="tray-row-main">
                      <span className="tray-name">{g.name}</span>
                      <span className="tray-note">{t('tray.group.count', { connected: gs.connected, total: gs.total })}</span>
                    </div>
                    <button
                      type="button"
                      className="tray-btn ghost small"
                      aria-label={t(gs.allUp ? 'tray.group.disconnectLabel' : 'tray.group.connectLabel', { name: g.name })}
                      disabled={gs.total === 0}
                      onClick={() => toggleGroup(g, gs.allUp)}
                    >
                      {t(gs.allUp ? 'tray.group.disconnect' : 'tray.group.connect')}
                    </button>
                  </li>
                )
              })}
            </ul>
          </section>
        )}

        {security && (
          <section aria-labelledby="tray-protection">
            <h2 id="tray-protection" className="tray-label">
              {t('tray.protection')}
            </h2>
            <dl className="tray-facts">
              <div>
                <dt>{t('tray.killswitch')}</dt>
                <dd className={security.kill_switch_state === 'active' ? 'caution' : undefined}>
                  {t(killSwitchWord(security.kill_switch_state))}
                </dd>
              </div>
              <div>
                <dt>{t('tray.dnsguard')}</dt>
                <dd>{t(dnsGuardWord(security.dns_guard_state))}</dd>
              </div>
              <div>
                <dt>{t('tray.publicip')}</dt>
                <dd className={security.leak_check?.public_ip ? 'mono' : undefined}>{publicIP}</dd>
              </div>
            </dl>
          </section>
        )}
      </main>

      <footer className="tray-foot">
        {quitOpen && host ? (
          <div className="tray-quit" role="group" aria-labelledby="tray-quit-title">
            <p id="tray-quit-title" className="tray-quit-title">
              {t('tray.quit.title')}
            </p>
            <button type="button" className="tray-btn ghost" onClick={() => host.post('quit-tray')} data-autofocus>
              {t('tray.quit.tray')}
              <span className="tray-btn-note">{t('tray.quit.trayNote')}</span>
            </button>
            <button type="button" className="tray-btn danger" onClick={() => host.post('quit-all')}>
              {t('tray.quit.all')}
            </button>
            <button type="button" className="tray-btn link" onClick={() => setQuitOpen(false)}>
              {t('tray.quit.cancel')}
            </button>
          </div>
        ) : (
          <div className="tray-actions">
            {host ? (
              <button type="button" className={`tray-btn ${noVPNs ? 'ghost' : 'primary'}`} onClick={() => host.post('open-dashboard')}>
                {t('tray.open')}
              </button>
            ) : (
              <a className={`tray-btn ${noVPNs ? 'ghost' : 'primary'}`} href="/">
                {t('tray.open')}
              </a>
            )}
            {host && (
              <button type="button" className="tray-btn ghost" onClick={() => setQuitOpen(true)}>
                {t('tray.quit')}
              </button>
            )}
          </div>
        )}
        <p className="tray-version">{version ? t('tray.version', { version }) : t('app.name')}</p>
      </footer>
    </div>
  )
}
