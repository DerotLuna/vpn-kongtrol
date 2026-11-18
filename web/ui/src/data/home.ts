// Kongtrol UI — Inicio (/) data: resources, actions and live samples · v1.0.1 · 2026-11-18
//
// Everything the Home page reads or changes that the shared layer
// (resources.ts) does not have yet: network overview, what goes through each
// tunnel, daemon notices (events), per-tunnel actions
// (connect, cancel, disconnect, reload, all), group quick launch and the
// latency samples behind the sparklines. Keys name resources, not pages, so
// /traffic can read the same cache entries.
import { useCallback, useEffect, useMemo, useRef } from 'react'
import { useMutation, useQuery, useQueryClient, type QueryClient } from '@tanstack/react-query'
import { ApiError, api, http } from '../api/client'
import { describeError } from '../api/errors'
import type { SecurityStatus, TunnelMetrics } from '../api/types'
import { usePreferences } from '../app/preferences-context'
import type { MessageKey } from '../i18n/messages'
import { keys } from './keys'
import { useOptimisticMutation } from './optimistic'
import { createStore, useStore } from './store'
import { toasts } from './toast'

// ── Types (mirrors of the Go DTOs; source in each comment) ────────────────

/** internal/api/handlers_routes.go handleNetworkOverview `overviewDTO`. */
export interface NetworkOverview {
  connected_tunnels: number
  default_routes: { destination: string; gateway: string; interface: string; metric: number }[] | null
  local_ips: string[] | null
  public_ip?: string
  public_ip_check_enabled: boolean
}

/** internal/api/handlers_tunnel_routes.go `tunnelRouteDTO`. */
export interface TunnelRoute {
  destination: string
  origin: 'server' | 'policy' | 'vpn'
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
  counts: Record<string, number> | null
  routes: TunnelRoute[] | null
}

/** internal/api/handlers_tunnel_routes.go `tunnelRoutesResponse`. */
export interface TunnelRoutesResponse {
  rest_via?: string
  rest_via_tunnel?: string
  tunnels: TunnelRoutes[] | null
}

/** internal/monitor/events.go `Event`. */
export interface DaemonEvent {
  seq: number
  time: string
  kind: string
  level: 'INFO' | 'WARN'
  key: string
  args?: string[]
  profile?: string
  policy?: string
  action?: string
  can_connect?: boolean
}

/** internal/api/handlers_events.go `EventsResponse`. */
export interface EventsResponse {
  events: DaemonEvent[] | null
  seq: number
}

/** POST /api/v1/tunnels/{name}/reload (handlers_tunnels.go handleReloadTunnel). */
export interface ReloadResult {
  status: 'restarting' | 'skipped_not_connected' | 'restart_required'
  tunnel: string
}

/** POST /api/v1/groups/{name}/connect (handlers_groups.go handleConnectGroup). */
export interface GroupConnectResult {
  status: 'connecting'
  group: string
  started: string[] | null
  already_connected: string[] | null
}

/** internal/security/killswitch_status.go `KillSwitchStatus` (the fields shown here). */
export interface KillSwitchDetail {
  state: 'off' | 'armed' | 'active' | string
  protected?: string[]
  blocking?: string[]
  policy_blocking?: string[]
  blocked_count?: number
}

/** The extra GET /api/v1/security/status fields the Home summary reads (handlers_security.go secStatus). */
export interface SecuritySummary extends SecurityStatus {
  leak_detection_configured?: boolean
  leak_check_available?: boolean
  kill_switch_status?: KillSwitchDetail
}

// ── Endpoints ─────────────────────────────────────────────────────────────

const enc = encodeURIComponent

export const homeApi = {
  networkOverview: () => http.get<NetworkOverview>('/api/v1/network/overview'),
  tunnelRoutes: () => http.get<TunnelRoutesResponse>('/api/v1/tunnels/routes'),
  events: (after: number) => http.get<EventsResponse>(`/api/v1/events?after=${after}`),
  reload: (name: string) => http.post<ReloadResult>(`/api/v1/tunnels/${enc(name)}/reload`),
  connectGroup: (name: string) =>
    http.post<GroupConnectResult>(`/api/v1/groups/${enc(name)}/connect`),
  disconnectGroup: (name: string) => http.post<unknown>(`/api/v1/groups/${enc(name)}/disconnect`),
}

