// Kongtrol UI — update notice and "delete all my data" hooks · v0.1.0 · 2026-11-18
import { useCallback, useEffect, useState } from 'react'
import { ApiError, api } from './client'
import type { UpdateStatus } from './types'

function toApiError(err: unknown): ApiError {
  return err instanceof ApiError ? err : new ApiError('server', 0, { error: String(err) })
}

export interface UpdatesState {
  status: UpdateStatus | null
  error: ApiError | null
  busy: boolean
  actionError: ApiError | null
  setEnabled: (on: boolean) => void
  reload: () => void
}

/** Opt-in update notice. Reading the state never contacts GitHub. */
export function useUpdates(): UpdatesState {
  const [status, setStatus] = useState<UpdateStatus | null>(null)
  const [error, setError] = useState<ApiError | null>(null)
  const [busy, setBusy] = useState(false)
  const [actionError, setActionError] = useState<ApiError | null>(null)

  const reload = useCallback(() => {
    api.updates().then(
      (st) => {
        setStatus(st)
        setError(null)
      },
      (err: unknown) => setError(toApiError(err)),
    )
  }, [])

  const setEnabled = useCallback((on: boolean) => {
    setBusy(true)
    setActionError(null)
    api
      .setUpdates(on)
      .then(
        (st) => setStatus(st),
        (err: unknown) => setActionError(toApiError(err)),
      )
      .finally(() => setBusy(false))
  }, [])

  useEffect(reload, [reload])

  return { status, error, busy, actionError, setEnabled, reload }
}

export interface PurgeState {
  busy: boolean
  done: boolean
  error: ApiError | null
  purge: () => void
}

/** "Borrar todos mis datos": one confirmed POST; the daemon restarts afterwards. */
export function usePurgeData(): PurgeState {
  const [busy, setBusy] = useState(false)
  const [done, setDone] = useState(false)
  const [error, setError] = useState<ApiError | null>(null)

  const purge = useCallback(() => {
    setBusy(true)
    setError(null)
    api
      .purgeData()
      .then(
        () => setDone(true),
        (err: unknown) => setError(toApiError(err)),
      )
      .finally(() => setBusy(false))
  }, [])

  return { busy, done, error, purge }
}
