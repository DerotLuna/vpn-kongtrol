// Kongtrol UI — Reglas (/rules) and Horarios (/rules/schedule) resources · v1.2.0 · 2026-11-23
//
// Policies, policy meta, the scheduler rules and the scheduler's two
// settings, read and changed only through these hooks (ADR-0038). Keys
// name daemon resources, not pages; a key another page also reads
// (`settings`) holds the raw GET response so both share one cache entry.
// Saves are optimistic (the list changes at once and rolls back if the
// daemon refuses); a refusal is also returned to the form (mutateAsync)
// so it shows inline next to the fields, not only as a toast.
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { http } from '../api/client'
import { describeError } from '../api/errors'
import { usePreferences } from '../app/preferences-context'
import {
  policyFromPayload,
  type DraftTestResult,
  type ExplainResult,
  type Policy,
  type PolicyPayload,
  type SchedulePayload,
  type ScheduleRule,
} from '../components/rules/model'
import { useOptimisticMutation } from './optimistic'
import { toasts } from './toast'

/** GET /api/v1/policies/meta (handlers_policy.go metaDTO). */
export interface PolicyMeta {
  profiles: string[] | null
  config_path: string
}

/** The two GET /api/v1/settings fields this page edits; the rest is kept as sent. */
export interface SchedulerSettings {
  scheduler_enabled: boolean
  scheduler_interval: string
  [field: string]: unknown
}

export const rulesKeys = {
  // Other pages cache GET /api/v1/policies under ['policies'] sorted by name;
  // this page needs the daemon's evaluation order, so it keeps its own entry
  // under the same prefix (invalidating ['policies'] refreshes both).
  policies: ['policies', 'evaluation-order'] as const,
  anyPolicies: ['policies'] as const,
  policyMeta: ['policy-meta'] as const,
  explain: ['policy-explain'] as const,
  schedule: ['scheduler-rules'] as const,
  settings: ['settings'] as const,
} as const

const enc = encodeURIComponent

export const rulesApi = {
  policies: () => http.get<Policy[] | null>('/api/v1/policies').then((l) => l ?? []),
  policyMeta: () => http.get<PolicyMeta>('/api/v1/policies/meta'),
  createPolicy: (p: PolicyPayload) => http.post<unknown>('/api/v1/policies', p),
  updatePolicy: (name: string, p: Partial<PolicyPayload>) => http.put<unknown>(`/api/v1/policies/${enc(name)}`, p),
  deletePolicy: (name: string) => http.delete<unknown>(`/api/v1/policies/${enc(name)}`),
  /** Every rule of `from` goes through `to`, one validated save (ADR-0104). */
  movePolicies: (from: string, to: string) =>
    http.post<{ moved: string[] | null }>('/api/v1/policies/move', { from, to }),
  /** Re-reads kongtrol.yaml (rules edited by hand) without restarting. */
  reloadPolicies: () => http.post<{ policies: number }>('/api/v1/policies/reload'),
  /** Dry run of an unsaved rule against one destination or app. */
  testPolicy: (rule: PolicyPayload, target: string, app: string) =>
    http.post<DraftTestResult>('/api/v1/policies/test', { rule, target, app }),
  /** Which saved rule a destination uses and why (`kongtrol policy show`). */
  explain: (target: string) => http.get<ExplainResult>(`/api/v1/policies/explain?target=${enc(target)}`),
  scheduleRules: () => http.get<ScheduleRule[] | null>('/api/v1/scheduler/rules').then((l) => l ?? []),
  createSchedule: (p: SchedulePayload) => http.post<unknown>('/api/v1/scheduler/rules', p),
  updateSchedule: (name: string, p: SchedulePayload) =>
    http.put<unknown>(`/api/v1/scheduler/rules/${enc(name)}`, p),
  deleteSchedule: (name: string) => http.delete<unknown>(`/api/v1/scheduler/rules/${enc(name)}`),
  settings: () => http.get<SchedulerSettings>('/api/v1/settings'),
  /** Sends only the fields given (per-section save; the daemon keeps the rest). */
  saveSettings: (patch: Partial<Pick<SchedulerSettings, 'scheduler_enabled' | 'scheduler_interval'>>) =>
    http.put<SchedulerSettings>('/api/v1/settings', patch),
}

