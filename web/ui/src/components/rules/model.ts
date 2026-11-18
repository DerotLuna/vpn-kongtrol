// Reglas + Horarios — pure model (no React, no i18n) · v1.0.0 · 2026-11-18
//
// Drafts, inline validation, overlap detection and the schedule preview,
// kept pure so they are unit-tested on their own (model.test.ts). The
// rules mirror the daemon's, with the Go source next to each one:
// - domains: exact name or one leading "*." label wildcard
//   (internal/policy/rules.go matchGlob);
// - IP ranges: an IP or a CIDR (policy.ParseRule → net.ParseCIDR);
// - on_tunnel_down "block" needs a domain or range, not only apps
//   (internal/config/semantics.go ValidateOnTunnelDown);
// - schedules: weekdays mon…sun, HH:MM 24 h, a window only when start and
//   end are both set, overnight windows wrap (internal/monitor/scheduler.go).

// ── Policies ──────────────────────────────────────────────────────────────

export type OnTunnelDown = 'allow' | 'block' | 'connect'
export const ON_TUNNEL_DOWN: readonly OnTunnelDown[] = ['allow', 'block', 'connect']

/** GET /api/v1/policies item (internal/api/handlers_policy.go policyDTO). */
export interface Policy {
  name: string
  via: string
  domains: string[] | null
  ip_ranges: string[] | null
  apps: string[] | null
  on_tunnel_down: string
  resolved_cidrs?: string[] | null
}

/** Body of POST/PUT /api/v1/policies (policyPatch). */
export interface PolicyPayload {
  name: string
  via: string
  on_tunnel_down: OnTunnelDown
  match: { domains: string[]; ip_ranges: string[]; apps: string[] }
}

/** The editor's state: lists as one entry per line, the way people paste them. */
export interface PolicyDraft {
  name: string
  via: string
  domains: string
  ipRanges: string
  apps: string
  onDown: OnTunnelDown
}

export function emptyDraft(via = ''): PolicyDraft {
  return { name: '', via, domains: '', ipRanges: '', apps: '', onDown: 'allow' }
}

export function onDownOf(p: Pick<Policy, 'on_tunnel_down'>): OnTunnelDown {
  return (ON_TUNNEL_DOWN as readonly string[]).includes(p.on_tunnel_down) ? (p.on_tunnel_down as OnTunnelDown) : 'allow'
}

export function draftFromPolicy(p: Policy): PolicyDraft {
  return {
    name: p.name,
    via: p.via,
    domains: (p.domains ?? []).join('\n'),
    ipRanges: (p.ip_ranges ?? []).join('\n'),
    apps: (p.apps ?? []).join('\n'),
    onDown: onDownOf(p),
  }
}

/** One entry per line; commas and spaces also separate (pasted lists). */
export function splitEntries(text: string): string[] {
  return text
    .split(/[\s,;]+/)
    .map((s) => s.trim())
    .filter(Boolean)
}

/** Apps keep spaces inside paths ("C:\Program Files\…"): one per line only. */
export function splitLines(text: string): string[] {
  return text
    .split(/\r?\n/)
    .map((s) => s.trim())
    .filter(Boolean)
}