export const homeKeys = {
  networkOverview: ['network', 'overview'] as const,
  tunnelRoutes: ['tunnel-routes'] as const,
} as const

export const homeMutationKeys = {
  groupAction: ['groups', 'action'] as const,
} as const

/** Legacy cadence (app.js): everything that is not the live feed refreshes every 30 s. */
export const SLOW_POLL_MS = 30_000
/** Legacy notices.js polls the events every 5 s. */
export const EVENTS_POLL_MS = 5_000
/** After a tunnel action, routes and egress settle a moment later (legacy: 600 ms). */
export const SETTLE_MS = 600

// ── Reads ─────────────────────────────────────────────────────────────────

export function useNetworkOverview() {
  return useQuery({ queryKey: homeKeys.networkOverview, queryFn: homeApi.networkOverview, refetchInterval: SLOW_POLL_MS })
}

export function useTunnelRoutes() {
  return useQuery({ queryKey: homeKeys.tunnelRoutes, queryFn: homeApi.tunnelRoutes, refetchInterval: SLOW_POLL_MS })
}

/** One tunnel's routes, selected from the shared answer (one request for every row). */
export function useTunnelRoutesFor(name: string): TunnelRoutes | undefined {
  const { data } = useTunnelRoutes()
  return useMemo(() => data?.tunnels?.find((tr) => tr.name === name), [data, name])
}

export function useGroupsList() {
  return useQuery({ queryKey: keys.groups, queryFn: api.groups, refetchInterval: SLOW_POLL_MS })
}

/**
 * "Revisar IP pública": one opt-in lookup now. Updates the security status
 * and the egress card (both show the IP); a failure is shown in place by
 * the caller (mutation.error) and as a toast.
 */
export function usePublicIPCheck() {
  const client = useQueryClient()
  const { t } = usePreferences()
  return useMutation({
    mutationFn: api.checkPublicIP,
    onError: (err) => {
      const { message, fix } = describeError(err, t)
      toasts.show({ tone: 'bad', message, detail: fix })
    },
    onSettled: () => {
      void client.invalidateQueries({ queryKey: keys.security })
      void client.invalidateQueries({ queryKey: homeKeys.networkOverview })
    },
  })
}

/** Routes and egress change a moment after a tunnel goes up or down. */
export function settleNetwork(client: QueryClient): void {
  setTimeout(() => {
    void client.invalidateQueries({ queryKey: homeKeys.tunnelRoutes })
    void client.invalidateQueries({ queryKey: homeKeys.networkOverview })
  }, SETTLE_MS)
}

// ── Per-tunnel actions ────────────────────────────────────────────────────

export type TunnelAction = 'connect' | 'cancel' | 'disconnect' | 'reload'

/** Which action runs on which tunnel right now (rows show it in place). */
export const pendingActions = createStore<Readonly<Record<string, TunnelAction>>>({})

function setPending(name: string, action: TunnelAction | null) {
  const next = { ...pendingActions.get() }
  if (action) next[name] = action
  else delete next[name]
  pendingActions.set(next)
}

export function usePendingAction(name: string): TunnelAction | undefined {
  return useStore(pendingActions)[name]
}

type Translate = ReturnType<typeof usePreferences>['t']

/**
 * Our own text for the daemon's plain-English refusals (writeError): the
 * status says enough to name the problem and the fix in the user's language.
 */
function guided(err: unknown, t: Translate, name: string, action: TunnelAction): unknown {
  if (!(err instanceof ApiError) || err.kind !== 'server' || err.hint) return err
  const msg = (key: MessageKey, fix: MessageKey) =>
    new ApiError('server', err.status, { error: t(key, { name }), hint: t(fix, { name }), error_key: err.key })
  if (err.status === 404) return msg('home.err.notFound', 'home.err.notFound.fix')
  if (err.status === 503) return msg('home.err.unavailable', 'home.err.unavailable.fix')
  if (action === 'reload' && err.status === 400) return msg('home.err.reloadConfig', 'home.err.reloadConfig.fix')
  if (err.status >= 500) {
    const key: MessageKey = action === 'reload' ? 'home.err.reload' : action === 'connect' ? 'home.err.connect' : 'home.err.disconnect'
    return new ApiError('server', err.status, { error: t(key, { name }), hint: t('home.err.retry.fix'), error_key: err.key })
  }
  return err
}

