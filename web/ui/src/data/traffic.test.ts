import { describe, expect, it, vi } from 'vitest'
import { tunnel } from '../test/fakes'
import {
  MAX_SAMPLES,
  connectedSignature,
  createTrafficHistory,
  effectiveFilter,
  explainLookup,
  filterPolicies,
  filterRoutes,
  formatBytes,
  formatDuration,
  groupTunnelRoutes,
  isDomain,
  lookupKey,
  parseLookup,
  policyResolved,
  routeFilterOptions,
  routeOriginIndex,
  tunnelAnchor,
  type Policy,
  type RouteRow,
  type TunnelRoutesResponse,
} from './traffic'

const policy = (name: string, via: string, extra: Partial<Policy> = {}): Policy => ({
  name,
  via,
  domains: null,
  ip_ranges: null,
  apps: null,
  on_tunnel_down: 'allow',
  resolved_cidrs: null,
  ...extra,
})

const routes: TunnelRoutesResponse = {
  rest_via: 'Wi-Fi',
  tunnels: [
    {
      name: 'work',
      connected: true,
      interface: 'tun0',
      all_traffic: false,
      total: 3,
      counts: { server: 1, policy: 2 },
      routes: [
        { destination: '10.0.0.0/8', origin: 'server' },
        { destination: '1.1.1.1/32', origin: 'policy', policy: 'ai' },
        { destination: '2.2.2.2/32', origin: 'policy', policy: 'ai' },
      ],
    },
  ],
}

describe('lookup', () => {
  it('parses app: as a program and anything else as a target', () => {
    expect(parseLookup('  ')).toBeNull()
    expect(parseLookup('app: chrome.exe')).toEqual({ app: 'chrome.exe' })
    expect(parseLookup('APP:code')).toEqual({ app: 'code' })
    expect(parseLookup(' https://claude.ai/x ')).toEqual({ target: 'https://claude.ai/x' })
    expect(lookupKey({ app: 'code' })).toBe('app:code')
  })

  it('explains what happens now, not only which rule matched', () => {
    const tunnels = [tunnel('work', 'connected'), tunnel('home', 'disconnected')]
    const policies = [policy('ai', 'work'), policy('corp', 'home', { on_tunnel_down: 'block' }), policy('tv', 'home')]
    const ctx = { tunnels, policies, routes }
    const hit = (via: string, rule: string) => ({ target: 'x', via, rule, matched: true })
    expect(explainLookup(hit('work', 'ai'), ctx)).toEqual({ kind: 'tunnel', via: 'work', rule: 'ai' })
    expect(explainLookup(hit('home', 'corp'), ctx)).toEqual({ kind: 'blocked', via: 'home', rule: 'corp' })
    expect(explainLookup(hit('home', 'tv'), ctx)).toEqual({ kind: 'tunnel_down', via: 'home', rule: 'tv' })
    const miss = { target: 'x', via: '', rule: '', matched: false }
    expect(explainLookup(miss, ctx)).toEqual({ kind: 'normal', iface: 'Wi-Fi' })
    expect(explainLookup(miss, { ...ctx, routes: { ...routes, rest_via_tunnel: 'work' } })).toEqual({ kind: 'all_traffic', via: 'work' })
    expect(explainLookup(miss, { ...ctx, routes: undefined })).toEqual({ kind: 'normal', iface: '' })
    // A program is recognised, never routed (ADR-0113): never "tunnel" / "normal".
    expect(explainLookup({ ...hit('work', 'ai'), app: 'claude.exe' }, ctx)).toEqual({ kind: 'app', rule: 'ai' })
    expect(explainLookup({ ...miss, app: 'claude.exe' }, ctx)).toEqual({ kind: 'app', rule: '' })
  })

  it('recognises domains (for the DNS detail) but not IPs', () => {
    expect(isDomain('claude.ai')).toBe(true)
    expect(isDomain('a.b.example.com')).toBe(true)
    expect(isDomain('10.0.0.1')).toBe(false)
    expect(isDomain('localhost')).toBe(false)
  })

  it('filters rules by name, via, domain, range or app (app: prefix ignored)', () => {
    const list = [
      policy('ai', 'work', { domains: ['claude.ai'] }),
      policy('corp', 'office', { ip_ranges: ['10.0.0.0/8'], apps: ['teams.exe'] }),
    ]
    expect(filterPolicies(list, '').map((p) => p.name)).toEqual(['ai', 'corp'])
    expect(filterPolicies(list, 'CLAUDE').map((p) => p.name)).toEqual(['ai'])
    expect(filterPolicies(list, '10.0.').map((p) => p.name)).toEqual(['corp'])
    expect(filterPolicies(list, 'app:teams').map((p) => p.name)).toEqual(['corp'])
    expect(filterPolicies(list, 'nothing')).toEqual([])
  })

  it('uses the policy’s own resolved IPs (with expiry) and falls back to the profile’s', () => {
    const own = [{ cidr: '1.1.1.1/32', host: 'claude.ai', last_seen: 'a', expires_at: 'b' }]
    expect(policyResolved(policy('a', 'w', { resolved: own, resolved_cidrs: ['9.9.9.9/32'] }))).toBe(own)
    expect(policyResolved(policy('a', 'w', { resolved_cidrs: ['9.9.9.9/32'] }))).toEqual([
      { cidr: '9.9.9.9/32', last_seen: '', expires_at: '' },
    ])
  })
})

