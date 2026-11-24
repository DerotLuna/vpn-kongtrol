// Reglas — "Rutas que trae cada VPN": pure model · v1.0.0 · 2026-11-24
//
// GET /api/v1/vpns/routes (internal/app/vpnroutes.go, ADR-0107): the routes
// each VPN brings by itself, read-only. This module folds a full tunnel
// into one "Todo el tráfico" row, builds the one-line summary per profile
// and the conflict sentence; the daemon decides who wins (longest prefix),
// the panel only words it. Tested in vpnRoutesModel.test.ts.
import type { MessageKey } from '../../i18n/messages'

type T = (key: MessageKey, vars?: Record<string, string | number>) => string

export type OwnRouteOrigin = 'file' | 'server'
export type ConflictWinner = 'rule' | 'vpn' | 'system'

/** internal/app/vpnroutes.go RouteConflict. */
export interface RouteConflict {
  rule: string
  via: string
  destination: string
  winner: ConflictWinner
}

/** internal/app/vpnroutes.go OwnRoute. */
export interface OwnRoute {
  destination: string
  origin: OwnRouteOrigin
  conflicts?: RouteConflict[] | null
}

/** internal/app/vpnroutes.go ProfileOwnRoutes. */
export interface ProfileOwnRoutes {
  name: string
  type: string
  connected: boolean
  source: 'config' | 'app' | 'unreadable'
  all_traffic: boolean
  rules_only: boolean
  server_pending: boolean
  routes: OwnRoute[] | null
}

export interface VpnRoutesResponse {
  profiles: ProfileOwnRoutes[] | null
}

/** One line of the list: a prefix, or the whole IPv4 / IPv6 default route in words. */
export interface RouteRow {
  kind: 'all' | 'all6' | 'route'
  destination: string
  origin: OwnRouteOrigin
  conflicts: RouteConflict[]
}

const FULL_V4 = new Set(['0.0.0.0/0', '0.0.0.0/1', '128.0.0.0/1'])

/**
 * The rows to show: 0.0.0.0/0, or both halves 0.0.0.0/1 + 128.0.0.0/1,
 * become one "all" row (first, with the origin of the first piece); ::/0
 * becomes "all6". A lone /1 half stays a prefix.
 */
export function routeRows(p: ProfileOwnRoutes): RouteRow[] {
  const routes = p.routes ?? []
  const has = (d: string) => routes.some((r) => r.destination === d)
  const fullV4 = has('0.0.0.0/0') || (has('0.0.0.0/1') && has('128.0.0.0/1'))
  const rows: RouteRow[] = []
  let allRow: RouteRow | null = null
  for (const r of routes) {
    if (fullV4 && FULL_V4.has(r.destination)) {
      allRow ??= { kind: 'all', destination: '0.0.0.0/0', origin: r.origin, conflicts: [] }
      continue
    }
    const kind = r.destination === '::/0' ? 'all6' : 'route'
    rows.push({ kind, destination: r.destination, origin: r.origin, conflicts: r.conflicts ?? [] })
  }
  return allRow ? [allRow, ...rows] : rows
}

/** Where the profile's routes come from, for the summary line. */
export function originSummary(rows: readonly RouteRow[]): 'file' | 'server' | 'mixed' | 'none' {
  const file = rows.some((r) => r.origin === 'file')
  const server = rows.some((r) => r.origin === 'server')
  if (file && server) return 'mixed'
  if (file) return 'file'
  if (server) return 'server'
  return 'none'
}

/** "ekiipago · 3 rutas · del servidor" (without the name, which is shown bold). */
export function profileSummary(p: ProfileOwnRoutes, t: T): string {
  if (p.source === 'app') return t('rules.vpnRoutes.summary.app')
  const rows = routeRows(p)
  if (rows.length === 0) return t('rules.vpnRoutes.summary.none')
  const parts = [t(rows.length === 1 ? 'rules.vpnRoutes.summary.one' : 'rules.vpnRoutes.summary.other', { count: rows.length })]
  const origin = originSummary(rows)
  if (origin !== 'none') parts.push(t(`rules.vpnRoutes.summary.${origin}` as MessageKey))
  if (p.rules_only) parts.push(t('rules.vpnRoutes.summary.unused'))
  return parts.join(' · ')
}

/** The prefix in words: "Todo el tráfico" for a full tunnel, else the CIDR. */
export function rowLabel(row: RouteRow, t: T): string {
  if (row.kind === 'all') return t('rules.vpnRoutes.allTraffic')
  if (row.kind === 'all6') return t('rules.vpnRoutes.allTrafficV6')
  return row.destination
}

/** What the "＋ Agregar" rule sheet gets as target: a host without /32 or /128. */
export function ruleTarget(destination: string): string {
  return destination.replace(/\/(32|128)$/, '')
}

/** Who wins between one of the user's rules and this VPN route, in one sentence. */
export function conflictText(c: RouteConflict, route: string, t: T): string {
  switch (c.winner) {
    case 'rule':
      return t('rules.vpnRoutes.conflict.rule', { rule: c.rule, destination: ruleTarget(c.destination), via: c.via })
    case 'vpn':
      return t('rules.vpnRoutes.conflict.vpn', { route, rule: c.rule })
    default:
      return t('rules.vpnRoutes.conflict.system', { rule: c.rule, destination: c.destination, via: c.via })
  }
}
