// Kongtrol UI — resource hooks on the shared cache · v0.1.0 · 2026-11-18
//
// Pages read daemon resources only through these hooks (or new ones next to
// them in phase 2), so every page shares one request, one cache entry and
// one live feed per resource.
import { useEffect, useRef } from 'react'
import { useIsMutating, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '../api/client'
import { describeError } from '../api/errors'
import type { HelperStatus, SecurityStatus, TunnelActionResult, TunnelMetrics } from '../api/types'
import { usePreferences } from '../app/preferences-context'
import { useLiveState } from './connection'
import { keys, mutationKeys } from './keys'
import { useOptimisticMutation } from './optimistic'
import { toasts } from './toast'

/** REST poll while the live feed is down (same cadence as the daemon's heartbeat). */
export const POLL_MS = 5_000
/** While the helper still needs action, notice an install made elsewhere (CLI, classic panel, tray). */
export const HELPER_POLL_MS = 10_000

// ── Tunnels ───────────────────────────────────────────────────────────────

export function useTunnelsQuery() {
  const live = useLiveState()
  return useQuery({
    queryKey: keys.tunnels,
    queryFn: api.tunnels,
    refetchInterval: live === 'live' ? false : POLL_MS,
  })
}

/** The Four States Rule plus "off". No response wins over "connected" (ADR-0030). */
export type TunnelState = 'connected' | 'connecting' | 'no_response' | 'error' | 'disconnected'

export function tunnelState(tn: TunnelMetrics): TunnelState {
  if (tn.health?.state === 'no_response') return 'no_response'
  switch (tn.Status) {
    case 'connected':
    case 'connecting':
    case 'error':
      return tn.Status
    default:
      return 'disconnected'
  }
}

export interface TunnelSummary {
  total: number
  connected: number
  /** Worst state across tunnels, for the header's aggregate status. */
  worst: TunnelState | 'none'
}

const SEVERITY: TunnelState[] = ['error', 'no_response', 'connecting', 'connected', 'disconnected']

export function summarizeTunnels(list: readonly TunnelMetrics[]): TunnelSummary {
  let connected = 0
  let worst: TunnelSummary['worst'] = 'none'
  for (const tn of list) {
    const s = tunnelState(tn)
    if (s === 'connected') connected++
    if (worst === 'none' || SEVERITY.indexOf(s) < SEVERITY.indexOf(worst)) worst = s
  }
  return { total: list.length, connected, worst }
}

type TunnelAction = 'connect' | 'disconnect'

/** Connect / disconnect with the expected state shown at once and rolled back on refusal. */
export function useTunnelAction() {
  const { t } = usePreferences()
  return useOptimisticMutation<{ name: string; action: TunnelAction }, TunnelActionResult, TunnelMetrics[]>({
    queryKey: keys.tunnels,
    mutationFn: ({ name, action }) => (action === 'connect' ? api.connect(name) : api.disconnect(name)),
    apply: (list, { name, action }) =>
      list.map((tn) =>
        tn.Name === name ? { ...tn, Status: action === 'connect' ? 'connecting' : 'disconnected' } : tn,
      ),
    // The daemon accepts at once (202) and the live feed shows the result in
    // place, so success needs no toast — except when the user must act: a
    // provider app with no CLI has to be disconnected by hand.
    successMessage: ({ name }, result) =>
      result?.status === 'waiting_user' ? t('tunnels.toast.manualDisconnect', { name }) : undefined,
  })
}

// ── Groups ────────────────────────────────────────────────────────────────

/** VPN groups (name + profiles). Phase 2 (/profiles/groups) adds the CRUD next to it. */
export function useGroupsQuery() {
  return useQuery({ queryKey: keys.groups, queryFn: api.groups })
}

// ── Security toggles ──────────────────────────────────────────────────────

export function useSecurityQuery() {
  // Local state only (a GET never triggers the public IP lookup).
  return useQuery({ queryKey: keys.security, queryFn: api.security, refetchInterval: POLL_MS })
}

export type SecurityToggle = 'kill_switch' | 'dns_guard'

/** Kill switch / DNS guard switch: flips at once, rolls back if the daemon refuses. */
export function useSecurityToggle(kind: SecurityToggle) {
  return useOptimisticMutation<boolean, unknown, SecurityStatus>({
    queryKey: keys.security,
    mutationFn: (on) => (kind === 'kill_switch' ? api.setKillSwitch(on) : api.setDNSGuard(on)),
    apply: (st, on) =>
      kind === 'kill_switch' ? { ...st, kill_switch_enabled: on } : { ...st, dns_guard_enabled: on },
  })
}

// ── Helper ────────────────────────────────────────────────────────────────

export type HelperUiState = 'unknown' | 'ready' | 'not_needed' | 'missing' | 'outdated'

export function helperUiState(st: HelperStatus | undefined): HelperUiState {
  if (!st) return 'unknown'
  if (st.elevated) return 'not_needed'
  if (!st.needs_install) return 'ready'
  // `outdated` comes from newer daemons; older ones: installed and running
  // but not speaking our protocol.
  const outdated = st.outdated ?? (st.installed && st.running && !st.available)
  return outdated ? 'outdated' : 'missing'
}

export function useHelperStatus() {
  return useQuery({
    queryKey: keys.helper,
    queryFn: api.helperStatus,
    refetchInterval: (q) => (q.state.data?.needs_install ? HELPER_POLL_MS : false),
  })
}

/** True while an install runs, whichever button started it. */
export function useHelperInstalling(): boolean {
  return useIsMutating({ mutationKey: mutationKeys.helperInstall }) > 0
}

/**
 * Installs (or updates) the helper. On success the cache gets the new
 * status at once, so every install button disappears without waiting for a
 * poll; the "ready" toast comes from useHelperReadyToast.
 */
export function useHelperInstall() {
  const client = useQueryClient()
  const { t } = usePreferences()
  return useMutation({
    mutationKey: mutationKeys.helperInstall,
    mutationFn: api.installHelper,
    onSuccess: (status) => {
      client.setQueryData(keys.helper, status)
      void client.invalidateQueries({ queryKey: keys.tunnels })
    },
    onError: (err) => {
      const { message, fix } = describeError(err, t)
      toasts.show({ tone: 'bad', message, detail: fix })
      void client.invalidateQueries({ queryKey: keys.helper })
    },
  })
}

/**
 * One toast when the helper goes from missing/outdated to ready — from this
 * page, the classic panel, the tray or `kongtrol service install`.
 */
export function useHelperReadyToast(): void {
  const { data } = useHelperStatus()
  const { t } = usePreferences()
  const previous = useRef<HelperUiState>('unknown')
  const state = helperUiState(data)
  useEffect(() => {
    const before = previous.current
    if ((before === 'missing' || before === 'outdated') && state === 'ready') {
      toasts.show({ tone: 'ok', message: t('helper.toast.ready') })
    }
    if (state !== 'unknown') previous.current = state
  }, [state, t])
}

// ── Version ───────────────────────────────────────────────────────────────

/** This daemon's version: GET /api/v1/version, else the helper status' cli_version (older daemons). */
export function useAppVersion(): string {
  const version = useQuery({ queryKey: keys.version, queryFn: api.version, staleTime: Infinity, retry: false })
  const helper = useHelperStatus()
  return version.data?.version || helper.data?.cli_version || ''
}
