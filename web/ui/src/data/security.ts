// Kongtrol UI — Seguridad (/security) data: queries + mutations · v1.1.0 · 2026-11-18
//
// Everything the Security page reads or changes, on the shared cache
// (ADR-0038). Cache keys name resources, so another page reading the same
// resource (VPN profiles, policies, settings) shares this entry: keep the
// key values identical to theirs.
//
// Toggles are optimistic (the switch moves at once and rolls back with a
// "what to do" toast if the daemon refuses). Turning a protection OFF and
// turning the external public IP lookup ON are confirmed by the page first
// (components/security), never here.
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { http } from '../api/client'
import type { LeakCheck, SecurityStatus } from '../api/types'
import { usePreferences } from '../app/preferences-context'
import { keys } from './keys'
import { useOptimisticMutation } from './optimistic'

// ── Types (mirrors of the Go JSON; source named above each) ──────────────

/** internal/security/killswitch_status.go `KillSwitchStatus`. */
export interface KillSwitchDetail {
  state: 'off' | 'armed' | 'active'
  protected?: string[]
  blocking?: string[]
  full_tunnel?: string[]
  policy_blocking?: string[]
  blocked_count: number
  error?: string
  error_key?: string
}

/** GET /api/v1/security/status: the full body (handlers_security.go handleSecurityStatus). */
export interface SecurityState extends SecurityStatus {
  leak_detection_configured?: boolean
  leak_check_available?: boolean
  kill_switch_status?: KillSwitchDetail
  /** Last kill switch apply failure, translated by the daemon (X-Kongtrol-Lang). */
  kill_switch_error?: { message: string; hint?: string; error_key: string; detail?: string }
  /** DNS servers the guard applies now (empty when idle) and their tunnel. */
  dns_guard_servers?: string[]
  dns_guard_interface?: string
  dns_guard_profiles?: Record<string, string[]>
}

/** GET /api/v1/security/blocked (handlers_security.go handleSecurityBlocked). */
export interface BlockedList {
  state: 'off' | 'armed' | 'active'
  /** CIDR prefixes blocked now. */
  prefixes: string[]
  count: number
  /** Tunnels that dropped and are blocked. */
  blocking: string[]
  /** Of those, the ones that carried all traffic (everything but endpoints is blocked). */
  full_tunnel: string[]
  /** Policies with on_tunnel_down: block whose VPN is off. */
  policy_blocking: string[]
}

/** internal/api/handlers_groups.go `settingsDTO`: the fields this page edits. */
export interface SecuritySettings {
  split_dns_enabled: boolean
  split_dns_interval: string
  kill_switch_mode: string
  kill_switch_allow_lan: boolean
  dns_guard_fallback_dns: string
  public_ip_check_enabled: boolean
  public_ip_endpoints: string[] | null
  /** profile → public IP expected while it is connected (leak check). */
  expected_ips?: Record<string, string>
}

/** PUT /api/v1/settings takes any subset (handlers_settings.go settingsPatch). */
export type SettingsPatch = Partial<SecuritySettings>

export type KillSwitchOverride = 'inherit' | 'on' | 'off'

/** internal/api/handlers_vpn.go `vpnDTO`: the fields this page shows. */
export interface ProfileSummary {
  name: string
  type: string
  kill_switch_override: KillSwitchOverride
}

/** GET /api/v1/policies (handlers_policy.go): the fields this page shows. */
export interface PolicySummary {
  name: string
  via: string
  domains: string[] | null
  ip_ranges: string[] | null
  apps: string[] | null
  on_tunnel_down: string
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
}

// ── Keys ──────────────────────────────────────────────────────────────────

export const securityKeys = {
  status: keys.security,
  settings: ['settings'] as const,
  vpns: ['vpns'] as const,
  policies: ['policies'] as const,
  policyEvents: ['events', 'policy'] as const,
  blocked: ['security', 'blocked'] as const,
} as const

/** Legacy security.js polled every 15 s; the shared security query polls every 5 s. */
export const SECURITY_POLL_MS = 5_000
const SLOW_POLL_MS = 15_000

const enc = encodeURIComponent

const fetchers = {
  status: () => http.get<SecurityState>('/api/v1/security/status'),
  settings: () => http.get<SecuritySettings>('/api/v1/settings'),
  vpns: () =>
    http.get<ProfileSummary[] | null>('/api/v1/vpns').then((l) => [...(l ?? [])].sort((a, b) => a.name.localeCompare(b.name))),
  policies: () =>
    http.get<PolicySummary[] | null>('/api/v1/policies').then((l) => [...(l ?? [])].sort((a, b) => a.name.localeCompare(b.name))),
  policyEvents: () =>
    http
      .get<{ events: DaemonEvent[] | null; seq: number }>('/api/v1/events?after=0')
      .then((r) => (r.events ?? []).filter((e) => e.kind === 'policy_tunnel_down').slice(-5).reverse()),
  blocked: () => http.get<BlockedList>('/api/v1/security/blocked'),
}

