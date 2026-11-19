// Kongtrol UI — Perfiles + Grupos: daemon resources and changes · v1.1.1 · 2026-11-19
//
// The only place /profiles and /profiles/groups talk to the daemon (ADR-0038
// "pages only compose hooks + kit"). Profile and group CRUD write
// kongtrol.yaml + the OS keychain; the daemon applies them live (ADR-0053)
// and answers `restart_required: true` only when a restart is still needed
// (setup mode), so this module also owns the restart flow the banner and
// first run use.
//
// Cache keys live here, not in keys.ts, so parallel phase-2 work never edits
// the same file; they are plain resource names, so another page reading
// ['vpns'] (e.g. /security's override table) shares the same entry.
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ApiError, http } from '../api/client'
import type { Group } from '../api/types'
import { keys } from './keys'
import { useOptimisticMutation } from './optimistic'
import { createStore, useStore } from './store'

// ── Types (mirrors of internal/api JSON; source named above each) ─────────

/** The 10 adapter types (`oneof` in internal/config/schema.go). */
export type AdapterType =
  | 'openvpn'
  | 'wireguard'
  | 'protonvpn'
  | 'nordvpn'
  | 'surfshark'
  | 'forticlient'
  | 'ciscoanyconnect'
  | 'globalprotect'
  | 'tailscale'
  | 'cloudflarewarp'

export type KillSwitchOverride = 'inherit' | 'on' | 'off'

/** internal/api/handlers_vpn.go `vpnDTO` (GET /api/v1/vpns). Credentials are never sent back. */
export interface VPNProfile {
  name: string
  type: AdapterType
  host?: string
  port?: number
  server?: string
  protocol?: string
  config?: string
  priority: number
  failover: string[] | null
  auth_method?: string
  cert?: string
  key?: string
  username?: string
  has_username_credential: boolean
  has_password_credential: boolean
  kill_switch_override: KillSwitchOverride
  auto_connect: boolean
  country?: string
  city?: string
  group?: string
  interface?: string
  wait_timeout?: string
  app_path?: string
}

/** handlers_vpn.go `vpnProfileReq`: absent = keep (update), "" = clear. */
export type ProfileRequest = Partial<{
  name: string
  type: AdapterType
  host: string
  server: string
  protocol: string
  config: string
  priority: number
  failover: string[]
  auth_method: string
  cert: string
  key: string
  username: string
  password: string
  auto_connect: boolean
  country: string
  city: string
  group: string
  interface: string
  wait_timeout: string
  app_path: string
}>

/** Create / update / rename / delete answer (handlers_vpn.go). */
export interface ProfileChangeResult {
  status: 'created' | 'updated' | 'renamed' | 'deleted'
  profile: string
  /** The old name, when status is "renamed". */
  previous?: string
  restart_required?: boolean
  /** A connected VPN keeps its old settings until reconnected (ADR-0053). */
  reconnect_required?: boolean
  /** PUT ?reconnect=true: the tunnel was disconnected and is reconnecting. */
  reconnected?: boolean
}

/** POST /api/v1/vpns/import body (handlers_vpn_import.go importReq). */
export interface ImportRequest {
  name: string
  type: 'wireguard' | 'openvpn'
  content_base64: string
  overwrite?: boolean
}

/** POST /api/v1/vpns/import answer (201). */
export interface ImportResult {
  status: 'imported'
  path: string
  type: 'wireguard' | 'openvpn'
  overwritten: boolean
}

/** internal/api/handlers_setup.go `SetupStatus`. */
export interface SetupStatus {
  setup_mode: boolean
  needs_setup: boolean
  restart_pending: boolean
  config_path: string
}

/** internal/api/handlers_detect.go `DetectedClient`. */
export interface DetectedClient {
  label: string
  adapter_type: string
  path: string
}

/** POST /api/v1/groups/{name}/connect (handlers_groups.go). */
export interface GroupConnectResult {
  status: 'connecting'
  group: string
  started: string[]
  already_connected: string[]
}

/** POST /api/v1/groups/{name}/reload: 202, or 409 with missing_profiles. */
export interface GroupReloadResult {
  status: 'restarting' | 'restart_required'
  group: string
  restarted?: string[] | null
  skipped?: string[] | null
  missing_profiles?: string[]
}

// ── API calls ─────────────────────────────────────────────────────────────

const enc = encodeURIComponent

