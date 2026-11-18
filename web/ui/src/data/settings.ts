// Kongtrol UI — Ajustes data: settings, setup, restart, shutdown · v1.1.0 · 2026-11-18
//
// GET/PUT /api/v1/settings (internal/api/handlers_settings.go). The PUT is a
// patch: every section sends only its own fields, so saving one section never
// overwrites another page's unsaved edits (legacy bug: settings.js saved the
// whole form). Read once, no polling (legacy parity); a save writes the
// expected values at once and rolls back if the daemon refuses
// (useOptimisticMutation), then re-reads the daemon's truth.
//
// Values are validated here before they are sent, with the same rules the
// daemon applies (time.ParseDuration; max size a positive whole number), so
// an empty box can never be saved as 0 (legacy bug: settings.js:201).
import { useCallback, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { http } from '../api/client'
import { toApiError } from '../api/errors'
import type { ApiError } from '../api/client'
import { usePreferences } from '../app/preferences-context'
import type { MessageKey } from '../i18n/messages'
import { useOptimisticMutation } from './optimistic'

/** internal/api/handlers_groups.go `settingsDTO` (the fields these pages use). */
export interface Settings {
  /** Read-only here: changed only with `kongtrol config dashboard set-port|set-bind`. */
  dashboard_bind: string
  dashboard_port: number
  health_check_interval: string
  health_check_timeout: string
  audit_log_path: string
  audit_log_max_size_mb: number
  /** Days kept, today included (0 = today and yesterday). */
  audit_log_retention_days?: number
  audit_log_sign: boolean
}

export type SettingsPatch = Partial<Omit<Settings, 'dashboard_bind' | 'dashboard_port'>>

/** GET /api/v1/setup — internal/api/handlers_setup.go `SetupStatus`. */
export interface SetupStatus {
  setup_mode: boolean
  needs_setup: boolean
  restart_pending: boolean
  config_path: string
}

export const settingsKeys = {
  settings: ['settings'] as const,
  setup: ['setup'] as const,
}

export function useSettingsQuery() {
  return useQuery({
    queryKey: settingsKeys.settings,
    queryFn: () => http.get<Settings>('/api/v1/settings'),
    staleTime: Infinity,
    refetchOnWindowFocus: false,
  })
}

export function useSaveSettings() {
  const { t } = usePreferences()
  return useOptimisticMutation<SettingsPatch, Settings, Settings>({
    queryKey: settingsKeys.settings,
    mutationFn: (patch) => http.put<Settings>('/api/v1/settings', patch),
    apply: (current, patch) => ({ ...current, ...patch }),
    successMessage: () => t('settings.toast.saved'),
  })
}

export function useSetupStatus() {
  return useQuery({
    queryKey: settingsKeys.setup,
    queryFn: () => http.get<SetupStatus>('/api/v1/setup'),
    staleTime: Infinity,
  })
}

// ── Validation ────────────────────────────────────────────────────────────

export interface Invalid {
  message: MessageKey
  fix: MessageKey
}

// time.ParseDuration: one or more <number><unit>, units ns us µs ms s m h.
const DURATION = /^(\d+(\.\d+)?(ns|us|µs|ms|s|m|h))+$/
const UNIT_MS: Record<string, number> = { ns: 1e-6, us: 1e-3, µs: 1e-3, ms: 1, s: 1_000, m: 60_000, h: 3_600_000 }

/** Milliseconds of a Go duration, or null when Go would refuse it. */
export function parseDuration(value: string): number | null {
  const v = value.trim()
  if (!DURATION.test(v)) return null
  let total = 0
  for (const m of v.matchAll(/(\d+(?:\.\d+)?)(ns|us|µs|ms|s|m|h)/g)) {
    total += Number(m[1]) * (UNIT_MS[m[2] as string] ?? 0)
  }
  return total
}

/** A health check every less than a second would flood the tunnels. */
export function validateDuration(value: string, minMs = 1_000): Invalid | null {
  if (!value.trim()) return { message: 'settings.err.durationEmpty', fix: 'settings.err.durationFix' }
  const ms = parseDuration(value)
  if (ms === null) return { message: 'settings.err.duration', fix: 'settings.err.durationFix' }
  if (ms < minMs) return { message: 'settings.err.durationShort', fix: 'settings.err.durationFix' }
  return null
}

export const MAX_SIZE_MB = { min: 1, max: 1024 } as const

/** Whole MB between 1 and 1024 (the daemon limit). Empty is an error, never 0. */
export function validateMaxSize(value: string): Invalid | null {
  const v = value.trim()
  if (!v) return { message: 'settings.err.sizeEmpty', fix: 'settings.err.sizeFix' }
  if (!/^\d+$/.test(v)) return { message: 'settings.err.size', fix: 'settings.err.sizeFix' }
  const n = Number(v)
  if (n < MAX_SIZE_MB.min || n > MAX_SIZE_MB.max) return { message: 'settings.err.size', fix: 'settings.err.sizeFix' }
  return null
}

export const RETENTION_DAYS = { min: 0, max: 365 } as const

/** Whole days 0–365 (0 = today and yesterday, the daemon default). Empty is an error. */
export function validateRetention(value: string): Invalid | null {
  const v = value.trim()
  if (!/^\d+$/.test(v) || Number(v) > RETENTION_DAYS.max) {
    return { message: 'activity.config.retention.err', fix: 'activity.config.retention.err.fix' }
  }
  return null
}

// ── Restart / shutdown ────────────────────────────────────────────────────

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))

