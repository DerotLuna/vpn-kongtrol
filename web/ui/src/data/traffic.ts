// Kongtrol UI — Tráfico (/traffic): reads, live samples and lookups · v1.0.0 · 2026-11-18
//
// Everything the Traffic page reads from the daemon, on the shared cache
// (ADR-0038): routes, what goes through each tunnel, policies (with their
// resolved IPs and expiry), "which VPN does X use?", DNS through a tunnel,
// the base network and per-profile stability history. Plus the live
// throughput samples, derived on this side from the tunnel feed the app
// already receives (the daemon keeps no per-tick history).
//
// Pure helpers (lookup parsing, filters, grouping, formatting) are exported
// so they are tested without rendering.
import { useEffect, useRef, useSyncExternalStore } from 'react'
import { useQuery, useQueryClient, type QueryClient } from '@tanstack/react-query'
import { apiCall } from '../api/client'
import type { TunnelMetrics } from '../api/types'
import type { Lang } from '../i18n/messages'
import { keys } from './keys'
import { useTunnelsQuery } from './resources'

// ── Daemon types (Go source in each comment) ─────────────────────────────

/** GET /api/v1/routes — internal/api/handlers_routes.go `routeDTO`. */
export interface RouteRow {
  destination: string
  gateway: string
  interface: string
  metric: number
  policy_name?: string
  policy_via?: string
  is_default: boolean
}

export type RouteOrigin = 'server' | 'policy' | 'vpn'

/** internal/api/handlers_tunnel_routes.go `tunnelRouteDTO`. */
export interface TunnelRoute {
  destination: string
  origin: RouteOrigin
  policy?: string
  gateway?: string
}

/** internal/api/handlers_tunnel_routes.go `tunnelRoutesDTO`. */
export interface TunnelRoutes {
  name: string
  connected: boolean
  interface?: string
  all_traffic: boolean
  total: number
  counts: Record<string, number>
  routes: TunnelRoute[]
}

/** GET /api/v1/tunnels/routes — `tunnelRoutesResponse`. */
export interface TunnelRoutesResponse {
  rest_via?: string
  rest_via_tunnel?: string
  tunnels: TunnelRoutes[]
}

/** internal/monitor/policyresolver.go `ResolvedIP`. */
export interface ResolvedIP {
  cidr: string
  host?: string
  last_seen: string
  expires_at: string
}

/** GET /api/v1/policies — internal/api/handlers_policy.go `policyDTO`. */
export interface Policy {
  name: string
  via: string
  domains: string[] | null
  ip_ranges: string[] | null
  apps: string[] | null
  on_tunnel_down: string
  /** Every resolved IP of the profile (legacy field). */
  resolved_cidrs: string[] | null
  /** This policy's own resolved IPs with expiry (newer daemons). */
  resolved?: ResolvedIP[] | null
}

/** GET /api/v1/resolve — `resolveDTO`. */
export interface ResolveResult {
  target: string
  app?: string
  via: string
  rule: string
  matched: boolean
}

/** GET /api/v1/dns/resolve — internal/api/handlers_policy.go `handleDNSResolve`. */
export interface DnsResolveResult {
  domain: string
  via: string
  ips: string[]
}

/** GET /api/v1/network/overview — internal/api/handlers_routes.go `overviewDTO`. */
export interface NetworkOverview {
  connected_tunnels: number
  default_routes: { destination: string; gateway: string; interface: string; metric: number }[] | null
  local_ips: string[] | null
  public_ip?: string
  public_ip_check_enabled: boolean
}

/** GET /api/v1/metrics/history — internal/monitor/collector.go `ProfileHistory`. */
export interface ProfileHistory {
  name: string
  samples: number
  connected_samples: number
  uptime_seconds: number
  reconnects: number
  jitter_ms: number
  last_latency_ms: number
}

// ── Cache keys (resources, not pages) ─────────────────────────────────────

