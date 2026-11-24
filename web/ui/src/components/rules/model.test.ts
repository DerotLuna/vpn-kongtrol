import { describe, expect, it } from 'vitest'
import { format, messages, type MessageKey } from '../../i18n/messages'
import {
  findOverlaps,
  orderMatters,
  overlapOrder,
  priorityLink,
  isActiveAt,
  isIpOrCidr,
  normalizeDomain,
  isDomainPattern,
  routeOutcome,
  scheduleToPayload,
  testSuggestions,
  validateSchedule,
  emptySchedule,
  type ExplainResult,
  type Policy,
} from './model'
import { describeSchedule, overlapOrderText } from './preview'
import { draftFromPolicy, validatePolicy } from './editorModel'

const tES = (k: MessageKey, v?: Record<string, string | number>) => format(messages.es[k], v)
const tEN = (k: MessageKey, v?: Record<string, string | number>) => format(messages.en[k], v)

const ctx = { names: ['Trabajo'], profiles: ['oficina', 'casa'] }
const draft = { name: 'x', via: 'oficina', destinations: '', onDown: 'inherit' } as const

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
    expect(normalizeDomain('*.café.example.')).toBe('*.xn--caf-dma.example')
    expect(normalizeDomain('https://café.example:8443/site')).toBe('xn--caf-dma.example')
  })

  it('accepts IPs and CIDRs like net.ParseCIDR, IPv6 included', () => {
    for (const ok of ['10.0.0.0/8', '192.0.2.10', '10.2.0.2/32', '2001:db8::/32', '::1']) expect(isIpOrCidr(ok), ok).toBe(true)
    for (const bad of ['10.0.0/8', '300.1.1.1', '10.0.0.0/33', '010.0.0.1', '10.0.0.0/8/1', 'abc', '2001:db8::/129']) expect(isIpOrCidr(bad), bad).toBe(false)
  })

  it('names the problem per field and the first bad entry', () => {
    const { errors: e, payload } = validatePolicy({ ...draft, via: '', name: 'trabajo', destinations: 'ok.com\nfoo.*.com' }, ctx)
    expect(e.name).toEqual({ code: 'duplicate', value: 'trabajo' })
    expect(e.via).toEqual({ code: 'required' })
    expect(e.destinations).toEqual({ code: 'invalid', value: 'foo.*.com', line: 2 })
    expect(payload).toBeNull()
  })

  it('needs something to match, and block needs a destination (not only apps)', () => {
    expect(validatePolicy(draft, ctx).errors.destinations?.code).toBe('noMatch')
    const app = { ...draft, destinations: 'app:chrome.exe' }
    expect(validatePolicy({ ...app, onDown: 'block' }, ctx).errors.onDown?.code).toBe('blockAppsOnly')
    expect(validatePolicy(app, ctx).errors).toEqual({})
  })

  it('renaming a rule to its own name is not a duplicate; no profiles is its own error', () => {
    const d = { ...draft, name: 'Trabajo', destinations: 'a.com' }
    expect(validatePolicy(d, { ...ctx, editing: 'Trabajo' }).errors).toEqual({})
    expect(validatePolicy(d, { names: [], profiles: [] }).errors.via?.code).toBe('noProfiles')
  })

  it('builds the API body: one entry per line or comma, deduplicated, apps keep spaces', () => {
    const { payload } = validatePolicy({
      ...draft,
      name: ' Web ',
      destinations: 'https://a.com/x,y, a.com\r\n*.a.com,10.0.0.0/8,10.0.0.0/8\napp:C:\\Program Files\\App, Inc\\app.exe\n',
      onDown: 'connect',
    }, ctx)
    expect(payload).toEqual({
      name: 'Web',
      via: 'oficina',
      on_tunnel_down: 'connect',
      match: { domains: ['a.com', '*.a.com'], ip_ranges: ['10.0.0.0/8'], apps: ['C:\\Program Files\\App, Inc\\app.exe'] },
    })
  })

  it('keeps a quoted app in a comma-separated list and deduplicates within each destination type', () => {
    const { payload } = validatePolicy({ ...draft, destinations: '"app:Code, Inc.exe",a.com,a.com,app:a.com' }, ctx)
    expect(payload?.match).toEqual({ domains: ['a.com'], ip_ranges: [], apps: ['Code, Inc.exe', 'a.com'] })
  })

  it('opens a paused rule with its existing matches and leaves state out of an edit', () => {
    const saved = { ...policy('Work', 'oficina', ['*.corp.com'], ['10.0.0.0/8']), apps: ['Code, Inc.exe'], disabled: true }
    const { payload } = validatePolicy(draftFromPolicy(saved), ctx)
    expect(payload?.match).toEqual({ domains: saved.domains, ip_ranges: saved.ip_ranges, apps: saved.apps })
    expect(payload).not.toHaveProperty('disabled')
  })
})

describe('overlaps', () => {
  it('ignores paused rules', () => {
    const paused = { ...policy('A', 'oficina', ['x.com']), disabled: true }
    expect(findOverlaps([paused, policy('B', 'casa', ['x.com'])])).toEqual([])
  })
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

describe('order between overlapping rules (ADR-0108)', () => {
  const pol = (name: string, via: string, domains: string[] | null, ip_ranges: string[] | null = null): Policy => ({
    name,
    via,
    domains,
    ip_ranges,
    apps: null,
    on_tunnel_down: 'allow',
  })
  const list = [pol('A', 'oficina', ['git.example.com']), pol('B', 'casa', ['*.example.com']), pol('C', 'lab', null, ['10.2.0.0/16']), pol('D', 'casa', null, ['10.0.0.0/8'])]

  it('says whether priority, position or one of each decided; nested ranges are never the order', () => {
    const [domain, nested] = findOverlaps(list)
    expect(domain?.kind).toBe('wildcard')
    expect(nested?.kind).toBe('nestedRange')
    expect(overlapOrder(domain!, new Map())).toBe('position')
    expect(overlapOrder(domain!, new Map([['oficina', 10], ['casa', 10]]))).toBe('position')
    expect(overlapOrder(domain!, new Map([['oficina', 20], ['casa', 10]]))).toBe('priority')
    expect(overlapOrder(domain!, new Map([['oficina', 20]]))).toBe('mixed')
    expect(overlapOrder(nested!, new Map())).toBeNull()
  })

  it('marks only the rules in an overlap the order settles, and links to the priority field', () => {
    expect([...orderMatters(findOverlaps(list))].sort()).toEqual(['A', 'B'])
    expect(priorityLink('mi casa')).toBe('/profiles?edit=mi%20casa&focus=priority')
  })

  it('explains each case in one line', () => {
    const [domain] = findOverlaps(list)
    expect(overlapOrderText(domain!, 'position', new Map(), tES)).toBe('Ninguna de las dos VPN tiene prioridad: decide la posición en la lista.')
    expect(overlapOrderText(domain!, 'position', new Map([['oficina', 10], ['casa', 10]]), tES)).toMatch(/misma prioridad/)
    expect(overlapOrderText(domain!, 'priority', new Map([['oficina', 20], ['casa', 10]]), tES)).toBe('Lo decide la prioridad: oficina (20) va antes que casa (10).')
    expect(overlapOrderText(domain!, 'mixed', new Map([['oficina', 20]]), tES)).toMatch(/Pon prioridad a las dos/)
  })
})
