// Kongtrol UI — "Guardar y activar" / "Guardar": create the profile, restart only if needed · v1.2.0 · 2026-11-25
//
// POST /api/v1/vpns answers `restart_required: false` when the daemon
// applied the new profile live (ADR-0053); then it is usable at once. Only
// `restart_required: true` (e.g. an older daemon) runs the
// shared restart (toasts, phases), which the sheet does by itself every
// time it is needed (owner decision, ADR-0055). A restart that fails or
// times out leaves the profile saved and offers to retry only the restart.
//
// Two ways to save a new profile (ADR-0115): "Guardar y activar" (the
// primary) also connects it once it is saved; "Guardar" only saves it. The
// hook remembers which one is running (`intent`), so the footer shows the
// spinner on the right button and a retried restart keeps the choice; the
// step tells the sheet, which connects (useTunnelActions) for "activar".
import { useState } from 'react'
import { useSaveProfile, type ProfileRequest } from '../../data/profiles'
import { useRestartAction } from '../profiles/useRestartAction'

export type ActivatePhase = 'idle' | 'saving' | 'restarting' | 'restartFailed'

/** "activate" = save and connect ("Guardar y activar"); "save" = save only ("Guardar"). */
export type SaveIntent = 'activate' | 'save'

export function useSaveAndActivate() {
  const save = useSaveProfile()
  const { restart } = useRestartAction()
  const [phase, setPhase] = useState<ActivatePhase>('idle')
  const [intent, setIntent] = useState<SaveIntent>('activate')

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
  const run = async (body: ProfileRequest, how: SaveIntent = 'activate') => {
    setIntent(how)
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

  return { run, retryRestart: activate, phase, intent, busy: phase === 'saving' || phase === 'restarting' }
}