export const trafficKeys = {
  routes: ['routes'] as const,
  tunnelRoutes: ['tunnelRoutes'] as const,
  policies: ['policies'] as const,
  overview: ['networkOverview'] as const,
  history: ['metricsHistory'] as const,
  resolve: (q: string) => ['resolve', q] as const,
  dns: (domain: string, via: string) => ['dnsResolve', domain, via] as const,
} as const

/** Same cadence as the classic panel (app.js:1231): route tables change rarely. */
export const TRAFFIC_POLL_MS = 30_000
/** After a tunnel goes up or down, give the OS a moment to settle its routes. */
export const ROUTE_SETTLE_MS = 600

const enc = encodeURIComponent

export const trafficApi = {
  routes: () => apiCall.get<RouteRow[] | null>('/api/v1/routes').then((r) => r ?? []),
  tunnelRoutes: () =>
    apiCall.get<TunnelRoutesResponse>('/api/v1/tunnels/routes').then((r) => ({ ...r, tunnels: r?.tunnels ?? [] })),
  // Same key and same result as data/security.ts (sorted by name), so both pages share one entry.
  policies: () =>
    apiCall.get<Policy[] | null>('/api/v1/policies').then((l) => [...(l ?? [])].sort((a, b) => a.name.localeCompare(b.name))),
  overview: () => apiCall.get<NetworkOverview>('/api/v1/network/overview'),
  history: () => apiCall.get<Record<string, ProfileHistory> | null>('/api/v1/metrics/history').then((h) => h ?? {}),
  resolve: (q: LookupQuery) =>
    apiCall.get<ResolveResult>(
      q.app ? `/api/v1/resolve?app=${enc(q.app)}` : `/api/v1/resolve?target=${enc(q.target ?? '')}`,
    ),
  dns: (domain: string, via: string) =>
    apiCall.get<DnsResolveResult>(`/api/v1/dns/resolve?domain=${enc(domain)}&via=${enc(via)}`),
}

// ── Hooks ────────────────────────────────────────────────────────────────

export function useTunnelRoutesQuery() {
  return useQuery({ queryKey: trafficKeys.tunnelRoutes, queryFn: trafficApi.tunnelRoutes, refetchInterval: TRAFFIC_POLL_MS })
}

/** The OS route table is slow to read on Windows: only while the developer view is open. */
export function useRoutesQuery(enabled: boolean) {
  return useQuery({ queryKey: trafficKeys.routes, queryFn: trafficApi.routes, refetchInterval: TRAFFIC_POLL_MS, enabled })
}

export function usePoliciesQuery() {
  return useQuery({ queryKey: trafficKeys.policies, queryFn: trafficApi.policies, refetchInterval: TRAFFIC_POLL_MS })
}

export function useNetworkOverviewQuery(enabled: boolean) {
  return useQuery({ queryKey: trafficKeys.overview, queryFn: trafficApi.overview, refetchInterval: TRAFFIC_POLL_MS, enabled })
}

export function useMetricsHistoryQuery() {
  return useQuery({ queryKey: trafficKeys.history, queryFn: trafficApi.history, refetchInterval: TRAFFIC_POLL_MS })
}

/** "¿Por dónde sale X?" — runs only for a submitted, non-empty query. */
export function useResolveQuery(input: string) {
  const q = parseLookup(input)
  return useQuery({
    queryKey: trafficKeys.resolve(lookupKey(q)),
    queryFn: () => trafficApi.resolve(q ?? {}),
    enabled: q !== null,
    staleTime: 0,
  })
}

/** Resolves a domain through the VPN's own DNS (what the tunnel will reach). On demand only. */
export function useDnsResolveQuery(domain: string, via: string, enabled: boolean) {
  return useQuery({
    queryKey: trafficKeys.dns(domain, via),
    queryFn: () => trafficApi.dns(domain, via),
    enabled: enabled && domain !== '' && via !== '',
    staleTime: 60_000,
  })
}