/** A pasted URL or host:port becomes the bare host the daemon matches. */
export function normalizeDomain(raw: string): string {
  let s = raw.trim().toLowerCase()
  s = s.replace(/^[a-z][a-z0-9+.-]*:\/\//, '')
  s = s.split(/[/?#]/, 1)[0] ?? ''
  s = s.replace(/:\d+$/, '').replace(/\.$/, '')
  return s
}

const LABEL = '[a-z0-9_](?:[a-z0-9_-]{0,61}[a-z0-9_])?'
const DOMAIN_RE = new RegExp(`^(?:\\*\\.)?(?:${LABEL}\\.)*${LABEL}$`)

/** Exact names and "*.example.com" (one label) are the only forms the engine matches. */
export function isDomainPattern(s: string): boolean {
  return s.length <= 253 && DOMAIN_RE.test(s)
}

const OCTET = '(?:25[0-5]|2[0-4]\\d|1\\d\\d|[1-9]?\\d)'
const IPV4_RE = new RegExp(`^${OCTET}(?:\\.${OCTET}){3}$`)

export function isIPv4(s: string): boolean {
  return IPV4_RE.test(s)
}

export function isIPv6(s: string): boolean {
  if (!s.includes(':') || /[^0-9a-fA-F:.]/.test(s)) return false
  try {
    return new URL(`http://[${s}]/`).hostname.length > 2
  } catch {
    return false
  }
}

/** An IP or CIDR as Go's net.ParseIP / net.ParseCIDR accept it. */
export function isIpOrCidr(s: string): boolean {
  const [ip = '', prefix, extra] = s.split('/')
  if (extra !== undefined) return false
  const v4 = isIPv4(ip)
  if (!v4 && !isIPv6(ip)) return false
  if (prefix === undefined) return true
  if (!/^\d{1,3}$/.test(prefix)) return false
  return Number(prefix) <= (v4 ? 32 : 128)
}

export type PolicyField = 'name' | 'via' | 'domains' | 'ipRanges' | 'apps' | 'onDown'

export type PolicyErrorCode = 'required' | 'duplicate' | 'noProfiles' | 'noMatch' | 'badDomain' | 'badRange' | 'blockAppsOnly'

export interface PolicyError {
  code: PolicyErrorCode
  /** The offending entry and its 1-based position, for list fields. */
  value?: string
  line?: number
}

export type PolicyErrors = Partial<Record<PolicyField, PolicyError>>

interface ValidateContext {
  /** Every saved policy name (duplicates are case-insensitive, like the daemon). */
  names: readonly string[]
  /** Name of the policy being edited, if any (renaming to itself is fine). */
  editing?: string | undefined
  profiles: readonly string[]
}

export function validatePolicy(d: PolicyDraft, ctx: ValidateContext): PolicyErrors {
  const errors: PolicyErrors = {}
  const name = d.name.trim()
  if (!name) errors.name = { code: 'required' }
  else if (
    ctx.names.some((n) => n.toLowerCase() === name.toLowerCase()) &&
    name.toLowerCase() !== ctx.editing?.toLowerCase()
  )
    errors.name = { code: 'duplicate', value: name }

  if (ctx.profiles.length === 0) errors.via = { code: 'noProfiles' }
  else if (!d.via) errors.via = { code: 'required' }

  const domains = splitEntries(d.domains).map(normalizeDomain)
  const badDomain = domains.findIndex((x) => !isDomainPattern(x))
  if (badDomain >= 0) errors.domains = { code: 'badDomain', value: domains[badDomain] ?? '', line: badDomain + 1 }

  const ranges = splitEntries(d.ipRanges)
  const badRange = ranges.findIndex((x) => !isIpOrCidr(x))
  if (badRange >= 0) errors.ipRanges = { code: 'badRange', value: ranges[badRange] ?? '', line: badRange + 1 }

  const apps = splitLines(d.apps)
  if (domains.length === 0 && ranges.length === 0 && apps.length === 0) errors.domains = { code: 'noMatch' }
  else if (d.onDown === 'block' && domains.length === 0 && ranges.length === 0) errors.onDown = { code: 'blockAppsOnly' }
  return errors
}

export function hasErrors(e: object): boolean {
  return Object.keys(e).length > 0
}

/** Field order for moving focus to the first problem. */
export const POLICY_FIELDS: readonly PolicyField[] = ['name', 'via', 'domains', 'ipRanges', 'apps', 'onDown']

export function toPayload(d: PolicyDraft): PolicyPayload {
  return {
    name: d.name.trim(),
    via: d.via,
    on_tunnel_down: d.onDown,
    match: {
      domains: [...new Set(splitEntries(d.domains).map(normalizeDomain))],
      ip_ranges: [...new Set(splitEntries(d.ipRanges))],
      apps: [...new Set(splitLines(d.apps))],
    },
  }
}

/** How a saved payload shows in the list before the daemon answers (optimistic). */
export function policyFromPayload(p: PolicyPayload): Policy {
  return {
    name: p.name,
    via: p.via,
    domains: p.match.domains,
    ip_ranges: p.match.ip_ranges,
    apps: p.match.apps,
    on_tunnel_down: p.on_tunnel_down,
  }
}

/** Starting points for people who do not write globs or CIDRs. */
export type TemplateId = 'company' | 'network' | 'site'

/** Example names, translated by the caller (miempresa.com / mycompany.com). */
export interface TemplateNames {
  company: string
  site: string
}

export function templateDraft(id: TemplateId, n: TemplateNames): Omit<PolicyDraft, 'via' | 'name'> {
  switch (id) {
    // A company's own sites: the domain itself and every first-level subdomain.
    case 'company':
      return { domains: `${n.company}\n*.${n.company}`, ipRanges: '', apps: '', onDown: 'block' }
    // An internal network (servers, printers, intranet) reached through the work VPN.
    case 'network':
      return { domains: '', ipRanges: '10.0.0.0/8', apps: '', onDown: 'allow' }
    // One site through another VPN (e.g. a service only available in another country).
    case 'site':
      return { domains: `${n.site}\n*.${n.site}`, ipRanges: '', apps: '', onDown: 'allow' }
  }
}

// ── Overlaps ──────────────────────────────────────────────────────────────

export type OverlapKind = 'sameDomain' | 'wildcard' | 'sameRange' | 'nestedRange'

/**
 * Two policies that send the same destination to different VPNs. `winner`
 * is the one the daemon uses: domains → first in evaluation order (the
 * API lists policies in that order: VPN priority, then position); IP
 * ranges → the most specific range, the order only for identical ones.
 */
export interface Overlap {
  kind: OverlapKind
  /** The destination both cover: a name, or the shared (inner) range. */
  value: string
  /** Each side's own entry (pattern or range) that covers it. */
  winner: { policy: string; via: string; entry: string }
  other: { policy: string; via: string; entry: string }
}

function ipv4ToInt(ip: string): number {
  return ip.split('.').reduce((acc, o) => acc * 256 + Number(o), 0)
}

interface V4Range {
  start: number
  end: number
  bits: number
}

/** IPv4 range of an IP or CIDR; IPv6 overlaps are not analysed (null). */
export function v4Range(cidr: string): V4Range | null {
  const [ip = '', p] = cidr.split('/')
  if (!isIPv4(ip)) return null
  const bits = p === undefined ? 32 : Number(p)
  if (!(bits >= 0 && bits <= 32)) return null
  const size = 2 ** (32 - bits)
  const start = Math.floor(ipv4ToInt(ip) / size) * size
  return { start, end: start + size - 1, bits }
}

export function findOverlaps(policies: readonly Policy[]): Overlap[] {
  const out: Overlap[] = []
  for (let i = 0; i < policies.length; i++) {
    for (let j = i + 1; j < policies.length; j++) {
      const a = policies[i]
      const b = policies[j]
      if (!a || !b || a.via === b.via) continue
      // Domains: the first rule in evaluation order wins (a comes before b).
      for (const da of a.domains ?? []) {
        for (const db of b.domains ?? []) {
          let value = ''
          let kind: OverlapKind = 'sameDomain'
          if (da === db) value = da
          else if (da.startsWith('*.') && isCoveredBy(db, da)) [value, kind] = [db, 'wildcard']
          else if (db.startsWith('*.') && isCoveredBy(da, db)) [value, kind] = [da, 'wildcard']
          if (!value) continue
          out.push({ kind, value, winner: { policy: a.name, via: a.via, entry: da }, other: { policy: b.name, via: b.via, entry: db } })
        }
      }
      // IP ranges: the most specific range wins; the order only breaks ties.
      for (const ra of a.ip_ranges ?? []) {
        const x = v4Range(ra)
        if (!x) continue
        for (const rb of b.ip_ranges ?? []) {
          const y = v4Range(rb)
          if (!y || x.end < y.start || y.end < x.start) continue
          if (x.bits === y.bits) {
            out.push({ kind: 'sameRange', value: ra, winner: { policy: a.name, via: a.via, entry: ra }, other: { policy: b.name, via: b.via, entry: rb } })
          } else {
            const [inner, outer] = x.bits > y.bits ? [{ p: a, v: ra }, { p: b, v: rb }] : [{ p: b, v: rb }, { p: a, v: ra }]
            out.push({
              kind: 'nestedRange',
              value: inner.v,
              winner: { policy: inner.p.name, via: inner.p.via, entry: inner.v },
              other: { policy: outer.p.name, via: outer.p.via, entry: outer.v },
            })
          }
        }
      }
    }
  }
  return out
}

/** Does "*.example.com" match this exact name (one label, like matchGlob)? */
function isCoveredBy(name: string, wildcard: string): boolean {
  if (name.startsWith('*.')) return false
  const suffix = wildcard.slice(1)
  if (!name.endsWith(suffix) || name.length <= suffix.length) return false
  return !name.slice(0, name.length - suffix.length).includes('.')
}

// ── Test a destination ───────────────────────────────────────────────────

/** GET /api/v1/policies/explain (policy.ExplainResult: Go field names, no json tags). */
export interface ExplainResult {
  Target: string
  Kind: string
  Matched: boolean
  Via: string
  RuleName: string
  Reason: string
  ReasonCode: string
  ReasonArgs: string[] | null
  DefaultTo: string
}

/** POST /api/v1/policies/test response. */
export interface DraftTestResult {
  matched: boolean
  via: string
  rule: string
  reason?: string
}

export type RouteOutcome = 'none' | 'app' | 'via' | 'connecting' | 'noResponse' | 'normal' | 'blocked' | 'connect'

type TunnelLike = 'connected' | 'connecting' | 'no_response' | 'error' | 'disconnected' | undefined

/**
 * What happens to this traffic right now: the matched rule's VPN state plus
 * its on_tunnel_down (AGENTS.md "Policy routing, kill switch,
 * on_tunnel_down"). Apps are only matched for diagnosis: the daemon does
 * not steer traffic per app, so an app match says so instead of a route.
 */
export function routeOutcome(ex: ExplainResult, onDown: OnTunnelDown, state: TunnelLike): RouteOutcome {
  if (!ex.Matched) return 'none'
  if (ex.Kind === 'app') return 'app'
  switch (state) {
    case 'connected':
      return 'via'
    case 'connecting':
      return 'connecting'
    case 'no_response':
      return 'noResponse'
    default:
      return onDown === 'block' ? 'blocked' : onDown === 'connect' ? 'connect' : 'normal'
  }
}

/** A few things worth testing, taken from the user's own rules. */
export function testSuggestions(policies: readonly Policy[], max = 3): string[] {
  const out: string[] = []
  for (const p of policies) {
    const d = p.domains?.[0]
    const r = p.ip_ranges?.[0]
    if (d) out.push(d.startsWith('*.') ? `www.${d.slice(2)}` : d)
    else if (r) {
      const range = v4Range(r)
      if (range) {
        const n = range.bits >= 31 ? range.start : range.start + 1
        out.push([24, 16, 8, 0].map((s) => Math.floor(n / 2 ** s) % 256).join('.'))
      } else out.push(r.split('/')[0] ?? r)
    }
    if (out.length >= max) break
  }
  return [...new Set(out)]
}

/** "app:chrome.exe" asks about an app; anything else is a destination. */
export function splitTestInput(raw: string): { target: string; app: string } {
  const s = raw.trim()
  if (/^app:/i.test(s)) return { target: '', app: s.slice(4).trim() }
  return { target: s, app: '' }
}

/** Go's time.ParseDuration forms people use here: 30s, 1m, 1m30s, 2h. */
export const DURATION_RE = /^(\d+(\.\d+)?(ns|us|µs|ms|s|m|h))+$/

// ── Schedules ─────────────────────────────────────────────────────────────

export const WEEKDAYS = ['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun'] as const
export type Weekday = (typeof WEEKDAYS)[number]

/** GET /api/v1/scheduler/rules item (scheduleRuleDTO). */
export interface ScheduleRule {
  name: string
  profiles: string[] | null
  weekdays?: string[] | null
  start?: string
  end?: string
  /** false while the rule is paused (daemon default true). */
  enabled?: boolean
}

export interface ScheduleDraft {
  name: string
  profiles: string[]
  /** Empty = every day (the daemon's meaning). */
  days: Weekday[]
  allDay: boolean
  start: string
  end: string
}

export function emptySchedule(): ScheduleDraft {
  return { name: '', profiles: [], days: ['mon', 'tue', 'wed', 'thu', 'fri'], allDay: false, start: '09:00', end: '18:00' }
}

function normDays(days: readonly string[] | null | undefined): Weekday[] {
  const set = new Set((days ?? []).map((d) => d.trim().toLowerCase()))
  return WEEKDAYS.filter((d) => set.has(d))
}

export function scheduleDraftFrom(r: ScheduleRule): ScheduleDraft {
  const allDay = !r.start || !r.end
  return {
    name: r.name,
    profiles: [...(r.profiles ?? [])],
    days: normDays(r.weekdays),
    allDay,
    start: allDay ? '09:00' : (r.start ?? ''),
    end: allDay ? '18:00' : (r.end ?? ''),
  }
}

export type ScheduleField = 'name' | 'profiles' | 'start' | 'end'
export type ScheduleErrorCode = 'required' | 'duplicate' | 'noProfiles' | 'pickProfile' | 'badTime' | 'sameTime'
export type ScheduleErrors = Partial<Record<ScheduleField, { code: ScheduleErrorCode; value?: string }>>

const HHMM = /^([01]\d|2[0-3]):[0-5]\d$/

export function validateSchedule(
  d: ScheduleDraft,
  ctx: { names: readonly string[]; editing?: string | undefined; profiles: readonly string[] },
): ScheduleErrors {
  const errors: ScheduleErrors = {}
  const name = d.name.trim()
  if (!name) errors.name = { code: 'required' }
  else if (
    ctx.names.some((n) => n.toLowerCase() === name.toLowerCase()) &&
    name.toLowerCase() !== ctx.editing?.toLowerCase()
  )
    errors.name = { code: 'duplicate', value: name }
  if (ctx.profiles.length === 0) errors.profiles = { code: 'noProfiles' }
  else if (d.profiles.length === 0) errors.profiles = { code: 'pickProfile' }
  if (!d.allDay) {
    if (!HHMM.test(d.start)) errors.start = { code: d.start ? 'badTime' : 'required' }
    if (!HHMM.test(d.end)) errors.end = { code: d.end ? 'badTime' : 'required' }
    // An equal start and end is a window that never opens (sm <= em, nm >= sm && nm < em).
    if (!errors.start && !errors.end && d.start === d.end) errors.end = { code: 'sameTime' }
  }
  return errors
}

export const SCHEDULE_FIELDS: readonly ScheduleField[] = ['name', 'profiles', 'start', 'end']

export interface SchedulePayload {
  name: string
  profiles: string[]
  weekdays: string[]
  start: string
  end: string
}

export function scheduleToPayload(d: ScheduleDraft): SchedulePayload {
  // Every day is stored as "no weekdays" (shorter YAML, same meaning).
  const days = d.days.length === WEEKDAYS.length ? [] : WEEKDAYS.filter((x) => d.days.includes(x))
  return {
    name: d.name.trim(),
    profiles: [...d.profiles],
    weekdays: days,
    start: d.allDay ? '' : d.start,
    end: d.allDay ? '' : d.end,
  }
}

function minutes(hhmm: string): number {
  const [h = '0', m = '0'] = hhmm.split(':')
  return Number(h) * 60 + Number(m)
}

/** Same answer as the daemon's ruleActiveNow (local time of this machine). */
export function isActiveAt(r: ScheduleRule, now: Date): boolean {
  const days = normDays(r.weekdays)
  const today = WEEKDAYS[(now.getDay() + 6) % 7] as Weekday
  if (days.length > 0 && !days.includes(today)) return false
  const start = (r.start ?? '').trim()
  const end = (r.end ?? '').trim()
  if (!start || !end) return true
  if (!HHMM.test(start) || !HHMM.test(end)) return false
  const s = minutes(start)
  const e = minutes(end)
  const n = now.getHours() * 60 + now.getMinutes()
  return s <= e ? n >= s && n < e : n >= s || n < e
}

/** How the days read: every day, Mon–Fri, weekend, a run of ≥3 days, or a list. */
export type DaysShape =
  | { kind: 'all' }
  | { kind: 'weekdays' }
  | { kind: 'weekend' }
  | { kind: 'range'; from: Weekday; to: Weekday }
  | { kind: 'list'; days: Weekday[] }

export function daysShape(input: readonly string[] | null | undefined): DaysShape {
  const days = normDays(input)
  if (days.length === 0 || days.length === 7) return { kind: 'all' }
  const key = days.join(',')
  if (key === 'mon,tue,wed,thu,fri') return { kind: 'weekdays' }
  if (key === 'sat,sun') return { kind: 'weekend' }
  const idx = days.map((d) => WEEKDAYS.indexOf(d))
  const consecutive = idx.every((v, i) => i === 0 || v === (idx[i - 1] ?? -2) + 1)
  if (consecutive && days.length >= 3) return { kind: 'range', from: days[0] as Weekday, to: days[days.length - 1] as Weekday }
  return { kind: 'list', days }
}

/** "09:00" → "9:00" for reading; the stored value stays HH:MM. */
export function readableTime(hhmm: string): string {
  return hhmm.replace(/^0(\d)/, '$1')
}

export function isOvernight(start: string, end: string): boolean {
  return HHMM.test(start) && HHMM.test(end) && minutes(start) > minutes(end)
}