async function answers(): Promise<boolean> {
  try {
    await http.get<SetupStatus>('/api/v1/setup')
    return true
  } catch {
    return false
  }
}

export type LifecyclePhase = 'idle' | 'running' | 'done' | 'failed' | 'timeout'

export interface LifecycleState {
  phase: LifecyclePhase
  error: ApiError | null
  run: () => void
}

/** Poll steps for the restart wait (legacy studio.js:846-895); tests shorten them. */
export const RESTART_WAIT = { stepMs: 500, goneSteps: 240, backSteps: 120 }

/**
 * POST /api/v1/restart (handlers_setup.go:54): the daemon starts its
 * replacement and stops. Wait for the old one to go (tunnels disconnect,
 * up to 2 min) and the new one to answer (up to 1 min).
 */
export function useRestart(onBack?: () => void): LifecycleState {
  const [phase, setPhase] = useState<LifecyclePhase>('idle')
  const [error, setError] = useState<ApiError | null>(null)
  const run = useCallback(() => {
    setPhase('running')
    setError(null)
    void (async () => {
      try {
        await http.post<unknown>('/api/v1/restart')
      } catch (err) {
        setError(toApiError(err))
        setPhase('failed')
        return
      }
      let gone = false
      for (let i = 0; i < RESTART_WAIT.goneSteps && !gone; i++) {
        await sleep(RESTART_WAIT.stepMs)
        gone = !(await answers())
      }
      let back = false
      for (let i = 0; gone && i < RESTART_WAIT.backSteps && !back; i++) {
        await sleep(RESTART_WAIT.stepMs)
        back = await answers()
      }
      if (!back) {
        setPhase('timeout')
        return
      }
      setPhase('done')
      onBack?.()
    })()
  }, [onBack])
  return { phase, error, run }
}

/** POST /api/v1/shutdown (handlers_tunnels.go:21): disconnect everything and stop. */
export function useShutdown(): LifecycleState {
  const [phase, setPhase] = useState<LifecyclePhase>('idle')
  const [error, setError] = useState<ApiError | null>(null)
  const run = useCallback(() => {
    setPhase('running')
    setError(null)
    http.post<unknown>('/api/v1/shutdown').then(
      () => setPhase('done'),
      (err: unknown) => {
        setError(toApiError(err))
        setPhase('failed')
      },
    )
  }, [])
  return { phase, error, run }
}