function withPending<T>(name: string, action: TunnelAction, run: () => Promise<T>): Promise<T> {
  setPending(name, action)
  return run().finally(() => setPending(name, null))
}

function patch(list: TunnelMetrics[], name: string, change: Partial<TunnelMetrics>): TunnelMetrics[] {
  return list.map((tn) => (tn.Name === name ? { ...tn, ...change } : tn))
}

/**
 * Connect, cancel and disconnect with the expected state at once and a
 * rollback (+ toast with what to do) if the daemon refuses.
 * - connect: the row turns "Conectando" at once; 409 (already connecting)
 *   is not an error.
 * - cancel: back to "Desconectada".
 * - disconnect: the row keeps its state with "Desconectando…" on the button
 *   until the daemon answers, because a provider app without a CLI answers
 *   202 waiting_user and stays connected until the user acts in the app
 *   (no flicker connected → off → connected).
 */
export function useTunnelActions() {
  const client = useQueryClient()
  const { t } = usePreferences()

  const mutation = useOptimisticMutation<{ name: string; action: Exclude<TunnelAction, 'reload'> }, string, TunnelMetrics[]>({
    queryKey: keys.tunnels,
    mutationFn: ({ name, action }) =>
      withPending(name, action, async () => {
        try {
          if (action === 'connect') return (await api.connect(name)).status
          if (action === 'cancel') {
            await api.cancelConnect(name)
            return 'connect_cancelled'
          }
          return (await api.disconnect(name)).status
        } catch (err) {
          if (action === 'connect' && err instanceof ApiError && err.status === 409) return 'connecting'
          throw guided(err, t, name, action)
        }
      }),
    apply: (list, { name, action }) => {
      if (action === 'connect') return patch(list, name, { Status: 'connecting', last_error: '', last_error_key: '' })
      if (action === 'cancel') return patch(list, name, { Status: 'disconnected' })
      return list
    },
    successMessage: ({ name, action }, status) => {
      if (action === 'disconnect' && status === 'waiting_user') return t('tunnels.toast.manualDisconnect', { name })
      return undefined
    },
  })

  const { mutate } = mutation
  const run = useCallback(
    (name: string, action: Exclude<TunnelAction, 'reload'>) =>
      mutate({ name, action }, { onSettled: () => settleNetwork(client) }),
    [mutate, client],
  )
  return run
}

/**
 * Reload: re-reads kongtrol.yaml and restarts the tunnel in place.
 * 202 restarting → "Recargando…"; skipped_not_connected → the config was
 * re-read, nothing to restart (legacy said "Recargando" here, a bug);
 * 409 restart_required → Kongtrol itself must restart.
 */
export function useTunnelReload() {
  const client = useQueryClient()
  const { t } = usePreferences()
  const { mutate } = useMutation({
    mutationFn: (name: string) =>
      withPending(name, 'reload', async () => {
        try {
          return await homeApi.reload(name)
        } catch (err) {
          if (err instanceof ApiError && err.status === 409) {
            throw new ApiError('server', 409, {
              error: t('home.toast.restartRequired', { name }),
              hint: t('home.toast.restartRequired.fix'),
              error_key: 'restart_required',
            })
          }
          throw guided(err, t, name, 'reload')
        }
      }),
    onSuccess: (res, name) => {
      toasts.show({
        tone: 'info',
        message: res.status === 'skipped_not_connected' ? t('home.toast.reloadedIdle', { name }) : t('home.toast.reloading', { name }),
      })
    },
    onError: (err, name) => {
      const { message, fix } = describeError(err, t)
      toasts.show({ tone: 'bad', message, detail: fix, action: { label: t('error.retry'), run: () => mutate(name) } })
    },
    onSettled: () => {
      void client.invalidateQueries({ queryKey: keys.tunnels })
      settleNetwork(client)
    },
  })
  return mutate
}

/** "Conectar todas" / "Desconectar todas": one action per tunnel, in parallel. */
export function bulkTargets(list: readonly TunnelMetrics[], action: 'connect' | 'disconnect'): string[] {
  return list
    .filter((tn) => (action === 'connect' ? tn.Status === 'disconnected' || tn.Status === 'error' : tn.Status === 'connected'))
    .map((tn) => tn.Name)
}

// ── Groups quick launch ───────────────────────────────────────────────────

export type GroupAction = 'connect' | 'disconnect'

