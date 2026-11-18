import { describe, expect, it } from 'vitest'
import { format, messages, type MessageKey } from '../../i18n/messages'
import {
  emptyDraft,
  findOverlaps,
  isActiveAt,
  isIpOrCidr,
  normalizeDomain,
  isDomainPattern,
  routeOutcome,
  scheduleToPayload,
  testSuggestions,
  toPayload,
  validatePolicy,
  validateSchedule,
  emptySchedule,
  type ExplainResult,
  type Policy,
} from './model'
import { describeSchedule } from './preview'

const tES = (k: MessageKey, v?: Record<string, string | number>) => format(messages.es[k], v)
const tEN = (k: MessageKey, v?: Record<string, string | number>) => format(messages.en[k], v)

const ctx = { names: ['Trabajo'], profiles: ['oficina', 'casa'] }

function policy(name: string, via: string, domains: string[] = [], ranges: string[] = []): Policy {
  return { name, via, domains, ip_ranges: ranges, apps: [], on_tunnel_down: 'allow' }
}

describe('policy validation (mirrors the daemon)', () => {
  it('accepts exact names and one leading *. label; rejects other globs', () => {
    expect(isDomainPattern('miempresa.com')).toBe(true)
    expect(isDomainPattern('*.miempresa.com')).toBe(true)
    expect(isDomainPattern('intranet')).toBe(true)
    expect(isDomainPattern('foo.*.com')).toBe(false)
    expect(isDomainPattern('*miempresa.com')).toBe(false)
    expect(isDomainPattern('**.x.com')).toBe(false)
  })

  it('cleans pasted URLs to the host', () => {
    expect(normalizeDomain('https://Git.MiEmpresa.com:8443/repo?x=1')).toBe('git.miempresa.com')
    expect(normalizeDomain('example.com.')).toBe('example.com')
  })

  it('accepts IPs and CIDRs like net.ParseCIDR, IPv6 included', () => {
    for (const ok of ['10.0.0.0/8', '192.0.2.10', '10.2.0.2/32', '2001:db8::/32', '::1']) expect(isIpOrCidr(ok), ok).toBe(true)
    for (const bad of ['10.0.0/8', '300.1.1.1', '10.0.0.0/33', '010.0.0.1', '10.0.0.0/8/1', 'abc', '2001:db8::/129']) expect(isIpOrCidr(bad), bad).toBe(false)
  })

  it('names the problem per field and the first bad entry', () => {
    const e = validatePolicy({ ...emptyDraft(), name: 'trabajo', domains: 'ok.com\nfoo.*.com', ipRanges: '10.0.0/8' }, ctx)
    expect(e.name).toEqual({ code: 'duplicate', value: 'trabajo' })
    expect(e.via).toEqual({ code: 'required' })
    expect(e.domains).toEqual({ code: 'badDomain', value: 'foo.*.com', line: 2 })
    expect(e.ipRanges).toEqual({ code: 'badRange', value: '10.0.0/8', line: 1 })
  })

  it('needs something to match, and block needs a destination (not only apps)', () => {
    expect(validatePolicy({ ...emptyDraft('oficina'), name: 'x' }, ctx).domains?.code).toBe('noMatch')
    const appsOnly = validatePolicy({ ...emptyDraft('oficina'), name: 'x', apps: 'chrome.exe', onDown: 'block' }, ctx)
    expect(appsOnly.onDown?.code).toBe('blockAppsOnly')
    expect(validatePolicy({ ...emptyDraft('oficina'), name: 'x', apps: 'chrome.exe' }, ctx)).toEqual({})
  })

  it('renaming a rule to its own name is not a duplicate; no profiles is its own error', () => {
    const d = { ...emptyDraft('oficina'), name: 'Trabajo', domains: 'a.com' }
    expect(validatePolicy(d, { ...ctx, editing: 'Trabajo' })).toEqual({})
    expect(validatePolicy(d, { names: [], profiles: [] }).via?.code).toBe('noProfiles')
  })

  it('builds the API body: one entry per line or comma, deduplicated, apps keep spaces', () => {
    const p = toPayload({
      name: ' Web ',
      via: 'oficina',
      domains: 'https://a.com/x, a.com\n*.a.com',
      ipRanges: '10.0.0.0/8 10.0.0.0/8',
      apps: 'C:\\Program Files\\App\\app.exe\n',
      onDown: 'connect',
    })
    expect(p).toEqual({
      name: 'Web',
      via: 'oficina',
      on_tunnel_down: 'connect',
      match: { domains: ['a.com', '*.a.com'], ip_ranges: ['10.0.0.0/8'], apps: ['C:\\Program Files\\App\\app.exe'] },
    })
  })
})

describe('overlaps', () => {
  it('same domain and wildcard: the first rule (evaluation order) wins', () => {
    const o = findOverlaps([policy('A', 'oficina', ['*.corp.com']), policy('B', 'casa', ['git.corp.com', 'corp.com'])])
    expect(o).toHaveLength(1)
    expect(o[0]).toMatchObject({ kind: 'wildcard', value: 'git.corp.com', winner: { policy: 'A', via: 'oficina' } })
    const same = findOverlaps([policy('A', 'oficina', ['x.com']), policy('B', 'casa', ['x.com'])])
    expect(same[0]?.kind).toBe('sameDomain')
  })

  it('nested ranges: the most specific wins whatever the order', () => {
    const o = findOverlaps([policy('Todo', 'oficina', [], ['10.0.0.0/8']), policy('Lab', 'casa', [], ['10.2.0.0/16'])])
    expect(o[0]).toMatchObject({ kind: 'nestedRange', value: '10.2.0.0/16', winner: { policy: 'Lab' }, other: { policy: 'Todo' } })
  })

  it('ignores rules on the same VPN and ranges that do not touch', () => {
    expect(findOverlaps([policy('A', 'oficina', ['x.com']), policy('B', 'oficina', ['x.com'])])).toEqual([])
    expect(findOverlaps([policy('A', 'oficina', [], ['10.0.0.0/16']), policy('B', 'casa', [], ['10.1.0.0/16'])])).toEqual([])
  })
})