// ── Policies ──────────────────────────────────────────────────────────────

/** In the daemon's evaluation order (VPN priority, then position). Loaded once; no polling. */
export function usePoliciesQuery() {
  return useQuery({ queryKey: rulesKeys.policies, queryFn: rulesApi.policies })
}

export function usePolicyMetaQuery() {
  return useQuery({ queryKey: rulesKeys.policyMeta, queryFn: rulesApi.policyMeta })
}

export interface SavePolicyVars {
  /** The saved name when editing (renames supported); undefined when creating. */
  original?: string | undefined
  payload: PolicyPayload
}

export function useSavePolicy() {
  const { t } = usePreferences()
  const client = useQueryClient()
  return useOptimisticMutation<SavePolicyVars, unknown, Policy[]>({
    queryKey: rulesKeys.policies,
    mutationFn: async ({ original, payload }) => {
      const res = original ? await rulesApi.updatePolicy(original, payload) : await rulesApi.createPolicy(payload)
      void client.invalidateQueries({ queryKey: rulesKeys.anyPolicies })
      void client.invalidateQueries({ queryKey: rulesKeys.explain })
      return res
    },
    apply: (list, { original, payload }) => {
      const next = policyFromPayload(payload)
      if (!original) return [...list, next]
      return list.map((p) => (p.name.toLowerCase() === original.toLowerCase() ? { ...p, ...next } : p))
    },
    successMessage: ({ original, payload }) =>
      t(original ? 'rules.toast.updated' : 'rules.toast.created', { name: payload.name }),
  })
}

/**
 * "Pasar reglas" (ADR-0104): every rule of `from` goes through `to`. Not
 * optimistic: the caller waits for the answer and keeps its card on error.
 */
export function useMovePolicies() {
  const { t } = usePreferences()
  const client = useQueryClient()
  return useMutation({
    mutationFn: ({ from, to }: { from: string; to: string }) => rulesApi.movePolicies(from, to),
    onSuccess: (res, { to }) => {
      void client.invalidateQueries({ queryKey: rulesKeys.anyPolicies })
      void client.invalidateQueries({ queryKey: rulesKeys.explain })
      const n = res.moved?.length ?? 0
      toasts.show({ tone: 'ok', message: t(n === 1 ? 'add.move.done.one' : 'add.move.done.other', { n, to }) })
    },
  })
}

export function useTogglePolicy() {
  const { t } = usePreferences()
  const client = useQueryClient()
  return useOptimisticMutation<{ name: string; disabled: boolean; inline?: boolean }, unknown, Policy[]>({
    queryKey: rulesKeys.policies,
    mutationFn: async ({ name, disabled }) => {
      const res = await rulesApi.updatePolicy(name, { disabled })
      void client.invalidateQueries({ queryKey: rulesKeys.anyPolicies })
      void client.invalidateQueries({ queryKey: rulesKeys.explain })
      return res
    },
    apply: (list, { name, disabled }) => list.map((p) => p.name === name ? { ...p, disabled } : p),
    successMessage: ({ name, disabled }) => t(disabled ? 'rules.toast.paused' : 'rules.toast.activated', { name }),
    onRefused: (_, { inline }) => inline === true,
  })
}

export function useDeletePolicy() {
  const { t } = usePreferences()
  const client = useQueryClient()
  return useOptimisticMutation<string, unknown, Policy[]>({
    queryKey: rulesKeys.policies,
    mutationFn: async (name) => {
      const res = await rulesApi.deletePolicy(name)
      void client.invalidateQueries({ queryKey: rulesKeys.anyPolicies })
      void client.invalidateQueries({ queryKey: rulesKeys.explain })
      return res
    },
    apply: (list, name) => list.filter((p) => p.name !== name),
    successMessage: (name) => t('rules.toast.deleted', { name }),
  })
}