export function useGroupAction() {
  const client = useQueryClient()
  const { t } = usePreferences()
  const mutation = useMutation({
    mutationKey: homeMutationKeys.groupAction,
    mutationFn: async ({ name, action }: { name: string; action: GroupAction; profiles: readonly string[] }) => {
      try {
        if (action === 'connect') return await homeApi.connectGroup(name)
        await homeApi.disconnectGroup(name)
        return null
      } catch (err) {
        if (err instanceof ApiError && err.kind === 'server' && !err.hint) {
          const key: MessageKey = err.status === 404 ? 'home.groups.err.notFound' : 'home.groups.err.failed'
          throw new ApiError('server', err.status, {
            error: t(key, { name }),
            hint: t(err.status === 404 ? 'home.groups.err.notFound.fix' : 'home.groups.err.failed.fix'),
            error_key: err.key,
            detail: err.message,
          })
        }
        throw err
      }
    },
    onSuccess: (res, { name, action, profiles }) => {
      if (action === 'disconnect') {
        toasts.show({ tone: 'info', message: t('home.groups.toast.disconnecting', { name }) })
        return
      }
      const started = res?.started ?? []
      const already = res?.already_connected ?? []
      const skipped = profiles.filter((p) => !started.includes(p) && !already.includes(p))
      const message =
        started.length > 0
          ? t('home.groups.toast.connecting', { name, n: started.length })
          : t('home.groups.toast.allConnected', { name })
      toasts.show({
        tone: 'info',
        message,
        ...(skipped.length > 0 ? { detail: t('home.groups.toast.skipped', { names: skipped.join(', ') }) } : {}),
      })
    },
    onError: (err) => {
      const { message, fix } = describeError(err, t)
      toasts.show({ tone: 'bad', message, detail: fix })
    },
    onSettled: () => {
      void client.invalidateQueries({ queryKey: keys.tunnels })
      settleNetwork(client)
    },
  })
  return mutation
}

// ── Latency samples (sparkline) ───────────────────────────────────────────

/** Samples kept per tunnel for the trend line. */
export const LATENCY_SAMPLES = 30

export const latencyHistory = createStore<Readonly<Record<string, readonly number[]>>>({})

/**
 * Records one latency sample per tunnel when the value changes (legacy
 * app.js: a repeated heartbeat value is not a new measurement). Pure, so it
 * is easy to test; returns the same object when nothing changed.
 */
export function recordLatency(
  history: Readonly<Record<string, readonly number[]>>,
  list: readonly TunnelMetrics[],
): Readonly<Record<string, readonly number[]>> {
  let next: Record<string, readonly number[]> | null = null
  for (const tn of list) {
    const prev = history[tn.Name] ?? []
    const connected = tn.Status === 'connected'
    if (!connected) {
      if (prev.length > 0) (next ??= { ...history })[tn.Name] = []
      continue
    }
    if (tn.LatencyMS <= 0 || prev.at(-1) === tn.LatencyMS) continue
    ;(next ??= { ...history })[tn.Name] = [...prev, tn.LatencyMS].slice(-LATENCY_SAMPLES)
  }
  return next ?? history
}

/**
 * Feeds the latency store from every tunnels update (REST or WebSocket).
 * Listens to the cache itself, not to renders, so two frames that arrive
 * before React renders still count as two samples.
 */
export function useLatencyRecorder(): void {
  const client = useQueryClient()
  useEffect(() => {
    const record = (list: unknown) => {
      if (Array.isArray(list)) latencyHistory.set(recordLatency(latencyHistory.get(), list as TunnelMetrics[]))
    }
    record(client.getQueryData(keys.tunnels))
    return client.getQueryCache().subscribe((event) => {
      if (event.type === 'updated' && event.action.type === 'success' && event.query.queryKey[0] === keys.tunnels[0] && event.query.queryKey.length === 1) {
        record(event.action.data)
      }
    })
  }, [client])
}

export function useLatencySamples(name: string): readonly number[] {
  return useStore(latencyHistory)[name] ?? NO_SAMPLES
}
const NO_SAMPLES: readonly number[] = []

// ── Daemon notices (GET /api/v1/events) ───────────────────────────────────

/** Kinds shown as notices (legacy notices.js:122). */
export const NOTICE_KINDS = new Set(['policy_tunnel_down', 'tunnel_watchdog', 'update_available'])
/** Recent notices kept on the page. */
export const NOTICES_MAX = 6
const SEQ_KEY = 'kongtrol.noticeSeq'

