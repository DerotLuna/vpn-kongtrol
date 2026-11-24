// Kongtrol UI — profile form model: draft ⇄ API body, and checks before saving · v1.2.0 · 2026-11-24
//
// Pure functions (tested in draft.test.ts). The daemon stays the authority
// (config.Validate answers with a translated error + hint); these checks
// only catch the common mistakes earlier, inline, with the same wording
// (internal/i18n config.err.*), so the user never waits for a round trip
// to learn a field is empty.
//
// Request rules (parity with legacy studio.js:578-636):
// - Fields the chosen type does not show are not sent (absent = kept by the
//   daemon), except the attach-only fields and a server on a type that
//   cannot pick a location: those are sent empty, so switching type never
//   leaves a stale value that fails validation.
// - Create sends only non-empty values; update sends every shown field
//   (empty = clear it).
// - The password is sent only when typed (blank = keep the stored one) and
//   is never read back.
import type { MessageKey } from '../../i18n/messages'
import type { AdapterType, ProfileRequest, VPNProfile } from '../../data/profiles'
import { specOf, type AdapterSpec, type AuthMethod } from './catalog'

export interface Draft {
  name: string
  type: AdapterType
  priority: string
  onTunnelDown: NonNullable<VPNProfile['on_tunnel_down']>
  failover: string[]
  host: string
  config: string
  server: string
  protocol: string
  autoConnect: boolean
  country: string
  city: string
  group: string
  iface: string
  waitTimeout: string
  appPath: string
  authMethod: AuthMethod
  cert: string
  key: string
  username: string
  password: string
}

export type DraftField = keyof Draft

/** internal/config MaxPriority: 0 or empty = by position, 1..1000 = higher wins (ADR-0108). */
export const MAX_PRIORITY = 1000

export function emptyDraft(type: AdapterType = 'openvpn'): Draft {
  return {
    name: '',
    type,
    priority: '10',
    onTunnelDown: 'allow',
    failover: [],
    host: '',
    config: '',
    server: '',
    protocol: '',
    autoConnect: true,
    country: '',
    city: '',
    group: '',
    iface: '',
    waitTimeout: '',
    appPath: '',
    authMethod: specOf(type).defaultAuth,
    cert: '',
    key: '',
    username: '',
    password: '',
  }
}

export function fromProfile(p: VPNProfile): Draft {
  const spec = specOf(p.type)
  const method = (p.auth_method || spec.defaultAuth) as AuthMethod
  return {
    name: p.name,
    type: spec.type,
    // Empty = no priority: the rule's position decides (ADR-0108).
    priority: p.priority > 0 ? String(p.priority) : '',
    onTunnelDown: p.on_tunnel_down ?? 'allow',
    failover: [...(p.failover ?? [])],
    host: p.host ?? '',
    config: p.config ?? '',
    server: p.server ?? '',
    protocol: p.protocol ?? '',
    autoConnect: p.auto_connect !== false,
    country: p.country ?? '',
    city: p.city ?? '',
    group: p.group ?? '',
    iface: p.interface ?? '',
    waitTimeout: p.wait_timeout ?? '',
    appPath: p.app_path ?? '',
    authMethod: method,
    cert: p.cert ?? '',
    key: p.key ?? '',
    username: p.username ?? '',
    password: '',
  }
}

/** Switching type keeps what still applies and sets that type's usual sign-in method. */
export function withType(d: Draft, type: AdapterType): Draft {
  return { ...d, type, authMethod: specOf(type).defaultAuth }
}

/** Which inputs the type shows (hidden ones are not sent). */
export function visibleFields(spec: AdapterSpec): Set<DraftField> {
  const f = new Set<DraftField>(['name', 'type', 'priority', 'failover'])
  if (spec.host) f.add('host')
  if (spec.config) f.add('config')
  if (spec.location) ['server', 'country', 'city'].forEach((k) => f.add(k as DraftField))
  if (spec.serverGroup) f.add('group')
  if (spec.protocol) f.add('protocol')
  if (spec.autoConnect) f.add('autoConnect')
  if (spec.attach) ['iface', 'waitTimeout', 'appPath'].forEach((k) => f.add(k as DraftField))
  if (spec.auth !== 'none') ['authMethod', 'cert', 'key', 'username', 'password'].forEach((k) => f.add(k as DraftField))
  return f
}

