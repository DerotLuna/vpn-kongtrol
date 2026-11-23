// Kongtrol UI — "＋ Agregar": pure helpers · v1.1.0 · 2026-11-19
//
// No React, no i18n (tested in model.test.ts). The rules mirror the
// daemon's, with the Go source named next to each one:
// - "Sitio, IP o programa": policy.ClassifyTargets / AutoName / CleanName /
//   SuggestName (internal/policy/classify.go), the same code behind
//   `kongtrol policy add`. Both read internal/policy/testdata/
//   classify_targets.json in their tests, so the sheet and the CLI read
//   the same entries the same way (ADR-0055);
// - names: config.ValidateName (internal/config/semantics.go): not blank,
//   ≤ 64 characters, none of < > " ' ` & \, no control characters or
//   line/paragraph separators;
// - file type by extension: profileimport.TypeForFile (.conf = WireGuard,
//   .ovpn = OpenVPN).
import { importName } from '../profiles/importModel'
import { isDomainPattern, isIPv4, isIPv6, isIpOrCidr, normalizeDomain, type TunnelDownChoice } from '../rules/model'

// ── Names ─────────────────────────────────────────────────────────────────

export const MAX_NAME = 64
/** policy.AutoName keeps generated names to 60 so " 2", " 3"… still fit. */
export const AUTO_NAME_MAX = 60

const FORBIDDEN = '<>"\'`&\\'

/** Control characters (C0, DEL, C1) and U+2028 / U+2029, like unicode.IsControl + the two separators. */
function isControl(c: string): boolean {
  const n = c.codePointAt(0) ?? 0
  return n < 0x20 || (n >= 0x7f && n <= 0x9f) || n === 0x2028 || n === 0x2029
}

const refused = (c: string) => FORBIDDEN.includes(c) || isControl(c)

/** Same answer as config.ValidateName. */
export function isValidName(name: string): boolean {
  if (name.trim() === '' || [...name].length > MAX_NAME) return false
  return ![...name].some(refused)
}

/** policy.CleanName: drops what ValidateName refuses and keeps at most `max` characters. */
export function cleanName(raw: string, max = AUTO_NAME_MAX): string {
  const kept: string[] = []
  for (const c of raw) {
    if (kept.length >= max) break
    if (!refused(c)) kept.push(c)
  }
  return kept.join('').trim()
}

/**
 * policy.SuggestName: `base`, or `base 2`, `base 3`… when a name is taken
 * (case-insensitive, like the daemon). `sep` is "-" for stored file names
 * (no spaces there).
 */
export function uniqueName(base: string, taken: readonly string[], sep = ' '): string {
  const used = new Set(taken.map((n) => n.toLowerCase()))
  if (!used.has(base.toLowerCase())) return base
  for (let i = 2; ; i++) {
    const name = `${base}${sep}${i}`
    if (!used.has(name.toLowerCase())) return name
  }
}

/** policy.AutoName's first argument: an app's file name, not its folder. */
export function nameSource(first: string): string {
  const i = Math.max(first.lastIndexOf('/'), first.lastIndexOf('\\'))
  return i >= 0 && i < first.length - 1 ? first.slice(i + 1) : first
}

/** policy.AutoName with the format already filled ("claude.ai por Trabajo"). */
export function autoName(filled: string, taken: readonly string[]): string {
  return uniqueName(cleanName(filled, AUTO_NAME_MAX), taken)
}

// ── Files ─────────────────────────────────────────────────────────────────

/** profileimport.TypeForFile: the extension decides; anything else is not a VPN file. */
export function typeForFile(fileName: string): 'wireguard' | 'openvpn' | null {
  const ext = /\.[^./\\]*$/.exec(fileName.toLowerCase())?.[0]
  if (ext === '.conf') return 'wireguard'
  if (ext === '.ovpn') return 'openvpn'
  return null
}

/**
 * The type once the file is read, like profileimport.TypeForContent
 * (ADR-0080): a .conf with an [Interface] section is WireGuard, one with
 * OpenVPN directives (client, remote, dev tun/tap, proto, tls-client, <ca>)
 * is OpenVPN — OpenVPN GUI on Windows keeps its profiles as .conf.
 */