// ── Queries ───────────────────────────────────────────────────────────────

export function useSecurityState() {
  // Same key and cadence as data/resources.ts useSecurityQuery: one request.
  return useQuery({ queryKey: securityKeys.status, queryFn: fetchers.status, refetchInterval: SECURITY_POLL_MS })
}

export function useSecuritySettings() {
  return useQuery({ queryKey: securityKeys.settings, queryFn: fetchers.settings })
}

export function useProfiles() {
  return useQuery({ queryKey: securityKeys.vpns, queryFn: fetchers.vpns })
}

export function usePolicies() {
  return useQuery({ queryKey: securityKeys.policies, queryFn: fetchers.policies, refetchInterval: SLOW_POLL_MS })
}

/** Latest "traffic for a rule whose VPN is off" warnings (policy watch), newest first. */
export function usePolicyEvents() {
  return useQuery({ queryKey: securityKeys.policyEvents, queryFn: fetchers.policyEvents, refetchInterval: SLOW_POLL_MS })
}

/**
 * What the kill switch blocks right now. Read only while it is on (nothing
 * to list otherwise), at the status cadence so the list follows a drop.
 */
export function useBlockedDestinations(enabled: boolean) {
  return useQuery({
    queryKey: securityKeys.blocked,
    queryFn: fetchers.blocked,
    enabled,
    refetchInterval: enabled ? SECURITY_POLL_MS : false,
  })
}

// ── Mutations ─────────────────────────────────────────────────────────────

/** Kill switch on/off. The page confirms "off" before calling. */
export function useKillSwitchToggle() {
  const { t } = usePreferences()
  return useOptimisticMutation<boolean, unknown, SecurityState>({
    queryKey: securityKeys.status,
    mutationFn: (enabled) => http.post('/api/v1/security/killswitch', { enabled }),
    apply: (st, enabled) => ({
      ...st,
      kill_switch_enabled: enabled,
      kill_switch: enabled ? !!st.kill_switch : false,
      kill_switch_state: enabled ? 'armed' : 'disabled',
      kill_switch_status: enabled
        ? { ...(st.kill_switch_status ?? { blocked_count: 0 }), state: 'armed' }
        : { state: 'off', blocked_count: 0 },
    }),
    successMessage: (enabled) => t(enabled ? 'security.ks.toast.on' : 'security.ks.toast.off'),
  })
}

/** DNS guard on/off. The page confirms "off" before calling. */
export function useDNSGuardToggle() {
  const { t } = usePreferences()
  return useOptimisticMutation<boolean, unknown, SecurityState>({
    queryKey: securityKeys.status,
    mutationFn: (enabled) => http.post('/api/v1/security/dnsguard', { enabled }),
    apply: (st, enabled) => ({
      ...st,
      dns_guard_enabled: enabled,
      dns_guard: enabled ? !!st.dns_guard : false,
      dns_guard_state: enabled ? 'idle' : 'disabled',
    }),
    successMessage: (enabled) => t(enabled ? 'security.dns.toast.on' : 'security.dns.toast.off'),
  })
}

/**
 * Public IP lookup opt-in. The page shows the external-service notice and
 * asks before turning it on (AGENTS.md "Local only").
 */
export function usePublicIPToggle() {
  const { t } = usePreferences()
  const client = useQueryClient()
  return useOptimisticMutation<boolean, { public_ip_check_host?: string }, SecurityState>({
    queryKey: securityKeys.status,
    mutationFn: (enabled) => http.post('/api/v1/security/publicip', { enabled }),
    apply: (st, enabled) => ({ ...st, public_ip_check_enabled: enabled, ...(enabled ? {} : { leak_check: null }) }),
    successMessage: (enabled) => {
      // The settings form shows the same opt-in; keep it in step.
      void client.invalidateQueries({ queryKey: securityKeys.settings })
      return t(enabled ? 'publicip.enabled' : 'security.ip.toast.off')
    },
  })
}

/** "Revisar ahora": one lookup; the answer goes straight into the cache. */
export function usePublicIPCheck() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: () => http.post<{ leak_check?: LeakCheck | null }>('/api/v1/security/publicip/check', {}),
    onSuccess: (res) => {
      client.setQueryData<SecurityState>(securityKeys.status, (st) => (st ? { ...st, leak_check: res.leak_check ?? null } : st))
    },
    onSettled: () => client.invalidateQueries({ queryKey: securityKeys.status }),
  })
}

