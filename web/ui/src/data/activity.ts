// Kongtrol UI — Actividad data: audit log + daemon notices · v1.2.0 · 2026-11-19
//
// Two resources, one cache entry each (ADR-0038):
// - GET /api/v1/audit?limit&profile…&level&from&to&before (internal/api/
//   handlers_audit.go, ADR-0042): the signed log, newest first, across every
//   retained day, with `signed` / `verified` per entry. Pages by cursor:
//   `before` = the last timestamp of a page; the X-Kongtrol-Audit-More
//   header says whether an older page exists ("Cargar más"). Dates filter
//   on the daemon (`from` / `to`, local days); `profile` repeats, one per
//   chosen profile (any of them matches). "En vivo" re-reads every 5 s;
//   unchanged rows keep their identity (structural sharing).
// - GET /api/v1/events?after=<seq> (handlers_events.go): the notices the
//   daemon keeps in memory (last 100, cmd/kongtrol/main.go). The first read
//   asks after=0 (the backlog); later reads only what is new. A lower seq
//   means the daemon restarted: the list starts over.
import { keepPreviousData, useInfiniteQuery, useQuery, useQueryClient } from '@tanstack/react-query'
import { http } from '../api/client'
import { messages, type MessageKey } from '../i18n/messages'
import { noticeTextKey } from './home'

// ── Audit log ─────────────────────────────────────────────────────────────

/** internal/api/handlers_audit.go `auditEntryDTO` (security.AuditEvent + signature check). */
export interface AuditEvent {
  id: string
  timestamp: string
  level: string
  action: string
  profile: string
  message: string
  /** Hex HMAC-SHA256; empty when the entry is not signed. */
  hmac: string
  /** The entry carries a signature. */
  signed?: boolean
  /** Its signature checks out against the key in the OS keychain. */
  verified?: boolean
}

export const AUDIT_LEVELS = ['INFO', 'WARN', 'ERROR', 'SECURITY'] as const
export type AuditLevel = (typeof AUDIT_LEVELS)[number]

/** Rows per page ("Cargar más" asks for the next older one); the API allows 1–1000. */
export const AUDIT_PAGE = 200
/** "En vivo": same cadence as the daemon's heartbeat. */
export const LIVE_MS = 5_000
/** The paging header (handlers_audit.go auditMoreHeader). */
export const AUDIT_MORE_HEADER = 'X-Kongtrol-Audit-More'

export interface AuditQuery {
  /** Chosen profiles; none = every profile. */
  profiles: readonly string[]
  level: AuditLevel | ''
  /** Local day YYYY-MM-DD, or '' (no bound). */
  from: string
  to: string
}

export interface AuditPage {
  events: AuditEvent[]
  more: boolean
}

export const activityKeys = {
  audit: (q: AuditQuery) => ['audit', q] as const,
  events: ['events', 'history'] as const,
}

export function auditURL({ profiles, level, from, to }: AuditQuery, before = ''): string {
  const params = new URLSearchParams({ limit: String(AUDIT_PAGE) })
  for (const p of profiles) if (p.trim()) params.append('profile', p.trim())
  if (level) params.set('level', level)
  if (from) params.set('from', from)
  if (to) params.set('to', to)
  if (before) params.set('before', before)
  return `/api/v1/audit?${params.toString()}`
}

/** The cursor for the page after `page`, or undefined when it was the last. */
export function nextCursor(page: AuditPage): string | undefined {
  const last = page.events.at(-1)
  return page.more && last ? last.timestamp : undefined
}

export function useAuditLog(q: AuditQuery, live: boolean, enabled = true) {
  return useInfiniteQuery({
    queryKey: activityKeys.audit(q),
    queryFn: async ({ pageParam }): Promise<AuditPage> => {
      const { data, headers } = await http.getWithHeaders<AuditEvent[] | null>(auditURL(q, pageParam))
      return { events: data ?? [], more: headers.get(AUDIT_MORE_HEADER) === 'true' }
    },
    initialPageParam: '',
    getNextPageParam: nextCursor,
    enabled,
    refetchInterval: live ? LIVE_MS : false,
    // A new filter keeps the last rows on screen until the answer arrives:
    // no skeleton flash, no layout jump.
    placeholderData: keepPreviousData,
  })
}

export type SignatureState = 'verified' | 'tampered' | 'unsigned'

/** Firma válida / no válida / sin firma (a signed entry the daemon could not verify is "no válida"). */
export function signatureState(ev: Pick<AuditEvent, 'hmac' | 'signed' | 'verified'>): SignatureState {
  const signed = ev.signed ?? ev.hmac !== ''
  if (!signed) return 'unsigned'
  return ev.verified ? 'verified' : 'tampered'
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
  /** An extra action button (ADR-0102): src/data/offers.ts. */
  offer?: string
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
      let res = await http.get<EventsResponse>(`/api/v1/events?after=${after}`)
      if (prev && res.seq < prev.seq) {
        // The daemon restarted (its sequence starts again): read it all.
        res = await http.get<EventsResponse>('/api/v1/events?after=0')
        return { seq: res.seq, events: mergeEvents([], res.events ?? []) }
      }
      return { seq: res.seq, events: mergeEvents(prev?.events ?? [], res.events ?? []) }
    },
    refetchInterval: LIVE_MS,
  })
}

/** activity.event.<daemon key>, else Inicio's text for it (kill switch notices, ADR-0102), when this UI knows it. */
function eventKey(key: string): MessageKey | null {
  const k = `activity.event.${key}`
  return k in messages.es ? (k as MessageKey) : (noticeTextKey(key) ?? null)
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