export const profilesApi = {
  list: () => http.get<VPNProfile[] | null>('/api/v1/vpns').then((l) => l ?? []),
  create: (body: ProfileRequest) => http.post<ProfileChangeResult>('/api/v1/vpns', body),
  update: (name: string, body: ProfileRequest) =>
    http.put<ProfileChangeResult>(`/api/v1/vpns/${enc(name)}`, body),
  remove: (name: string) => http.delete<ProfileChangeResult>(`/api/v1/vpns/${enc(name)}`),
  /** Validated like the privileged helper does; 409 api.err.import_exists unless overwrite. */
  importFile: (body: ImportRequest) => http.post<ImportResult>('/api/v1/vpns/import', body),
  setup: () => http.get<SetupStatus>('/api/v1/setup'),
  /** Older daemons have no detect endpoint: a 404 means "nothing to suggest". */
  detect: () =>
    http.get<DetectedClient[] | null>('/api/v1/vpns/detect').then(
      (l) => l ?? [],
      (err: unknown) => {
        if (err instanceof ApiError && err.status === 404) return []
        throw err
      },
    ),
  restart: () => http.post<{ status: string }>('/api/v1/restart'),
  groups: () => http.get<Group[] | null>('/api/v1/groups').then((l) => l ?? []),
  createGroup: (name: string, profiles: string[]) => http.post<unknown>('/api/v1/groups', { name, profiles }),
  updateGroup: (name: string, profiles: string[]) =>
    http.put<unknown>(`/api/v1/groups/${enc(name)}`, { name, profiles }),
  removeGroup: (name: string) => http.delete<unknown>(`/api/v1/groups/${enc(name)}`),
  connectGroup: (name: string) => http.post<GroupConnectResult>(`/api/v1/groups/${enc(name)}/connect`),
  disconnectGroup: (name: string) => http.post<unknown>(`/api/v1/groups/${enc(name)}/disconnect`),
  /** A 409 restart_required is an answer (which profiles need a restart), not a failure. */
  reloadGroup: async (name: string): Promise<GroupReloadResult> => {
    try {
      return await http.post<GroupReloadResult>(`/api/v1/groups/${enc(name)}/reload`)
    } catch (err) {
      if (err instanceof ApiError && err.status === 409 && err.body.status === 'restart_required') {
        return err.body as unknown as GroupReloadResult
      }
      throw err
    }
  },
}

export const profileKeys = {
  vpns: ['vpns'] as const,
  setup: ['setup'] as const,
  detect: ['vpns', 'detect'] as const,
} as const

// ── Reads ─────────────────────────────────────────────────────────────────

/** Loaded once per page view and after each change; no polling (parity: legacy loads once). */
export function useProfilesQuery() {
  return useQuery({ queryKey: profileKeys.vpns, queryFn: profilesApi.list, staleTime: 30_000 })
}

export function useSetupQuery() {
  return useQuery({ queryKey: profileKeys.setup, queryFn: profilesApi.setup, staleTime: 30_000 })
}

/** Installed VPN clients; scanning the disk is slow-ish, so once per page view. */
export function useDetectedClients(enabled = true) {
  return useQuery({
    queryKey: profileKeys.detect,
    queryFn: profilesApi.detect,
    staleTime: Infinity,
    retry: false,
    enabled,
  })
}

/** Groups share keys.groups with Inicio's quick launch. */
export function useGroupsList() {
  return useQuery({ queryKey: keys.groups, queryFn: profilesApi.groups })
}

// ── Restart pending (any profile change until the daemon restarts) ───────

const pending = createStore(false)

export function useRestartPending(): boolean {
  const local = useStore(pending)
  const setup = useSetupQuery().data
  return local || !!setup?.restart_pending
}

export function markRestartPending(): void {
  pending.set(true)
}

// ── Profile changes ───────────────────────────────────────────────────────

export interface SaveProfileVars {
  /** Profile being edited; undefined = create. */
  original?: string | undefined
  body: ProfileRequest
}

export function useSaveProfile() {
  const client = useQueryClient()
  return useMutation<ProfileChangeResult, unknown, SaveProfileVars>({
    mutationFn: ({ original, body }) =>
      original ? profilesApi.update(original, body) : profilesApi.create(body),
    onSuccess: (res) => {
      if (res?.restart_required !== false) markRestartPending()
      // A rename also rewrote policies, groups, schedules, expected IPs and
      // favorites (ADR-0042): every page re-reads its resource.
      if (res?.status === 'renamed') void client.invalidateQueries()
    },
    onSettled: () => {
      void client.invalidateQueries({ queryKey: profileKeys.vpns })
      void client.invalidateQueries({ queryKey: profileKeys.setup })
    },
  })
}

/** Delete: the row leaves at once and comes back if the daemon refuses. */
/** Rewrites a daemon error in the user's language (components/profiles/errors.ts). */
export type Localize = (err: unknown) => unknown

async function localized<T>(run: () => Promise<T>, localize: Localize): Promise<T> {
  try {
    return await run()
  } catch (err) {
    throw localize(err)
  }
}

export function useDeleteProfile(
  successMessage: (name: string) => string,
  localize: Localize,
  onRefused?: (err: unknown, name: string) => boolean,
) {
  return useOptimisticMutation<string, ProfileChangeResult, VPNProfile[]>({
    queryKey: profileKeys.vpns,
    mutationFn: (name) =>
      localized(async () => {
        const res = await profilesApi.remove(name)
        // Applied live (ADR-0053) unless the daemon says otherwise.
        if (res?.restart_required !== false) markRestartPending()
        return res
      }, localize),
    apply: (list, name) => list.filter((p) => p.name !== name),
    successMessage: (name) => successMessage(name),
    ...(onRefused ? { onRefused } : {}),
  })
}