export function typeForContent(fileName: string, text: string): 'wireguard' | 'openvpn' | null {
  const byName = typeForFile(fileName)
  if (byName !== 'wireguard') return byName
  const lines = text.split('\n').map((l) => l.trim().toLowerCase())
  if (lines.includes('[interface]')) return 'wireguard'
  const openvpn = lines.some((l) => {
    const [first = '', second = ''] = l.split(/\s+/)
    return ['client', 'remote', 'proto', 'tls-client', '<ca>'].includes(first) || (first === 'dev' && /^(tun|tap)/.test(second))
  })
  return openvpn ? 'openvpn' : 'wireguard'
}

/** A profile name from the file name, valid for both the profile and the stored copy, never taken. */
export function profileNameFromFile(fileName: string, taken: readonly string[]): string {
  return uniqueName(importName('', fileName), taken, '-')
}

/**
 * The .ovpn asks for a user and password (an `auth-user-pass` directive
 * outside a comment). Checked here only to show the fields; OpenVPN and
 * the helper's validator stay the authority.
 */
export function ovpnNeedsCredentials(text: string): boolean {
  return /^[ \t]*auth-user-pass(?:[ \t]|$)/m.test(text)
}

// ── "Sitio, IP o programa" (policy.ClassifyTargets) ───────────────────────

export type EntryKind = 'domain' | 'ip' | 'app'

export interface Entry {
  kind: EntryKind
  /** What the rule stores: a host or *.host, an IP or CIDR, a program. */
  value: string
  /** How it reads in sentences ("claude.ai", "teams.exe"). */
  label: string
}

/** Why an input cannot be used (each has a message + what to do). */
export type EntryProblem = 'empty' | 'appEmpty' | 'invalid'

/** Valid, but probably not what was meant: a hint, never a refusal (the CLI accepts it too). */
export type EntryHint = 'noDot' | 'exe'

export type Classified =
  | { ok: true; entry: Entry; hint?: EntryHint }
  | { ok: false; problem: EntryProblem; value: string }

/** net.ParseIP(s).String() for the forms the engine accepts (IPv6 compressed, lowercase). */
function canonicalIP(s: string): string | null {
  if (isIPv4(s)) return s
  if (!isIPv6(s)) return null
  try {
    return new URL(`http://[${s}]/`).hostname.slice(1, -1)
  } catch {
    return null
  }
}

/** policy.ipOrCIDR: a bare IP (canonical) or a CIDR as written. */
function ipOrCIDR(s: string): string | null {
  const ip = canonicalIP(s)
  if (ip) return ip
  return s.includes('/') && isIpOrCidr(s) ? s : null
}

/** policy.hostOf: the bare host of a URL or host[:port][/path], lowercased, no trailing dot. */
export function hostOf(raw: string): string {
  let s = raw
  if (s.includes('://')) {
    try {
      s = new URL(s).hostname
    } catch {
      return ''
    }
    if (!s) return ''
  }
  const cut = s.search(/[/?#]/)
  if (cut >= 0) s = s.slice(0, cut)
  // net.SplitHostPort: "[v6]:port" or exactly one colon.
  const bracketed = /^\[([^\]]*)\]:[^:]*$/.exec(s)
  if (bracketed) s = bracketed[1] ?? ''
  else if (s.split(':').length === 2) s = s.slice(0, s.indexOf(':'))
  s = s.replace(/^[[\]]+|[[\]]+$/g, '')
  if (s.endsWith('.')) s = s.slice(0, -1)
  return s.trim().toLowerCase()
}

/** One entry, read exactly like policy.ClassifyTargets reads it. */
export function classifyEntry(raw: string): Classified {
  const s = raw.trim()
  if (!s) return { ok: false, problem: 'empty', value: '' }
  if (s.slice(0, 4).toLowerCase() === 'app:') {
    const value = s.slice(4).trim()
    return value ? { ok: true, entry: { kind: 'app', value, label: nameSource(value) } } : { ok: false, problem: 'appEmpty', value: raw }
  }
  const range = ipOrCIDR(s)
  if (range) return { ok: true, entry: { kind: 'ip', value: range, label: range } }
  const slash = s.indexOf('/')
  // An IP with a bad prefix length (10.0.0.0/40) is refused, not read as a path.
  if (slash > 0 && !s.includes('://') && canonicalIP(s.slice(0, slash))) return { ok: false, problem: 'invalid', value: raw }
  const target = hostOf(s)
  const ip = canonicalIP(target)
  if (ip) return { ok: true, entry: { kind: 'ip', value: ip, label: ip } }
  const host = normalizeDomain(target)
  if (!host || host.length > 253 || !isDomainPattern(host)) return { ok: false, problem: 'invalid', value: raw }
  const entry: Entry = { kind: 'domain', value: host, label: host.replace(/^\*\./, '') }
  if (/\.exe$/.test(host)) return { ok: true, entry, hint: 'exe' }
  if (!host.includes('.')) return { ok: true, entry, hint: 'noDot' }
  return { ok: true, entry }
}

/** Split each line independently. Quotes force a single entry. An unquoted
 * app takes the rest of its line; commas inside URL paths stay literal
 * unless followed by whitespace. */
export function readDestinations(input: string) {
  const checked: { line: number; result: Classified }[] = []
  for (const [index, line] of input.split(/\r\n?|\n/).entries()) {
    let rest = line.trim()
    while (rest) {
      const quoted = /^"([^"]+)"\s*(?:,|$)/.exec(rest)
      if (quoted) {
        checked.push({ line: index + 1, result: classifyEntry(quoted[1]!) })
        rest = rest.slice(quoted[0].length).trim()
        continue
      }
      const app = /^app:/i.test(rest)
      const inPath = /^[a-z][a-z0-9+.-]*:\/\/[^/?#,]+[/?#]/i.test(rest)
      const comma = app ? -1 : rest.search(inPath ? /,(?=\s|$)/ : /,/)
      const raw = (comma < 0 ? rest : rest.slice(0, comma)).trim()
      if (raw) checked.push({ line: index + 1, result: classifyEntry(raw) })
      rest = comma < 0 ? '' : rest.slice(comma + 1).trim()
    }
  }
  return checked
}