const TEXT: [DraftField, keyof ProfileRequest][] = [
  ['host', 'host'],
  ['config', 'config'],
  ['server', 'server'],
  ['protocol', 'protocol'],
  ['country', 'country'],
  ['city', 'city'],
  ['group', 'group'],
  ['iface', 'interface'],
  ['waitTimeout', 'wait_timeout'],
  ['appPath', 'app_path'],
  ['authMethod', 'auth_method'],
  ['cert', 'cert'],
  ['key', 'key'],
  ['username', 'username'],
]

/** Attach-mode fields the daemon rejects on other types: cleared, not just hidden. */
const ATTACH_ONLY: DraftField[] = ['country', 'city', 'group', 'iface', 'waitTimeout', 'appPath']

export function toRequest(d: Draft, mode: 'create' | 'update'): ProfileRequest {
  const spec = specOf(d.type)
  const shown = visibleFields(spec)
  const body: Record<string, unknown> = {
    name: d.name.trim(),
    type: d.type,
    priority: Number.parseInt(d.priority, 10) || 0,
    failover: d.failover.map((s) => s.trim()).filter(Boolean),
    // true is the default and is stored as "unset"; a type without the
    // option always sends true (the daemon rejects false there).
    auto_connect: !spec.autoConnect || d.autoConnect,
  }
  if (mode === 'update' || d.onTunnelDown !== 'allow') body.on_tunnel_down = d.onTunnelDown
  for (const [field, key] of TEXT) {
    const value = d[field] as string
    const trimmed = field === 'authMethod' ? value : value.trim()
    if (shown.has(field)) body[key] = trimmed
    else if (ATTACH_ONLY.includes(field) || (field === 'server' && !spec.location)) body[key] = ''
  }
  if (shown.has('password') && d.password !== '') body.password = d.password
  if (mode === 'create') {
    for (const [k, v] of Object.entries(body)) {
      if (v === '' || (Array.isArray(v) && v.length === 0)) delete body[k]
    }
  }
  return body as ProfileRequest
}

// ── Checks ────────────────────────────────────────────────────────────────

export interface Problem {
  key: MessageKey
  vars?: Record<string, string | number>
}

export type Problems = Partial<Record<DraftField, Problem>>

const UNITS: Record<string, number> = { ns: 1e-9, us: 1e-6, 'µs': 1e-6, ms: 1e-3, s: 1, m: 60, h: 3600 }

/** Go time.ParseDuration subset (no sign), in seconds; NaN when invalid. */
export function parseGoDuration(v: string): number {
  const s = v.trim()
  if (!/^(\d+(\.\d+)?(ns|us|µs|ms|s|m|h))+$/.test(s)) return Number.NaN
  let total = 0
  for (const m of s.matchAll(/(\d+(?:\.\d+)?)(ns|us|µs|ms|s|m|h)/g)) total += Number(m[1]) * (UNITS[m[2] ?? 's'] ?? 0)
  return total
}

