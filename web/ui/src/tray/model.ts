// Kongtrol UI — tray flyout state rules · v1.0.0 · 2026-11-18
//
// Pure functions so the Four States rule (DESIGN.md) is tested without
// rendering: connected = live, connecting = steel, no response = caution,
// error = danger; every state also has a word.
import type { Group, SecurityStatus, TunnelMetrics } from '../api/types'
import type { MessageKey } from '../i18n/messages'

/** What the user just asked for and the feed has not confirmed yet. */
export type Pending = 'connect' | 'disconnect'

export type Tone = 'live' | 'steel' | 'caution' | 'danger' | 'idle'

export interface RowState {
  tone: Tone
  word: MessageKey
  /** Switch position: on while up or on its way up. */
  on: boolean
  busy: boolean
  note: { key: MessageKey; vars?: Record<string, string> } | null
}

export function rowState(tn: TunnelMetrics, pending?: Pending): RowState {
  const attach = tn.attach
  if (pending === 'disconnect' || attach?.phase === 'waiting_disconnect') {
    return {
      tone: 'steel',
      word: 'tray.status.disconnecting',
      on: false,
      busy: true,
      note: attach?.phase === 'waiting_disconnect' ? { key: 'tray.note.waitingDisconnect', vars: { app: attach.app } } : null,
    }
  }
  if (tn.Status === 'connected' && tn.health?.state === 'no_response') {
    return { tone: 'caution', word: 'tray.status.noResponse', on: true, busy: false, note: { key: 'tray.note.noResponse' } }
  }
  if (tn.Status === 'connected') {
    return { tone: 'live', word: 'tray.status.connected', on: true, busy: false, note: null }
  }
  if (tn.Status === 'connecting' || pending === 'connect') {
    return {
      tone: 'steel',
      word: 'tray.status.connecting',
      on: true,
      busy: true,
      note: attach?.phase === 'waiting_user' ? { key: 'tray.note.waitingUser', vars: { app: attach.app } } : null,
    }
  }
  if (tn.Status === 'error') {
    return { tone: 'danger', word: 'tray.status.error', on: false, busy: false, note: { key: 'tray.note.error' } }
  }
  return { tone: 'idle', word: 'tray.status.disconnected', on: false, busy: false, note: null }
}

/** Whether the feed now shows what the user asked for. */
export function pendingDone(tn: TunnelMetrics | undefined, pending: Pending): boolean {
  if (!tn) return true
  if (pending === 'connect') return tn.Status === 'connected' || tn.Status === 'error'
  return tn.Status === 'disconnected' || tn.Status === 'error'
}

export interface Aggregate {
  tone: Tone
  key: MessageKey
  vars?: Record<string, number>
}

/**
 * The header line: errors first, then servers not answering, then work in
 * progress, then the count.
 */
export function aggregate(rows: RowState[]): Aggregate {
  const errors = rows.filter((r) => r.tone === 'danger').length
  if (errors > 0) return { tone: 'danger', key: 'tray.agg.error', vars: { count: errors } }
  const silent = rows.filter((r) => r.tone === 'caution').length
  if (silent > 0) return { tone: 'caution', key: 'tray.agg.noResponse', vars: { count: silent } }
  if (rows.some((r) => r.busy)) return { tone: 'steel', key: 'tray.agg.connecting' }
  const connected = rows.filter((r) => r.tone === 'live').length
  if (connected === 0) return { tone: 'idle', key: 'tray.agg.none' }
  return { tone: 'live', key: 'tray.agg.some', vars: { connected, total: rows.length } }
}

export interface GroupState {
  connected: number
  total: number
  /** Every member is up: the button disconnects the group. */
  allUp: boolean
}

export function groupState(group: Group, tunnels: TunnelMetrics[]): GroupState {
  const byName = new Map(tunnels.map((tn) => [tn.Name, tn]))
  const members = group.profiles.filter((p) => byName.has(p))
  const connected = members.filter((p) => byName.get(p)?.Status === 'connected').length
  return { connected, total: members.length, allUp: members.length > 0 && connected === members.length }
}

export function killSwitchWord(state: SecurityStatus['kill_switch_state']): MessageKey {
  return (
    {
      disabled: 'tray.killswitch.disabled',
      idle: 'tray.killswitch.idle',
      armed: 'tray.killswitch.armed',
      active: 'tray.killswitch.active',
    } as const
  )[state] ?? 'tray.killswitch.disabled'
}

export function dnsGuardWord(state: SecurityStatus['dns_guard_state']): MessageKey {
  return (
    {
      disabled: 'tray.dnsguard.disabled',
      idle: 'tray.dnsguard.idle',
      active: 'tray.dnsguard.active',
    } as const
  )[state] ?? 'tray.dnsguard.disabled'
}
