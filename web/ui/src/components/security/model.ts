// Kongtrol UI — Seguridad: daemon state → what the page says · v1.0.0 · 2026-11-18
//
// Pure functions (tested in Security.test.tsx). Pills follow DESIGN.md's
// Four States: green only when protection is working on something, caution
// when the kill switch is blocking (it acts, the user should look), neutral
// "on" when a protection is on but has nothing to do yet, idle when off.
// Each state also has a sentence, so color is never the only signal.
import type { PillTone } from '../ui/StatusPill'
import type { MessageKey } from '../../i18n/messages'
import type { SecurityState } from '../../data/security'

type Vars = Record<string, string | number>

export interface StateView {
  tone: PillTone
  pill: MessageKey
  line: MessageKey
  vars: Vars
  /** The same one-liner `kongtrol status` prints (internal/i18n). */
  cli: MessageKey
}

export type KillSwitchPhase = 'off' | 'armedNone' | 'armed' | 'active'

export function killSwitchPhase(st: SecurityState): KillSwitchPhase {
  if (!st.kill_switch_enabled) return 'off'
  const d = st.kill_switch_status
  if (d?.state === 'active' || st.kill_switch_state === 'active') return 'active'
  return (d?.protected?.length ?? 0) > 0 ? 'armed' : 'armedNone'
}

export function killSwitchView(st: SecurityState): StateView {
  const d = st.kill_switch_status
  const phase = killSwitchPhase(st)
  const names = (d?.blocking ?? []).join(', ')
  const n = d?.protected?.length ?? 0
  switch (phase) {
    case 'active':
      return {
        tone: 'caution',
        pill: 'security.ks.pill.active',
        line: 'security.ks.state.active',
        vars: { names, count: d?.blocked_count ?? 0 },
        cli: 'security.ks.cli.active',
      }
    case 'armed':
      return { tone: 'ok', pill: 'security.ks.pill.armed', line: 'security.ks.state.armed', vars: { n }, cli: 'security.ks.cli.armed' }
    case 'armedNone':
      return { tone: 'on', pill: 'security.ks.pill.armedNone', line: 'security.ks.state.armedNone', vars: {}, cli: 'security.ks.cli.armedNone' }
    default:
      return { tone: 'idle', pill: 'security.ks.pill.off', line: 'security.ks.state.off', vars: {}, cli: 'security.ks.cli.off' }
  }
}

export function dnsView(st: SecurityState): StateView {
  const state = st.dns_guard_enabled ? st.dns_guard_state : 'disabled'
  switch (state) {
    case 'active':
      return { tone: 'ok', pill: 'security.dns.pill.active', line: 'security.dns.state.active', vars: {}, cli: 'security.dns.cli.active' }
    case 'disabled':
      return { tone: 'idle', pill: 'security.dns.pill.disabled', line: 'security.dns.state.disabled', vars: {}, cli: 'security.dns.cli.disabled' }
    default:
      return { tone: 'on', pill: 'security.dns.pill.idle', line: 'security.dns.state.idle', vars: {}, cli: 'security.dns.cli.idle' }
  }
}

export type LeakPhase = 'pending' | 'clean' | 'leak' | 'error'

export function leakPhase(st: SecurityState): LeakPhase {
  const lk = st.leak_check
  if (!lk) return 'pending'
  return lk.state
}

export const LEAK_TONE: Record<LeakPhase, PillTone> = { pending: 'unknown', clean: 'ok', leak: 'bad', error: 'bad' }

/** Key for a policy-watch event (internal/i18n policy.event.*), or null for kinds this page does not show. */
export function watchKey(key: string): MessageKey | null {
  switch (key) {
    case 'policy.event.allow':
      return 'security.watch.allow'
    case 'policy.event.block':
      return 'security.watch.block'
    case 'policy.event.connect':
      return 'security.watch.connect'
    case 'policy.event.connect_failed':
      return 'security.watch.connect_failed'
    default:
      return null
  }
}

/** True when a policy's VPN-off action is "block". */
export function downAction(onTunnelDown: string): 'allow' | 'block' | 'connect' {
  return onTunnelDown === 'block' || onTunnelDown === 'connect' ? onTunnelDown : 'allow'
}

export function isIP(value: string): boolean {
  const v = value.trim()
  if (/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.test(v)) return v.split('.').every((p) => Number(p) <= 255)
  return v.includes(':') && /^[0-9a-f:.]+$/i.test(v)
}

/** Go time.ParseDuration syntax (e.g. 60s, 1m30s, 1.5h). */
export function isDuration(value: string): boolean {
  return /^(\d+(\.\d+)?(ns|us|µs|ms|s|m|h))+$/.test(value.trim())
}