/** Per-profile kill switch override (inherit / on / off). */
export function useOverride() {
  const { t } = usePreferences()
  const client = useQueryClient()
  return useOptimisticMutation<{ name: string; override: KillSwitchOverride }, unknown, ProfileSummary[]>({
    queryKey: securityKeys.vpns,
    mutationFn: ({ name, override }) => http.put(`/api/v1/vpns/${enc(name)}/killswitch`, { override }),
    apply: (list, { name, override }) => list.map((p) => (p.name === name ? { ...p, kill_switch_override: override } : p)),
    successMessage: ({ name }) => {
      void client.invalidateQueries({ queryKey: securityKeys.status })
      return t('security.override.saved', { name })
    },
  })
}

/** A switch stored in the settings (allow LAN, split DNS): optimistic. */
export function useSettingsToggle() {
  const client = useQueryClient()
  return useOptimisticMutation<SettingsPatch, SecuritySettings, SecuritySettings>({
    queryKey: securityKeys.settings,
    mutationFn: (patch) => http.put<SecuritySettings>('/api/v1/settings', patch),
    apply: (st, patch) => ({ ...st, ...patch }),
    successMessage: () => {
      void client.invalidateQueries({ queryKey: securityKeys.status })
      return undefined
    },
  })
}

/**
 * Text settings (fallback DNS, split DNS interval, lookup services): saved
 * on "Guardar", not optimistic; the form shows a refusal inline next to the
 * field, with the daemon's translated "what to do" line.
 */
export function useSettingsSave() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (patch: SettingsPatch) => http.put<SecuritySettings>('/api/v1/settings', patch),
    onSuccess: (dto) => {
      client.setQueryData(securityKeys.settings, dto)
      void client.invalidateQueries({ queryKey: securityKeys.status })
    },
  })
}

/** Re-reads every resource on the page (the header's "Actualizar"). */
export function useRefreshSecurity() {
  const client = useQueryClient()
  return () =>
    Promise.all(
      [securityKeys.status, securityKeys.settings, securityKeys.vpns, securityKeys.policies, securityKeys.policyEvents].map(
        (queryKey) => client.refetchQueries({ queryKey }),
      ),
    )
}

// ── Pure helpers (tested) ─────────────────────────────────────────────────

/** Splits the services textarea: one URL per line, blanks dropped. */
export function parseEndpoints(text: string): string[] {
  return text
    .split(/\r?\n/)
    .map((l) => l.trim())
    .filter(Boolean)
}

/** Lines that are not https:// URLs (config.ValidatePublicIPEndpoint rejects them too). */
export function invalidEndpoints(list: readonly string[]): string[] {
  return list.filter((ep) => {
    try {
      const u = new URL(ep)
      return u.protocol !== 'https:' || !u.hostname
    } catch {
      return true
    }
  })
}

/** One editable row of security.integrity_check.expected_ips. */
export interface ExpectedRow {
  profile: string
  ip: string
}

/** The stored map as rows, by profile name (stable order). */
export function expectedRows(map: Record<string, string> | undefined): ExpectedRow[] {
  return Object.entries(map ?? {})
    .map(([profile, ip]) => ({ profile, ip }))
    .sort((a, b) => a.profile.localeCompare(b.profile))
}

export type ExpectedProblem =
  | { row: number; field: 'profile'; kind: 'pick' | 'duplicate' }
  | { row: number; field: 'ip'; kind: 'invalid' }

/** First problem per row: no VPN picked, a VPN twice, or a value that is not an IP. */
export function expectedProblems(rows: readonly ExpectedRow[], isIP: (v: string) => boolean): ExpectedProblem[] {
  const seen = new Set<string>()
  const out: ExpectedProblem[] = []
  rows.forEach((r, row) => {
    if (!r.profile) out.push({ row, field: 'profile', kind: 'pick' })
    else if (seen.has(r.profile)) out.push({ row, field: 'profile', kind: 'duplicate' })
    else if (!isIP(r.ip)) out.push({ row, field: 'ip', kind: 'invalid' })
    seen.add(r.profile)
  })
  return out
}

/** Rows → the PUT /api/v1/settings `expected_ips` patch (the whole map is replaced). */
export function expectedMap(rows: readonly ExpectedRow[]): Record<string, string> {
  const out: Record<string, string> = {}
  for (const r of rows) if (r.profile && r.ip.trim()) out[r.profile] = r.ip.trim()
  return out
}

/** LAN allowed now: allow_lan, or the legacy `mode: loose` (schema.go LANAllowed). */
export function lanAllowed(s: Pick<SecuritySettings, 'kill_switch_allow_lan' | 'kill_switch_mode'>): boolean {
  return s.kill_switch_allow_lan || s.kill_switch_mode === 'loose'
}

/** Patch for the "allow LAN" switch; turning it off also clears a legacy `mode: loose`. */
export function lanPatch(on: boolean, current: Pick<SecuritySettings, 'kill_switch_mode'>): SettingsPatch {
  if (on) return { kill_switch_allow_lan: true }
  return current.kill_switch_mode === 'loose'
    ? { kill_switch_allow_lan: false, kill_switch_mode: 'strict' }
    : { kill_switch_allow_lan: false }
}
