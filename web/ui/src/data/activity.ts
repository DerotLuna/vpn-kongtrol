// Kongtrol UI — Actividad data: audit log + daemon notices · v1.0.0 · 2026-11-18
//
// Two resources, one cache entry each (ADR-0038):
// - GET /api/v1/audit?limit&profile&level (internal/api/handlers_security.go
//   handleAuditLog): the signed log, newest first. The daemon reads the last
//   two daily files only. "En vivo" re-reads it every 5 s; earlier rows keep
//   their identity (structural sharing), so only new rows render.
// - GET /api/v1/events?after=<seq> (handlers_events.go): the notices the
//   daemon keeps in memory (last 100, cmd/kongtrol/main.go). The first read
//   asks after=0 (the backlog); later reads only what is new. A lower seq
//   means the daemon restarted: the list starts over.
import { keepPreviousData, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiCall } from '../api/client'
import { messages, type MessageKey } from '../i18n/messages'

// ── Audit log ─────────────────────────────────────────────────────────────

/** internal/security/audit.go `AuditEvent`. */
export interface AuditEvent {
  id: string
  timestamp: string
  level: string
  action: string
  profile: string
  message: string
  /** Hex HMAC-SHA256; empty when the log is not signed. */
  hmac: string
  /**
   * Result of checking the HMAC with the key. The daemon does not send it
   * yet (security.VerifyEvent has no API caller): until it does, a signed
   * entry reads "firmada · sin comprobar", never "válida".
   */
  verified?: boolean
}

export const AUDIT_LEVELS = ['INFO', 'WARN', 'ERROR', 'SECURITY'] as const
export type AuditLevel = (typeof AUDIT_LEVELS)[number]
export type AuditRange = 'all' | 'hour' | 'today' | 'yesterday'
export const AUDIT_RANGES: readonly AuditRange[] = ['all', 'hour', 'today', 'yesterday']

/** Rows per read, as the classic panel; "Mostrar más" asks for the next page. */
export const AUDIT_PAGE = 300
/** "En vivo": same cadence as the daemon's heartbeat. */
export const LIVE_MS = 5_000

export interface AuditQuery {
  profile: string
  level: AuditLevel | ''
  limit: number
}

export const activityKeys = {
  audit: (q: AuditQuery) => ['audit', q] as const,
  events: ['events', 'history'] as const,
}

export function auditURL({ profile, level, limit }: AuditQuery): string {
  const params = new URLSearchParams({ limit: String(limit) })
  if (profile.trim()) params.set('profile', profile.trim())
  if (level) params.set('level', level)
  return `/api/v1/audit?${params.toString()}`
}

export function useAuditLog(q: AuditQuery, live: boolean) {
  return useQuery({
    queryKey: activityKeys.audit(q),
    queryFn: () => apiCall.get<AuditEvent[] | null>(auditURL(q)).then((l) => l ?? []),
    refetchInterval: live ? LIVE_MS : false,
    // A new filter keeps the last rows on screen until the answer arrives:
    // no skeleton flash, no layout jump.
    placeholderData: keepPreviousData,
  })
}

function startOfDay(d: Date): number {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime()
}

/** Date filter, applied in the page (the API has no date parameter). */
export function inRange(ev: Pick<AuditEvent, 'timestamp'>, range: AuditRange, now: Date = new Date()): boolean {
  if (range === 'all') return true
  const at = new Date(ev.timestamp).getTime()
  if (Number.isNaN(at)) return false
  const today = startOfDay(now)
  switch (range) {
    case 'hour':
      return at >= now.getTime() - 3_600_000
    case 'today':
      return at >= today
    case 'yesterday':
      return at >= today - 86_400_000 && at < today
  }
}

export type SignatureState = 'verified' | 'tampered' | 'signed' | 'unsigned'

export function signatureState(ev: Pick<AuditEvent, 'hmac' | 'verified'>): SignatureState {
  if (!ev.hmac) return 'unsigned'
  if (ev.verified === true) return 'verified'
  if (ev.verified === false) return 'tampered'
  return 'signed'
}

/** One JSON object per line, the format of the log file itself (and `kongtrol logs --json`). */
export function auditToJSONL(events: readonly AuditEvent[]): string {
  return events.map((ev) => JSON.stringify(ev)).join('\n') + (events.length ? '\n' : '')
}

/** Saves text as a file in the browser; nothing leaves the machine. */
export function downloadText(filename: string, text: string, type = 'application/x-ndjson'): void {
  const url = URL.createObjectURL(new Blob([text], { type }))
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.append(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}

// ── Daemon notices ────────────────────────────────────────────────────────

/** internal/monitor/events.go `Event`. */
export interface DaemonEvent {
  seq: number
  time: string
  kind: string
  level: 'INFO' | 'WARN' | string
  key: string
  args?: string[]
  profile?: string
  policy?: string
  action?: string
  can_connect?: boolean
}

interface EventsResponse {
  events: DaemonEvent[]
  seq: number
}

export interface EventsHistory {
  seq: number
  events: DaemonEvent[]
}

/** The daemon keeps 100; a page open for days keeps a little more. */
export const EVENTS_MAX = 200

/** New first, no duplicates, capped. Unchanged events keep their object. */
export function mergeEvents(prev: readonly DaemonEvent[], next: readonly DaemonEvent[], max = EVENTS_MAX): DaemonEvent[] {
  if (next.length === 0) return prev as DaemonEvent[]
  const seen = new Set(prev.map((e) => e.seq))
  const fresh = next.filter((e) => !seen.has(e.seq))
  return [...fresh, ...prev].sort((a, b) => b.seq - a.seq).slice(0, max)
}

export function useEventsHistory() {
  const client = useQueryClient()
  return useQuery({
    queryKey: activityKeys.events,
    queryFn: async (): Promise<EventsHistory> => {
      const prev = client.getQueryData<EventsHistory>(activityKeys.events)
      const after = prev?.seq ?? 0
      let res = await apiCall.get<EventsResponse>(`/api/v1/events?after=${after}`)
      if (prev && res.seq < prev.seq) {
        // The daemon restarted (its sequence starts again): read it all.
        res = await apiCall.get<EventsResponse>('/api/v1/events?after=0')
        return { seq: res.seq, events: mergeEvents([], res.events ?? []) }
      }
      return { seq: res.seq, events: mergeEvents(prev?.events ?? [], res.events ?? []) }
    },
    refetchInterval: LIVE_MS,
  })
}

/** activity.event.<daemon key>, when this UI knows it. */
function eventKey(key: string): MessageKey | null {
  const k = `activity.event.${key}`
  return k in messages.es ? (k as MessageKey) : null
}

type T = (key: MessageKey, vars?: Record<string, string | number>) => string

/** The notice in the reader's language; {0}, {1}… are the daemon's args. */
export function eventText(ev: Pick<DaemonEvent, 'key' | 'args'>, t: T): string {
  const vars: Record<string, string> = {}
  ;(ev.args ?? []).forEach((a, i) => {
    vars[String(i)] = a
  })
  const k = eventKey(ev.key)
  return k ? t(k, vars) : t('activity.event.unknown', { key: ev.key })
}
