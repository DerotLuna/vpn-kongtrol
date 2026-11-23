// Kongtrol UI — Ajustes › Al iniciar · v1.1.0 · 2026-11-23
//
// GET/PUT /api/v1/startup (internal/api/handlers_startup.go, ADR-0098):
// "Abrir Kongtrol al iniciar sesión" (the OS's per-user autostart entry)
// and "Al iniciar, volver a conectar las VPN que tenías conectadas"
// (preferences.json). The PUT is a patch; each switch sends only its field.
// `ask` is the question after the first connect (ADR-0101), answered with
// {"restore_answer": bool}: the daemon then sets both switches itself.
// feature:restore_at_start
import { useQuery } from '@tanstack/react-query'
import { http } from '../api/client'
import { useOptimisticMutation } from './optimistic'

/** internal/api/handlers_startup.go `startupDTO`. */
export interface StartupSettings {
  start_at_login: boolean
  /** False on a CLI-only install: there is no tray to open at sign-in. */
  start_at_login_supported: boolean
  restore_tunnels: boolean
  /** "¿Volver a conectar tus VPN al encender el equipo?" is waiting (ADR-0101). */
  ask?: boolean
}

export type StartupPatch = Partial<Pick<StartupSettings, 'start_at_login' | 'restore_tunnels'>>

export const startupKey = ['startup'] as const

export function useStartupQuery() {
  return useQuery({
    queryKey: startupKey,
    queryFn: () => http.get<StartupSettings>('/api/v1/startup'),
    refetchOnWindowFocus: false,
  })
}

export function useSaveStartup() {
  return useOptimisticMutation<StartupPatch, StartupSettings, StartupSettings>({
    queryKey: startupKey,
    mutationFn: (patch) => http.put<StartupSettings>('/api/v1/startup', patch),
    apply: (current, patch) => ({ ...current, ...patch }),
  })
}

/** Answers the restore question: true = "Sí, activar", false = "Ahora no". */
export function useAnswerRestore() {
  return useOptimisticMutation<boolean, StartupSettings, StartupSettings>({
    queryKey: startupKey,
    mutationFn: (on) => http.put<StartupSettings>('/api/v1/startup', { restore_answer: on }),
    apply: (current, on) => ({
      ...current,
      ask: false,
      restore_tunnels: on,
      start_at_login: on && current.start_at_login_supported ? true : current.start_at_login,
    }),
  })
}