/** Names of the connected tunnels, joined: changes only when a tunnel goes up or down. */
export function connectedSignature(list: readonly TunnelMetrics[] | undefined): string {
  return (list ?? [])
    .filter((tn) => tn.Status === 'connected')
    .map((tn) => tn.Name)
    .join('\n')
}

/**
 * Route tables follow the tunnels: when one goes up or down (from any
 * surface), refresh routes, per-tunnel routes, policies and the base
 * network shortly after, instead of waiting for the 30 s poll.
 */
export function useRefreshRoutesOnTunnelChange(): void {
  const client = useQueryClient()
  const { data } = useTunnelsQuery()
  const sig = connectedSignature(data)
  const previous = useRef<string | null>(null)
  useEffect(() => {
    if (data === undefined) return
    if (previous.current === null || previous.current === sig) {
      previous.current = sig
      return
    }
    previous.current = sig
    const timer = setTimeout(() => {
      for (const key of [trafficKeys.tunnelRoutes, trafficKeys.routes, trafficKeys.policies, trafficKeys.overview]) {
        void client.invalidateQueries({ queryKey: key })
      }
    }, ROUTE_SETTLE_MS)
    return () => clearTimeout(timer)
  }, [sig, data, client])
}

// ── Lookup ("¿por dónde sale?") ───────────────────────────────────────────

export interface LookupQuery {
  target?: string
  app?: string
}

/** `app:chrome.exe` asks about a program; anything else is a site, IP, URL, rule or profile name. */
export function parseLookup(input: string): LookupQuery | null {
  const s = input.trim()
  if (!s) return null
  const m = /^app:\s*(.+)$/i.exec(s)
  if (m) return { app: m[1]!.trim() }
  return { target: s }
}

export function lookupKey(q: LookupQuery | null): string {
  if (!q) return ''
  return q.app ? `app:${q.app}` : (q.target ?? '')
}

export type LookupVerdict =
  | { kind: 'tunnel'; via: string; rule: string }
  | { kind: 'tunnel_down'; via: string; rule: string }
  | { kind: 'blocked'; via: string; rule: string }
  | { kind: 'all_traffic'; via: string }
  | { kind: 'normal'; iface: string }

export interface LookupContext {
  tunnels: readonly TunnelMetrics[]
  policies: readonly Policy[]
  routes: TunnelRoutesResponse | undefined
}

/**
 * Turns the daemon's answer into what actually happens now: a rule's VPN
 * may be off (then the traffic uses the normal connection, or is blocked
 * when the rule says on_tunnel_down: block), and without a rule a VPN that
 * took all traffic still carries it.
 */
export function explainLookup(res: ResolveResult, ctx: LookupContext): LookupVerdict {
  if (res.matched && res.via) {
    const up = ctx.tunnels.some((tn) => tn.Name === res.via && tn.Status === 'connected')
    if (up) return { kind: 'tunnel', via: res.via, rule: res.rule }
    const pol = ctx.policies.find((p) => p.name === res.rule)
    if (pol?.on_tunnel_down === 'block') return { kind: 'blocked', via: res.via, rule: res.rule }
    return { kind: 'tunnel_down', via: res.via, rule: res.rule }
  }
  if (ctx.routes?.rest_via_tunnel) return { kind: 'all_traffic', via: ctx.routes.rest_via_tunnel }
  return { kind: 'normal', iface: ctx.routes?.rest_via ?? '' }
}

/** True when the lookup is a domain name (so it can be resolved through the tunnel's DNS). */
export function isDomain(target: string): boolean {
  return /^(?=.{1,253}$)([a-z0-9-]+\.)+[a-z]{2,}$/i.test(target) && !/^[\d.]+$/.test(target)
}