export function useReloadPolicies() {
  const { t } = usePreferences()
  const client = useQueryClient()
  return useMutation({
    mutationFn: rulesApi.reloadPolicies,
    onSuccess: (res) => {
      toasts.show({ tone: 'ok', message: t('rules.toast.reloaded', { count: res.policies }) })
    },
    onError: (err) => {
      const { message, fix } = describeError(err, t)
      toasts.show({ tone: 'bad', message: t('rules.toast.reloadFailed'), detail: `${message} ${fix}` })
    },
    onSettled: () => {
      void client.invalidateQueries({ queryKey: rulesKeys.anyPolicies })
      void client.invalidateQueries({ queryKey: rulesKeys.explain })
    },
  })
}

/** Dry run of the editor's draft; the result shows inline (no toast). */
export function useTestDraft() {
  return useMutation({
    mutationFn: ({ rule, target, app }: { rule: PolicyPayload; target: string; app: string }) =>
      rulesApi.testPolicy(rule, target, app),
  })
}

/** "Probar un destino" against the saved rules; re-asked after any rule change. */
export function useExplain(target: string) {
  return useQuery({
    queryKey: [...rulesKeys.explain, target],
    queryFn: () => rulesApi.explain(target),
    enabled: target !== '',
    staleTime: 0,
    retry: false,
  })
}

// ── Schedules ─────────────────────────────────────────────────────────────

export function useScheduleRulesQuery() {
  return useQuery({ queryKey: rulesKeys.schedule, queryFn: rulesApi.scheduleRules })
}

export function useSettingsQuery() {
  return useQuery({ queryKey: rulesKeys.settings, queryFn: rulesApi.settings })
}

export interface SaveScheduleVars {
  original?: string | undefined
  payload: SchedulePayload
}

export function useSaveSchedule() {
  const { t } = usePreferences()
  return useOptimisticMutation<SaveScheduleVars, unknown, ScheduleRule[]>({
    queryKey: rulesKeys.schedule,
    mutationFn: ({ original, payload }) =>
      original ? rulesApi.updateSchedule(original, payload) : rulesApi.createSchedule(payload),
    apply: (list, { original, payload }) => {
      if (!original) return [...list, payload]
      // The editor does not touch `enabled`: a paused rule stays paused.
      return list.map((r) => (r.name.toLowerCase() === original.toLowerCase() ? { ...r, ...payload } : r))
    },
    successMessage: ({ original, payload }) =>
      t(original ? 'schedule.toast.updated' : 'schedule.toast.created', { name: payload.name }),
  })
}

export function useDeleteSchedule() {
  const { t } = usePreferences()
  return useOptimisticMutation<string, unknown, ScheduleRule[]>({
    queryKey: rulesKeys.schedule,
    mutationFn: rulesApi.deleteSchedule,
    apply: (list, name) => list.filter((r) => r.name !== name),
    successMessage: (name) => t('schedule.toast.deleted', { name }),
  })
}

/**
 * Pause / resume one rule (PUT with only `enabled`). The daemon reloads
 * the scheduler on save (ADR-0042), so it applies at once: the switch moves
 * now and rolls back with a "what to do" toast if refused.
 */
export function useToggleSchedule() {
  return useOptimisticMutation<{ name: string; enabled: boolean }, unknown, ScheduleRule[]>({
    queryKey: rulesKeys.schedule,
    mutationFn: ({ name, enabled }) => http.put<unknown>(`/api/v1/scheduler/rules/${enc(name)}`, { enabled }),
    apply: (list, { name, enabled }) => list.map((r) => (r.name === name ? { ...r, enabled } : r)),
  })
}

type SchedulerPatch =Partial<Pick<SchedulerSettings, 'scheduler_enabled' | 'scheduler_interval'>>

/** Scheduler on/off (saves at once) and interval: only these fields are sent. */
export function useSaveSchedulerSettings() {
  const { t } = usePreferences()
  return useOptimisticMutation<SchedulerPatch, SchedulerSettings, SchedulerSettings>({
    queryKey: rulesKeys.settings,
    mutationFn: rulesApi.saveSettings,
    apply: (st, patch) => ({ ...st, ...patch }),
    successMessage: (patch) =>
      patch.scheduler_interval !== undefined ? t('schedule.toast.intervalSaved') : undefined,
  })
}