describe('routes', () => {
  const table: RouteRow[] = [
    { destination: '0.0.0.0/0', gateway: '192.168.1.1', interface: 'Wi-Fi', metric: 25, is_default: true, policy_name: 'default' },
    { destination: '10.0.0.0/8', gateway: '', interface: 'tun0', metric: 1, is_default: false },
    { destination: '1.1.1.1/32', gateway: '', interface: 'tun0', metric: 10, is_default: false, policy_name: 'ai' },
    { destination: '172.16.0.0/12', gateway: '', interface: 'eth1', metric: 5, is_default: false },
  ]
  const idx = routeOriginIndex(routes)

  it('joins OS routes with the tunnel that carries them (iface|dest, then dest)', () => {
    expect(idx.get('tun0|10.0.0.0/8')).toEqual({ profile: 'work', origin: 'server', policy: '' })
    expect(idx.get('1.1.1.1/32')?.policy).toBe('ai')
  })

  it('filters by default, profile, origin and policy; unknown filters fall back to all', () => {
    const pick = (f: string) => filterRoutes(table, f, idx).map((r) => r.destination)
    expect(pick('all')).toHaveLength(4)
    expect(pick('default')).toEqual(['0.0.0.0/0'])
    expect(pick('profile:work')).toEqual(['10.0.0.0/8', '1.1.1.1/32'])
    expect(pick('origin:server')).toEqual(['10.0.0.0/8'])
    expect(pick('policy:ai')).toEqual(['1.1.1.1/32'])
    const options = routeFilterOptions([policy('ai', 'work'), policy('ai', 'work')], routes)
    expect(options.map((o) => o.value)).toEqual([
      'all',
      'default',
      'profile:work',
      'origin:server',
      'origin:policy',
      'origin:vpn',
      'policy:ai',
    ])
    expect(effectiveFilter('profile:work', options)).toBe('profile:work')
    expect(effectiveFilter('profile:gone', options)).toBe('all')
  })

  it('groups consecutive routes by origin and rule', () => {
    expect(groupTunnelRoutes(routes.tunnels[0]!.routes)).toEqual([
      { origin: 'server', policy: '', destinations: ['10.0.0.0/8'] },
      { origin: 'policy', policy: 'ai', destinations: ['1.1.1.1/32', '2.2.2.2/32'] },
    ])
  })

  it('changes the connected signature only when a tunnel goes up or down', () => {
    const a = [tunnel('a', 'connected', { BytesSent: 1 }), tunnel('b', 'disconnected')]
    const b = [tunnel('a', 'connected', { BytesSent: 99 }), tunnel('b', 'connecting')]
    expect(connectedSignature(a)).toBe(connectedSignature(b))
    expect(connectedSignature([tunnel('a', 'connected'), tunnel('b', 'connected')])).not.toBe(connectedSignature(a))
  })

  it('builds safe anchors for any profile name', () => {
    expect(tunnelAnchor('Mi VPN #1')).toBe('tunnel-Mi_VPN__1')
  })
})