/** Classic panel's policy filter (app.js:892): name, via, domains, IP ranges, apps. */
export function filterPolicies(policies: readonly Policy[], query: string): Policy[] {
  const q = query.trim().toLowerCase().replace(/^app:\s*/, '')
  if (!q) return [...policies]
  return policies.filter((p) =>
    [p.name, p.via, ...(p.domains ?? []), ...(p.ip_ranges ?? []), ...(p.apps ?? [])].some((v) =>
      String(v).toLowerCase().includes(q),
    ),
  )
}

/** A policy's resolved IPs: its own with expiry when the daemon sends them, else the profile's. */
export function policyResolved(p: Policy): ResolvedIP[] {
  if (p.resolved) return p.resolved
  return (p.resolved_cidrs ?? []).map((cidr) => ({ cidr, last_seen: '', expires_at: '' }))
}

// ── Routes ───────────────────────────────────────────────────────────────

export interface RouteInfo {
  profile: string
  origin: RouteOrigin
  policy: string
}

/** "iface|destination" (and bare destination) → the tunnel that carries it (app.js:755). */
export function routeOriginIndex(tr: TunnelRoutesResponse | undefined): Map<string, RouteInfo> {
  const idx = new Map<string, RouteInfo>()
  for (const tunnel of tr?.tunnels ?? []) {
    for (const r of tunnel.routes ?? []) {
      const info = { profile: tunnel.name, origin: r.origin, policy: r.policy ?? '' }
      idx.set(`${tunnel.interface ?? ''}|${r.destination}`, info)
      if (!idx.has(r.destination)) idx.set(r.destination, info)
    }
  }
  return idx
}

export function routeInfo(r: RouteRow, idx: Map<string, RouteInfo>): RouteInfo | null {
  return idx.get(`${r.interface || ''}|${r.destination}`) ?? idx.get(r.destination) ?? null
}

export type RouteFilter = string

export interface FilterOption {
  value: RouteFilter
  kind: 'all' | 'default' | 'profile' | 'origin' | 'policy'
  name: string
}

/** all, default, profile:x, origin:server|policy|vpn, policy:x (app.js:872). */
export function routeFilterOptions(policies: readonly Policy[], tr: TunnelRoutesResponse | undefined): FilterOption[] {
  const byName = (a: string, b: string) => a.localeCompare(b)
  const profiles = (tr?.tunnels ?? []).filter((t) => t.connected).map((t) => t.name).sort(byName)
  const names = [...new Set(policies.map((p) => p.name).filter(Boolean))].sort(byName)
  return [
    { value: 'all', kind: 'all', name: '' },
    { value: 'default', kind: 'default', name: '' },
    ...profiles.map((n): FilterOption => ({ value: `profile:${n}`, kind: 'profile', name: n })),
    ...(['server', 'policy', 'vpn'] as const).map((o): FilterOption => ({ value: `origin:${o}`, kind: 'origin', name: o })),
    ...names.map((n): FilterOption => ({ value: `policy:${n}`, kind: 'policy', name: n })),
  ]
}

/** Keeps the chosen filter while it still exists; otherwise back to all. */
export function effectiveFilter(filter: RouteFilter, options: readonly FilterOption[]): RouteFilter {
  return options.some((o) => o.value === filter) ? filter : 'all'
}

export function filterRoutes(routes: readonly RouteRow[], filter: RouteFilter, idx: Map<string, RouteInfo>): RouteRow[] {
  if (filter === 'all') return [...routes]
  if (filter === 'default') return routes.filter((r) => r.is_default)
  const [kind, ...rest] = filter.split(':')
  const value = rest.join(':')
  return routes.filter((r) => {
    const info = routeInfo(r, idx)
    if (kind === 'profile') return info?.profile === value
    if (kind === 'origin') return info?.origin === value
    if (kind === 'policy') return r.policy_name === value || info?.policy === value
    return true
  })
}

export interface RouteGroup {
  origin: RouteOrigin
  policy: string
  destinations: string[]
}