export const recentNotices = createStore<readonly DaemonEvent[]>([])

/** The last event seen survives page changes in this tab; a fresh tab starts at -1 (no backlog). */
export function readNoticeSeq(): number {
  try {
    const v = sessionStorage.getItem(SEQ_KEY)
    return v === null ? -1 : Number(v)
  } catch {
    return -1
  }
}

function writeNoticeSeq(seq: number) {
  try {
    sessionStorage.setItem(SEQ_KEY, String(seq))
  } catch {
    /* private mode: a notice may repeat after a page change */
  }
}

/**
 * Polls the daemon's notices every 5 s: new ones go to the recent list and
 * to one toast each; a policy whose tunnel is off offers "Conectar <perfil>".
 * Rendering (text, action) is passed in so this stays free of UI.
 */
export function useDaemonNotices(onNotice: (e: DaemonEvent) => void): void {
  const { data } = useQuery({
    queryKey: ['events', 'poll'],
    queryFn: async () => {
      const res = await homeApi.events(readNoticeSeq())
      if (typeof res.seq === 'number') writeNoticeSeq(res.seq)
      return res
    },
    refetchInterval: EVENTS_POLL_MS,
    staleTime: 0,
    gcTime: 0,
  })
  // onNotice changes with the language; each answer is handled once.
  const handler = useRef(onNotice)
  useEffect(() => {
    handler.current = onNotice
  }, [onNotice])
  useEffect(() => {
    const fresh = (data?.events ?? []).filter((e) => NOTICE_KINDS.has(e.kind))
    if (fresh.length === 0) return
    recentNotices.set([...fresh].reverse().concat(recentNotices.get()).slice(0, NOTICES_MAX))
    fresh.forEach((e) => handler.current(e))
  }, [data])
}

export function useRecentNotices(): readonly DaemonEvent[] {
  return useStore(recentNotices)
}

/** Daemon notice keys this page translates (internal/i18n mirrors). */
const NOTICE_TEXT: Record<string, MessageKey> = {
  'policy.event.allow': 'home.notice.policy.allow',
  'policy.event.block': 'home.notice.policy.block',
  'policy.event.connect': 'home.notice.policy.connect',
  'policy.event.connect_failed': 'home.notice.policy.connectFailed',
  'watchdog.event.reconnect_failing': 'home.notice.watchdog.reconnectFailing',
  'watchdog.event.failover': 'home.notice.watchdog.failover',
  'watchdog.event.reconnected': 'home.notice.watchdog.reconnected',
  'watchdog.event.no_response': 'home.notice.watchdog.noResponse',
  'watchdog.event.responding': 'home.notice.watchdog.responding',
  'watchdog.event.manual_reconnect': 'home.notice.watchdog.manualReconnect',
  'update.event.available': 'home.notice.update.available',
}

/** One notice as text in the page's language; unknown keys fall back to a generic line. */
export function noticeText(e: DaemonEvent, t: Translate): string {
  const key = NOTICE_TEXT[e.key]
  const args = e.args ?? []
  if (!key) return t('home.notice.generic', { profile: e.profile ?? args[0] ?? '' })
  return t(key, { 0: args[0] ?? '', 1: args[1] ?? '', 2: args[2] ?? '' })
}

/** A notice with an action waits this long (legacy notices.js: 60 s). */
export const ACTION_TOAST_MS = 60_000

/**
 * Polls the daemon's notices and toasts each new one. Mounted once, by the
 * shell (AppShell), so every page gets them and Inicio's list shares the poll.
 */
export function useNoticeToasts(): void {
  const { t } = usePreferences()
  const run = useTunnelActions()
  const onNotice = useCallback(
    (e: DaemonEvent) => {
      const message = noticeText(e, t)
      const tone = e.level === 'WARN' ? 'bad' : 'info'
      if (e.can_connect && e.profile) {
        const profile = e.profile
        toasts.show({
          tone,
          message,
          duration: ACTION_TOAST_MS,
          action: { label: t('home.notices.connect', { profile }), run: () => run(profile, 'connect') },
        })
        return
      }
      toasts.show({ tone, message })
    },
    [t, run],
  )
  useDaemonNotices(onNotice)
}
