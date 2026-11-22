// Kongtrol UI — "Guardar y activar": create the profile, restart only if needed · v1.1.0 · 2026-11-19
//
// POST /api/v1/vpns answers `restart_required: false` when the daemon
// applied the new profile live (ADR-0053); then it is usable at once. Only
// `restart_required: true` (e.g. an older daemon) runs the
// shared restart (toasts, phases), which the sheet does by itself every
// time it is needed (owner decision, ADR-0055). A restart that fails or
// times out leaves the profile saved and offers to retry only the restart.
import { useState } from 'react'
import { useSaveProfile, type ProfileRequest } from '../../data/profiles'
import { useRestartAction } from '../profiles/useRestartAction'

export type ActivatePhase = 'idle' | 'saving' | 'restarting' | 'restartFailed'

export function useSaveAndActivate() {
  const save = useSaveProfile()
  const { restart } = useRestartAction()
  const [phase, setPhase] = useState<ActivatePhase>('idle')

  const activate = async () => {
    setPhase('restarting')
    const outcome = await restart()
    if (outcome === 'done' || outcome === 'reauth') {
      setPhase('idle')
      return outcome
    }
    setPhase('restartFailed')
    return 'failed'
  }

  /** Creates the profile (a refusal is thrown to the form) and restarts when needed. */
  const run = async (body: ProfileRequest) => {
    setPhase('saving')
    let restartRequired: boolean
    try {
      const res = await save.mutateAsync({ body })
      // Absent (very old daemon) counts as required, like useSaveProfile does.
      restartRequired = res?.restart_required !== false
    } catch (err) {
      setPhase('idle')
      throw err
    }
    if (!restartRequired) {
      setPhase('idle')
      return 'done'
    }
    return activate()
  }

  return { run, retryRestart: activate, phase, busy: phase === 'saving' || phase === 'restarting' }
}
