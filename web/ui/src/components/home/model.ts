// Kongtrol UI — Inicio view model: pure mappings from daemon data to words · v1.0.0 · 2026-11-18
//
// Kept apart from the components (fast refresh) and tested on their own.
import type { TunnelMetrics } from '../../api/types'
import type { SecuritySummary } from '../../data/home'
import type { MessageKey } from '../../i18n/messages'
import type { PillTone } from '../ui/StatusPill'

// ── Tunnels ───────────────────────────────────────────────────────────────

/** Daemon last_error_key / health reason_key → our "what to do" line (legacy app.js LAST_ERROR_KEYS). */
export const ERROR_KEYS: Record<string, MessageKey> = {
  'service.error.not_installed': 'home.lastError.notInstalled',
  'service.error.outdated': 'home.lastError.outdated',
  'service.error.denied': 'home.lastError.denied',
  'privops.denied.wg_not_managed': 'home.lastError.wgNotManaged',
  'privops.denied.not_wireguard': 'home.lastError.notWireGuard',
  'privops.denied.file_owner': 'home.lastError.fileOwner',
  'privops.denied.file_writable': 'home.lastError.fileWritable',
  'privops.denied.session_owner': 'home.lastError.sessionOwner',
}

type Attach = NonNullable<TunnelMetrics['attach']>

/** Mirrors vpn.AttachState.MessageKey (internal/vpn/adapter.go); null when nothing to say. */
export function attachKey(a: Attach | undefined): MessageKey | null {
  switch (a?.phase) {
    case 'attached':
      return 'home.attach.attached'
    case 'auto':
      return a.location ? 'home.attach.autoLocation' : 'home.attach.auto'
    case 'user':
      return 'home.attach.user'
    case 'waiting_user':
      return a.reason ? 'home.attach.automationFailed' : 'home.attach.waitingUser'
    case 'waiting_disconnect':
      return 'home.attach.waitingDisconnect'
    case 'dropped':
      return 'home.attach.dropped'
    default:
      return null
  }
}

/** Mirrors vpn.AttachState.Waiting: the user has to act in the provider app. */
export function attachWaiting(a: Attach | undefined): boolean {
  return !!a && (a.phase === 'waiting_user' || a.phase === 'waiting_disconnect' || a.phase === 'dropped')
}

/** Polyline points for the samples, min-max scaled into the box (legacy charts.js sparkline). */
export function sparkPoints(samples: readonly number[], w: number, h: number, pad: number): string {
  if (samples.length === 0) return ''
  const min = Math.min(...samples)
  const max = Math.max(...samples)
  const span = max - min || 1
  const step = samples.length > 1 ? (w - pad * 2) / (samples.length - 1) : 0
  return samples
    .map((v, i) => {
      const x = pad + i * step
      const y = h - pad - ((v - min) / span) * (h - pad * 2)
      return `${x.toFixed(1)},${y.toFixed(1)}`
    })
    .join(' ')
}

// ── Protection ────────────────────────────────────────────────────────────

export interface SecurityLine {
  tone: PillTone
  label: MessageKey
  detail?: { key: MessageKey; vars?: Record<string, string | number> }
}

export interface SecurityView {
  killSwitch: SecurityLine
  dnsGuard: SecurityLine
  leak: SecurityLine
}

/**
 * GET /api/v1/security/status → the three summary lines, the same states as
 * `kongtrol status` and the classic shellSecurityBadges. The kill switch
 * blocking is "caution" (degraded: a VPN dropped), armed and DNS active are
 * "on" (a setting doing its job, never green), a passed leak check is "ok".
 */
export function securityView(st: SecuritySummary): SecurityView {
  const ks = st.kill_switch_status
  const ksState = ks?.state ?? st.kill_switch_state
  let killSwitch: SecurityLine
  if (ksState === 'active') {
    const names = [...(ks?.blocking ?? []), ...(ks?.policy_blocking ?? [])].join(', ')
    killSwitch = { tone: 'caution', label: 'home.sec.ks.active', detail: { key: 'home.sec.ks.blocking', vars: { names } } }
  } else if (ksState === 'armed' || (ksState === 'idle' && st.kill_switch_enabled)) {
    const n = ks?.protected?.length ?? 0
    killSwitch = {
      tone: 'on',
      label: 'home.sec.ks.armed',
      detail:
        n === 0
          ? { key: 'home.sec.ks.armedNone' }
          : n === 1
            ? { key: 'home.sec.ks.protecting.one' }
            : { key: 'home.sec.ks.protecting.other', vars: { n } },
    }
  } else {
    killSwitch = { tone: 'idle', label: 'home.sec.ks.off', detail: { key: 'home.sec.ks.offDetail' } }
  }

  let dnsGuard: SecurityLine
  if (!st.dns_guard_enabled || st.dns_guard_state === 'disabled') dnsGuard = { tone: 'idle', label: 'home.sec.dns.disabled' }
  else if (st.dns_guard_state === 'active' || st.dns_guard) dnsGuard = { tone: 'on', label: 'home.sec.dns.active' }
  else dnsGuard = { tone: 'idle', label: 'home.sec.dns.idle', detail: { key: 'home.sec.dns.idleDetail' } }

  let leak: SecurityLine = { tone: 'idle', label: 'home.sec.leak.disabled', detail: { key: 'home.sec.leak.offHint' } }
  if (st.leak_detection_configured && !st.leak_check_available && !st.public_ip_check_enabled) {
    leak = { tone: 'caution', label: 'home.sec.leak.unavailable' }
  }
  if (st.public_ip_check_enabled) leak = { tone: 'unknown', label: 'home.sec.leak.pending' }
  const lk = st.public_ip_check_enabled ? st.leak_check : null
  if (lk?.state === 'clean') leak = { tone: 'ok', label: 'home.sec.leak.clean' }
  else if (lk?.state === 'leak') leak = { tone: 'bad', label: 'home.sec.leak.leak' }
  else if (lk?.state === 'error') leak = { tone: 'bad', label: 'home.sec.leak.error' }

  return { killSwitch, dnsGuard, leak }
}