/** The daemon refuses $VAR / ${VAR} in fields set from the panel (handlers_vpn.go containsEnvReference). */
export function hasEnvReference(v: string): boolean {
  return /\$[{_A-Za-z]/.test(v)
}

/**
 * The failover graph with this draft's list in place; returns the first
 * cycle as "a → b → a", or '' (same search as config.validateFailover).
 */
export function failoverCycle(name: string, list: string[], profiles: readonly VPNProfile[]): string {
  const graph = new Map<string, string[]>()
  for (const p of profiles) graph.set(p.name, p.failover ?? [])
  graph.set(name, list)
  const state = new Map<string, 1 | 2>()
  const path: string[] = []
  const visit = (n: string): string[] | null => {
    state.set(n, 1)
    path.push(n)
    for (const t of graph.get(n) ?? []) {
      if (state.get(t) === 1) return [...path.slice(path.indexOf(t)), t]
      if (!state.has(t)) {
        const c = visit(t)
        if (c) return c
      }
    }
    path.pop()
    state.set(n, 2)
    return null
  }
  const cycle = visit(name)
  return cycle ? cycle.join(' → ') : ''
}

interface CheckContext {
  profiles: readonly VPNProfile[]
  /** Name of the profile being edited; undefined while creating. */
  editing?: string | undefined
}

export function checkDraft(d: Draft, { profiles, editing }: CheckContext): Problems {
  const spec = specOf(d.type)
  const shown = visibleFields(spec)
  const out: Problems = {}
  const name = d.name.trim()
  if (!name) out.name = { key: 'profiles.check.nameRequired' }
  else if (name !== editing && profiles.some((p) => p.name === name)) out.name = { key: 'profiles.check.nameTaken', vars: { name } }

  if (spec.config && !d.config.trim()) out.config = { key: spec.config === 'ovpn' ? 'profiles.check.ovpnRequired' : 'profiles.check.confRequired' }
  if (spec.host === 'required' && !d.host.trim()) out.host = { key: 'profiles.check.hostRequired' }

  if (shown.has('waitTimeout') && d.waitTimeout.trim()) {
    const secs = parseGoDuration(d.waitTimeout)
    if (!(secs >= 10)) out.waitTimeout = { key: 'profiles.check.waitTimeout', vars: { value: d.waitTimeout.trim() } }
  }
  if (spec.location && d.server.trim() && (d.country.trim() || d.city.trim() || d.group.trim())) {
    out.server = { key: 'profiles.check.serverCombo' }
  }
  const priority = d.priority.trim()
  if (!/^\d*$/.test(priority) || Number(priority) > MAX_PRIORITY) out.priority = { key: 'profiles.check.priority', vars: { max: MAX_PRIORITY } }

  for (const [field] of TEXT) {
    if (out[field] || !shown.has(field)) continue
    if (hasEnvReference(d[field] as string)) out[field] = { key: 'profiles.check.envRef' }
  }

  const self = name || editing || ''
  const seen = new Set<string>()
  for (const target of d.failover) {
    if (target === self) out.failover = { key: 'profiles.check.failoverSelf' }
    else if (seen.has(target)) out.failover = { key: 'profiles.check.failoverDuplicate', vars: { name: target } }
    else if (!profiles.some((p) => p.name === target)) out.failover = { key: 'profiles.check.failoverUnknown', vars: { name: target } }
    if (out.failover) break
    seen.add(target)
  }
  if (!out.failover && self && d.failover.length > 0) {
    const others = profiles.filter((p) => p.name !== editing)
    const cycle = failoverCycle(self, d.failover, others)
    if (cycle) out.failover = { key: 'profiles.check.failoverCycle', vars: { cycle } }
  }
  return out
}

/** Which form field a daemon validation error belongs to (config.err.* keys), for inline display. */
export function fieldForErrorKey(key: string): DraftField | undefined {
  if (key.startsWith('config.err.failover')) return 'failover'
  switch (key) {
    case 'config.err.wait_timeout':
      return 'waitTimeout'
    case 'config.err.profile_priority':
      return 'priority'
    // Only root can read the config file (ADR-0058).
    case 'config.err.file_root_only':
      return 'config'
    case 'config.err.group_nord':
      return 'group'
    case 'config.err.no_location':
    case 'config.err.server_combo':
      return 'server'
    // Rename refusals (PUT with another name, ADR-0042).
    case 'api.err.vpn_exists':
    case 'api.err.vpn_rename_busy':
      return 'name'
    default:
      return undefined
  }
}

/** Other profiles that name `name` as a backup: deleting it would leave them pointing at nothing. */
export function backupOf(name: string, profiles: readonly VPNProfile[]): string[] {
  return profiles.filter((p) => p.name !== name && (p.failover ?? []).includes(name)).map((p) => p.name)
}

/** The list filter (parity with the classic one: name, type, host, server; plus the brand name). */
export function matchesFilter(p: VPNProfile, q: string): boolean {
  const needle = q.trim().toLowerCase()
  if (!needle) return true
  return [p.name, p.type, specOf(p.type).label, p.host ?? '', p.server ?? ''].some((v) => v.toLowerCase().includes(needle))
}