describe('test a destination', () => {
  const ex = (over: Partial<ExplainResult>): ExplainResult => ({
    Target: 'git.corp.com',
    Kind: 'domain',
    Matched: true,
    Via: 'oficina',
    RuleName: 'A',
    Reason: '',
    ReasonCode: 'matched_domain',
    ReasonArgs: ['*.corp.com'],
    DefaultTo: 'default route',
    ...over,
  })

  it('combines the match, the VPN state and on_tunnel_down', () => {
    expect(routeOutcome(ex({}), 'allow', 'connected')).toBe('via')
    expect(routeOutcome(ex({}), 'allow', 'disconnected')).toBe('normal')
    expect(routeOutcome(ex({}), 'block', undefined)).toBe('blocked')
    expect(routeOutcome(ex({}), 'connect', 'error')).toBe('connect')
    expect(routeOutcome(ex({}), 'allow', 'no_response')).toBe('noResponse')
    expect(routeOutcome(ex({ Matched: false }), 'block', 'connected')).toBe('none')
    expect(routeOutcome(ex({ Kind: 'app' }), 'allow', 'connected')).toBe('app')
  })

  it('suggests things to try from the user’s own rules', () => {
    expect(testSuggestions([policy('A', 'o', ['*.corp.com']), policy('B', 'o', [], ['10.2.0.0/16']), policy('C', 'o', [], ['192.0.2.7'])])).toEqual([
      'www.corp.com',
      '10.2.0.1',
      '192.0.2.7',
    ])
  })
})

describe('schedules', () => {
  it('previews the rule in plain words (ES and EN)', () => {
    const office = { name: 'o', profiles: ['oficina'], weekdays: ['mon', 'tue', 'wed', 'thu', 'fri'], start: '09:00', end: '18:00' }
    expect(describeSchedule(office, tES, 'es')).toBe(
      'De lunes a viernes, de 9:00 a 18:00: conecta «oficina». Fuera de ese horario, Kongtrol la desconecta.',
    )
    expect(describeSchedule(office, tEN, 'en')).toBe(
      'Monday to Friday, from 9:00 to 18:00: connects “oficina”. Outside that time, Kongtrol disconnects it.',
    )
    const night = { name: 'n', profiles: ['a', 'b'], weekdays: ['mon', 'wed', 'fri'], start: '22:00', end: '06:00' }
    expect(describeSchedule(night, tES, 'es')).toBe(
      'Lunes, miércoles y viernes, de 22:00 a 6:00 del día siguiente: conecta «a» y «b». Fuera de ese horario, Kongtrol las desconecta.',
    )
    expect(describeSchedule({ name: 'r', profiles: ['x'], weekdays: ['tue', 'wed', 'thu'] }, tES, 'es')).toBe(
      'De martes a jueves, todo el día: conecta «x». Fuera de ese horario, Kongtrol la desconecta.',
    )
    expect(describeSchedule({ name: 'always', profiles: ['x'], weekdays: [] }, tEN, 'en')).toBe(
      'Kongtrol keeps “x” connected all the time.',
    )
  })

  it('is active exactly when the daemon would be (overnight wraps, weekday of today)', () => {
    const rule = { name: 'n', profiles: ['a'], weekdays: ['fri'], start: '22:00', end: '06:00' }
    expect(isActiveAt(rule, new Date(2026, 10, 20, 23, 0))).toBe(true) // Fri 23:00
    expect(isActiveAt(rule, new Date(2026, 10, 21, 1, 0))).toBe(false) // Sat 01:00: Saturday not ticked
    expect(isActiveAt(rule, new Date(2026, 10, 20, 12, 0))).toBe(false)
    expect(isActiveAt({ name: 'd', profiles: ['a'], start: '09:00', end: '18:00' }, new Date(2026, 10, 18, 18, 0))).toBe(false)
  })

  it('validates name, VPN and times; equal times never open', () => {
    const c = { names: ['Oficina'], profiles: ['vpn'] }
    expect(validateSchedule({ ...emptySchedule(), name: 'oficina' }, c)).toMatchObject({
      name: { code: 'duplicate' },
      profiles: { code: 'pickProfile' },
    })
    expect(validateSchedule({ ...emptySchedule(), name: 'x', profiles: ['vpn'], start: '10:00', end: '10:00' }, c).end?.code).toBe('sameTime')
    expect(validateSchedule({ ...emptySchedule(), name: 'x', profiles: ['vpn'], start: '', end: '25:00' }, c)).toMatchObject({
      start: { code: 'required' },
      end: { code: 'badTime' },
    })
    expect(validateSchedule({ ...emptySchedule(), name: 'x', profiles: ['vpn'], allDay: true, start: '' }, c)).toEqual({})
  })

  it('stores every day as no weekdays and all day as no times', () => {
    expect(
      scheduleToPayload({ name: ' a ', profiles: ['v'], days: ['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun'], allDay: true, start: '09:00', end: '18:00' }),
    ).toEqual({ name: 'a', profiles: ['v'], weekdays: [], start: '', end: '' })
  })
})