describe('traffic history', () => {
  const at = (s: number) => new Date(Date.UTC(2026, 10, 18, 12, 0, s)).toISOString()
  const frame = (s: number, rx: number, tx: number, extra = {}) =>
    tunnel('work', 'connected', { UpdatedAt: at(s), BytesReceived: rx, BytesSent: tx, ...extra })

  it('turns cumulative counters into per-second rates, clamped at zero', () => {
    const h = createTrafficHistory()
    h.record([frame(0, 1000, 100)])
    expect(h.get('work').points).toEqual([]) // first frame is only the baseline
    h.record([frame(5, 6000, 600)])
    h.record([frame(10, 2000, 600)]) // counter reset → 0, never negative
    const pts = h.get('work').points
    expect(pts.map((p) => [p.rx, p.tx])).toEqual([
      [1000, 100],
      [0, 0],
    ])
  })

  it('ignores frames older than the last one (a late REST answer)', () => {
    const h = createTrafficHistory()
    h.record([frame(10, 10_000, 0)])
    h.record([frame(5, 5000, 0)])
    h.record([frame(15, 15_000, 0)])
    expect(h.get('work').points.map((p) => p.rx)).toEqual([1000])
  })

  it('ignores the same frame twice and keeps at most MAX_SAMPLES points', () => {
    const h = createTrafficHistory()
    for (let s = 0; s <= MAX_SAMPLES + 5; s++) {
      h.record([frame(s, s * 10, 0)])
      h.record([frame(s, s * 10, 0)])
    }
    expect(h.get('work').points).toHaveLength(MAX_SAMPLES)
  })

  it('records latency only when it changes', () => {
    const h = createTrafficHistory()
    h.record([frame(0, 0, 0, { LatencyMS: 20 })])
    h.record([frame(5, 0, 0, { LatencyMS: 20 })])
    h.record([frame(10, 0, 0, { LatencyMS: 25 })])
    h.record([frame(15, 0, 0, { LatencyMS: 0 })])
    expect(h.get('work').latency.map((l) => l.ms)).toEqual([20, 25])
  })

  it('notifies only the tunnels whose frame changed and starts over after a disconnect', () => {
    const h = createTrafficHistory()
    const work = vi.fn()
    const home = vi.fn()
    h.subscribe('work', work)
    h.subscribe('home', home)
    const homeFrame = tunnel('home', 'connected', { UpdatedAt: at(0) })
    h.record([frame(0, 0, 0), homeFrame])
    work.mockClear()
    home.mockClear()
    h.record([frame(5, 500, 0), homeFrame]) // only work moved
    expect(work).toHaveBeenCalledTimes(1)
    expect(home).not.toHaveBeenCalled()
    const before = h.get('home')
    h.record([frame(10, 900, 0), homeFrame])
    expect(h.get('home')).toBe(before) // identity kept → no re-render
    h.record([tunnel('work', 'disconnected'), homeFrame])
    expect(h.get('work').points).toEqual([])
  })
})

describe('formatting', () => {
  it('formats bytes in SI units with the page language', () => {
    expect(formatBytes(0, 'es')).toBe('0 B')
    expect(formatBytes(999, 'en')).toBe('999 B')
    expect(formatBytes(1500, 'en')).toBe('1.5 kB')
    expect(formatBytes(1500, 'es')).toBe('1,5 kB')
    expect(formatBytes(2_500_000_000, 'en')).toBe('2.5 GB')
  })

  it('formats durations compactly', () => {
    expect(formatDuration(45)).toBe('45 s')
    expect(formatDuration(3725)).toBe('1 h 2 min')
    expect(formatDuration(90_000)).toBe('1 d 1 h')
  })
})
