// Kongtrol UI — the restart action shared by the banner, toasts and the add sheet · v1.1.0 · 2026-11-19
//
// Runs data/profiles.ts runRestart and reports the result as one toast
// (running, done, timeout, or the daemon's refusal with what to do). It
// also returns the outcome, so a flow can go on only once Kongtrol is back
// ("Guardar y activar" in the add sheet, ADR-0055).
import { useCallback } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router'
import { describeError } from '../../api/errors'
import { usePreferences } from '../../app/preferences-context'
import { resetRestartPhase, runRestart, useRestartPhase, type RestartPhase } from '../../data/profiles'
import { toasts } from '../../data/toast'
import { localizeError } from './errors'

const BUSY: RestartPhase[] = ['requesting', 'stopping', 'starting']

export type RestartOutcome = 'done' | 'timeout' | 'failed' | 'busy' | 'reauth'

export function useRestartAction() {
  const client = useQueryClient()
  const navigate = useNavigate()
  const { t } = usePreferences()
  const phase = useRestartPhase()
  const busy = BUSY.includes(phase)
  const restart = useCallback(
    async (opts: { thenHome?: boolean; progress?: string } = {}): Promise<RestartOutcome> => {
      if (BUSY.includes(phase)) return 'busy'
      // The progress toast stays until there is a result, then gives way to it.
      const running = toasts.show({ tone: 'info', message: opts.progress ?? t('profiles.restart.running'), duration: 0 })
      try {
        const result = await runRestart(client)
        toasts.dismiss(running)
        if (result === 'reauth') return result
        if (result === 'timeout') {
          toasts.show({ tone: 'bad', message: t('profiles.restart.timeout'), detail: t('profiles.restart.timeout.fix') })
          return 'timeout'
        }
        toasts.show({ tone: 'ok', message: t('profiles.restart.done') })
        resetRestartPhase()
        if (opts.thenHome) void navigate('/')
        return 'done'
      } catch (err) {
        toasts.dismiss(running)
        const { message, fix } = describeError(localizeError(err, t), t)
        toasts.show({ tone: 'bad', message: t('profiles.restart.failed'), detail: `${message} · ${fix}` })
        return 'failed'
      }
    },
    [client, navigate, phase, t],
  )
  return { restart, busy, phase }
}