export interface RuleMatch {
  domains: string[]
  ip_ranges: string[]
  apps: string[]
}

export interface Targets extends RuleMatch {
  /** The first entry in its stored form (names the rule). */
  first: string
  invalid: string[]
}

/** The rule's match lists; "incluye subdominios" adds *.host after each host. Duplicates dropped, order kept. */
export function matchOf(entries: readonly Entry[], includeSubdomains: boolean): RuleMatch {
  const domains = new Set<string>()
  const ranges = new Set<string>()
  const apps = new Set<string>()
  for (const e of entries) {
    if (e.kind === 'ip') ranges.add(e.value)
    else if (e.kind === 'app') apps.add(e.value)
    else {
      domains.add(e.value)
      if (includeSubdomains && !e.value.startsWith('*.')) domains.add(`*.${e.value}`)
    }
  }
  return { domains: [...domains], ip_ranges: [...ranges], apps: [...apps] }
}

/** policy.ClassifyTargets: the whole list, with the entries none of the forms accept. */
export function classifyTargets(entries: readonly string[], includeSubdomains: boolean): Targets {
  const ok: Entry[] = []
  const invalid: string[] = []
  for (const raw of entries) {
    const c = classifyEntry(raw)
    if (c.ok) ok.push(c.entry)
    else if (c.problem !== 'empty') invalid.push(raw)
  }
  return { ...matchOf(ok, includeSubdomains), first: ok[0]?.value ?? '', invalid }
}

/** What to ask the daemon about for the preview: a host or the first address of a range. */
export function probeOf(entry: Entry): { target: string; app: string } {
  if (entry.kind === 'app') return { target: '', app: entry.value }
  if (entry.kind === 'ip') return { target: entry.value.split('/')[0] ?? entry.value, app: '' }
  return { target: entry.label, app: '' }
}

// ── Which VPN ─────────────────────────────────────────────────────────────

interface TunnelLike {
  Name: string
  Status: string
}

/**
 * "sale por" starts on: the deep link's profile when it exists, else the
 * only profile, else the first connected one, else nothing (the user picks).
 */
export function defaultVia(profiles: readonly string[], tunnels: readonly TunnelLike[], wanted?: string): string {
  if (wanted && profiles.includes(wanted)) return wanted
  if (profiles.length === 1) return profiles[0] ?? ''
  const up = tunnels.find((tn) => tn.Status === 'connected' && profiles.includes(tn.Name))
  return up?.Name ?? ''
}

// ── The rule draft (kept while the sheet adds a missing VPN first) ────────

export interface RuleDraft {
  input: string
  subs: boolean
  via: string
  onDown: TunnelDownChoice
  /** '' = the generated name. */
  name: string
}

export function emptyRuleDraft(input = '', via = ''): RuleDraft {
  return { input, subs: true, via, onDown: 'inherit', name: '' }
}