/** Consecutive routes of one origin (+ policy) as one group; the daemon already sorts them. */
export function groupTunnelRoutes(routes: readonly TunnelRoute[]): RouteGroup[] {
  const groups: RouteGroup[] = []
  for (const r of routes) {
    const last = groups.at(-1)
    const policy = r.policy ?? ''
    if (last && last.origin === r.origin && last.policy === policy) last.destinations.push(r.destination)
    else groups.push({ origin: r.origin, policy, destinations: [r.destination] })
  }
  return groups
}

/** Element id of a tunnel's card on /traffic (deep link: /traffic#<id>). */
export function tunnelAnchor(name: string): string {
  return `tunnel-${name.replace(/[^\w-]/g, '_')}`
}

// ── Live throughput samples ──────────────────────────────────────────────

export interface TrafficPoint {
  /** ms since epoch */
  at: number
  /** bytes per second since the previous sample */
  rx: number
  tx: number
}

export interface LatencyPoint {
  at: number
  ms: number
}

export interface TrafficSeries {
  points: readonly TrafficPoint[]
  latency: readonly LatencyPoint[]
}

/** 120 samples at the daemon's 5 s heartbeat ≈ the last 10 minutes (as the classic panel). */
export const MAX_SAMPLES = 120

const EMPTY: TrafficSeries = { points: [], latency: [] }

interface Baseline {
  stamp: string
  at: number
  rx: number
  tx: number
  series: TrafficSeries
}

/**
 * Per-tunnel throughput, from the cumulative byte counters of each feed
 * frame. Only tunnels whose frame changed notify their own subscribers, so
 * one live tick re-renders only the charts of tunnels that moved.
 */
export function createTrafficHistory() {
  const state = new Map<string, Baseline>()
  const listeners = new Map<string, Set<() => void>>()
  const notify = (name: string) => listeners.get(name)?.forEach((fn) => fn())

  return {
    record(list: readonly TunnelMetrics[], now: number = Date.now()): void {
      const live = new Set<string>()
      for (const tn of list) {
        if (tn.Status !== 'connected') continue
        live.add(tn.Name)
        const stamp = tn.UpdatedAt || String(now)
        const parsed = Date.parse(tn.UpdatedAt)
        const at = Number.isFinite(parsed) ? parsed : now
        const rx = tn.BytesReceived || 0
        const tx = tn.BytesSent || 0
        const prev = state.get(tn.Name)
        // Same frame again (REST refetch), or an older one arriving late
        // (a slow REST answer after a newer WebSocket frame): keep the order.
        if (prev && (prev.stamp === stamp || at <= prev.at)) continue
        if (!prev) {
          const latency = tn.LatencyMS > 0 ? [{ at, ms: tn.LatencyMS }] : []
          state.set(tn.Name, { stamp, at, rx, tx, series: { points: [], latency } })
          notify(tn.Name)
          continue
        }
        const secs = Math.max((at - prev.at) / 1000, 0.001)
        const point = { at, rx: Math.max(0, rx - prev.rx) / secs, tx: Math.max(0, tx - prev.tx) / secs }
        const lastLatency = prev.series.latency.at(-1)
        // LatencyMS only changes once per health check (≈30 s): record a
        // point only when it moves, so the window shows real variation.
        const latency =
          tn.LatencyMS > 0 && tn.LatencyMS !== lastLatency?.ms
            ? [...prev.series.latency, { at, ms: tn.LatencyMS }].slice(-MAX_SAMPLES)
            : prev.series.latency
        const series = { points: [...prev.series.points, point].slice(-MAX_SAMPLES), latency }
        state.set(tn.Name, { stamp, at, rx, tx, series })
        notify(tn.Name)
      }
      // A tunnel that went down starts over: its counters reset on reconnect.
      for (const name of [...state.keys()]) {
        if (!live.has(name)) {
          state.delete(name)
          notify(name)
        }
      }
    },
    get(name: string): TrafficSeries {
      return state.get(name)?.series ?? EMPTY
    },
    subscribe(name: string, fn: () => void): () => void {
      let set = listeners.get(name)
      if (!set) listeners.set(name, (set = new Set()))
      set.add(fn)
      return () => set.delete(fn)
    },
    reset(): void {
      const names = [...state.keys()]
      state.clear()
      names.forEach(notify)
    },
  }
}