/** "Importar archivo": stores the file; the form then fills its path. Not cached. */
export function useImportProfileFile() {
  return useMutation<ImportResult, unknown, ImportRequest>({ mutationFn: profilesApi.importFile })
}

// ── Restart ───────────────────────────────────────────────────────────────

export type RestartPhase = 'idle' | 'requesting' | 'stopping' | 'starting' | 'done' | 'failed' | 'timeout'

const restartPhase = createStore<RestartPhase>('idle')

export function useRestartPhase(): RestartPhase {
  return useStore(restartPhase)
}

const sleep = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms))

async function answers(): Promise<boolean> {
  try {
    await profilesApi.setup()
    return true
  } catch (err) {
    // A 4xx (e.g. 401) is still an answer: the daemon is up.
    return err instanceof ApiError && err.kind !== 'offline' && err.status > 0 && err.status < 500
  }
}

export interface RestartTiming {
  stepMs: number
  /** Polls while the old daemon disconnects its tunnels (legacy: 240 × 500 ms = 2 min). */
  stopSteps: number
  /** Polls until the new daemon answers (legacy: 120 × 500 ms = 1 min). */
  startSteps: number
}

export const RESTART_TIMING: RestartTiming = { stepMs: 500, stopSteps: 240, startSteps: 120 }

/**
 * POST /api/v1/restart, then wait for the old daemon to stop answering and
 * the new one to answer (legacy studio.js:846-895), then re-read everything:
 * the new daemon may have new profiles, groups and tunnels. Rejects with the
 * daemon's error on refusal (e.g. 501 when this daemon cannot restart).
 */
export async function runRestart(
  client: ReturnType<typeof useQueryClient>,
  timing: RestartTiming = RESTART_TIMING,
): Promise<'done' | 'timeout'> {
  restartPhase.set('requesting')
  try {
    await profilesApi.restart()
  } catch (err) {
    restartPhase.set('failed')
    throw err
  }
  restartPhase.set('stopping')
  let gone = false
  for (let i = 0; i < timing.stopSteps && !gone; i++) {
    await sleep(timing.stepMs)
    gone = !(await answers())
  }
  restartPhase.set('starting')
  let back = false
  for (let i = 0; gone && i < timing.startSteps && !back; i++) {
    await sleep(timing.stepMs)
    back = await answers()
  }
  if (!back) {
    restartPhase.set('timeout')
    return 'timeout'
  }
  pending.set(false)
  restartPhase.set('done')
  await client.invalidateQueries()
  return 'done'
}

export function resetRestartPhase(): void {
  restartPhase.set('idle')
}

// ── Group changes ─────────────────────────────────────────────────────────

export interface SaveGroupVars {
  original?: string | undefined
  name: string
  profiles: string[]
}

/** Create / update: the list shows the new group at once, rolled back on refusal. */
export function useSaveGroup(successMessage: (vars: SaveGroupVars) => string, localize: Localize) {
  return useOptimisticMutation<SaveGroupVars, unknown, Group[]>({
    queryKey: keys.groups,
    mutationFn: ({ original, name, profiles }) =>
      localized(
        () => (original ? profilesApi.updateGroup(original, profiles) : profilesApi.createGroup(name, profiles)),
        localize,
      ),
    apply: (list, { original, name, profiles }) => {
      const key = original ?? name
      const next = list.filter((g) => g.name !== key)
      return [...next, { name: key, profiles }].sort((a, b) => a.name.localeCompare(b.name))
    },
    successMessage: (vars) => successMessage(vars),
  })
}

export function useDeleteGroup(successMessage: (name: string) => string, localize: Localize) {
  return useOptimisticMutation<string, unknown, Group[]>({
    queryKey: keys.groups,
    mutationFn: (name) => localized(() => profilesApi.removeGroup(name), localize),
    apply: (list, name) => list.filter((g) => g.name !== name),
    successMessage: (name) => successMessage(name),
  })
}

export type GroupAction = 'connect' | 'disconnect' | 'reload'

export type GroupActionResult =
  | { action: 'connect'; result: GroupConnectResult }
  | { action: 'disconnect'; result: unknown }
  | { action: 'reload'; result: GroupReloadResult }

/** Connect / disconnect / reload a whole group; the live feed shows each tunnel's state. */
export function useGroupAction() {
  const client = useQueryClient()
  return useMutation<GroupActionResult, unknown, { name: string; action: GroupAction }>({
    mutationFn: async ({ name, action }): Promise<GroupActionResult> => {
      switch (action) {
        case 'connect':
          return { action, result: await profilesApi.connectGroup(name) }
        case 'disconnect':
          return { action, result: await profilesApi.disconnectGroup(name) }
        default:
          return { action: 'reload', result: await profilesApi.reloadGroup(name) }
      }
    },
    onSettled: () => client.invalidateQueries({ queryKey: keys.tunnels }),
  })
}