export type TrafficHistory = ReturnType<typeof createTrafficHistory>

/** One history for the app: it keeps filling while the page stays mounted. */
export const trafficHistory = createTrafficHistory()

/** Feeds the history from the tunnels cache entry (WebSocket frames and REST polls alike). */
export function useTrafficRecorder(history: TrafficHistory = trafficHistory): void {
  const client = useQueryClient()
  useEffect(() => startRecording(client, history), [client, history])
}

export function startRecording(client: QueryClient, history: TrafficHistory): () => void {
  const initial = client.getQueryData<TunnelMetrics[]>(keys.tunnels)
  if (initial) history.record(initial)
  return client.getQueryCache().subscribe((event) => {
    if (event.type !== 'updated' || event.action.type !== 'success') return
    const key = event.query.queryKey
    if (key.length !== 1 || key[0] !== keys.tunnels[0]) return
    const data = event.query.state.data as TunnelMetrics[] | undefined
    if (data) history.record(data)
  })
}

export function useTrafficSeries(name: string, history: TrafficHistory = trafficHistory): TrafficSeries {
  return useSyncExternalStore(
    (fn) => history.subscribe(name, fn),
    () => history.get(name),
  )
}

// ── Formatting ───────────────────────────────────────────────────────────

const UNITS = ['B', 'kB', 'MB', 'GB', 'TB']

/** SI units (1 kB = 1000 B), as network rates are usually shown. */
export function formatBytes(n: number, lang: Lang): string {
  if (!Number.isFinite(n) || n <= 0) return `0 B`
  const i = Math.min(UNITS.length - 1, Math.floor(Math.log10(n) / 3))
  const v = n / 1000 ** i
  const digits = i === 0 || v >= 100 ? 0 : 1
  return `${new Intl.NumberFormat(lang, { maximumFractionDigits: digits, minimumFractionDigits: 0 }).format(v)} ${UNITS[i]}`
}

export function formatRate(bytesPerSec: number, lang: Lang): string {
  return `${formatBytes(bytesPerSec, lang)}/s`
}

/** 3725 → "1 h 2 min"; under a minute → "45 s". Same abbreviations in ES and EN. */
export function formatDuration(totalSeconds: number): string {
  const s = Math.max(0, Math.floor(totalSeconds))
  const d = Math.floor(s / 86_400)
  const h = Math.floor((s % 86_400) / 3600)
  const m = Math.floor((s % 3600) / 60)
  if (d > 0) return h > 0 ? `${d} d ${h} h` : `${d} d`
  if (h > 0) return m > 0 ? `${h} h ${m} min` : `${h} h`
  if (m > 0) return `${m} min`
  return `${s} s`
}

/** "en 5 h" / "in 5 hours" relative to now; '' for a missing or bad date. */
export function formatRelative(iso: string, lang: Lang, now: number = Date.now()): string {
  const at = Date.parse(iso)
  if (!iso || !Number.isFinite(at)) return ''
  const secs = Math.round((at - now) / 1000)
  const rtf = new Intl.RelativeTimeFormat(lang, { numeric: 'auto', style: 'short' })
  const abs = Math.abs(secs)
  if (abs < 60) return rtf.format(secs, 'second')
  if (abs < 3600) return rtf.format(Math.round(secs / 60), 'minute')
  if (abs < 86_400) return rtf.format(Math.round(secs / 3600), 'hour')
  return rtf.format(Math.round(secs / 86_400), 'day')
}

export function formatClock(at: number, lang: Lang): string {
  return new Intl.DateTimeFormat(lang, { hour: '2-digit', minute: '2-digit', second: '2-digit' }).format(at)
}
