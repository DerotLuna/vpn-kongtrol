// Kongtrol UI — data hooks · v0.3.0 · 2026-11-18
import { useCallback, useEffect, useState } from 'react'
import { ApiError, api, subscribeTunnels } from './client'
import type { DoctorReport, HelperStatus, PublicIPStatus, TunnelMetrics } from './types'

function toApiError(err: unknown): ApiError {
  return err instanceof ApiError ? err : new ApiError('server', 0, { error: String(err) })
}

const POLL_MS = 5_000

export interface TunnelsState {
  tunnels: TunnelMetrics[] | null
  error: ApiError | null
  live: boolean
  reload: () => void
}

/** REST snapshot first, then the WebSocket feed; polls while the feed is down. */
export function useTunnels(): TunnelsState {
  const [tunnels, setTunnels] = useState<TunnelMetrics[] | null>(null)
  const [error, setError] = useState<ApiError | null>(null)
  const [live, setLive] = useState(false)

  const reload = useCallback(() => {
    api.tunnels().then(
      (list) => {
        setTunnels(list)
        setError(null)
      },
      (err: unknown) => setError(toApiError(err)),
    )
  }, [])

  useEffect(() => {
    reload()
    let poll: ReturnType<typeof setInterval> | undefined
    const stopPolling = () => {
      clearInterval(poll)
      poll = undefined
    }
    const close = subscribeTunnels(
      (list) => {
        stopPolling()
        setLive(true)
        setTunnels(list)
        setError(null)
      },
      () => {
        setLive(false)
        poll ??= setInterval(reload, POLL_MS)
      },
    )
    return () => {
      stopPolling()
      close()
    }
  }, [reload])

  return { tunnels, error, live, reload }
}

export interface HelperState {
  status: HelperStatus | null
  error: ApiError | null
  installing: boolean
  installError: ApiError | null
  installed: boolean
  install: () => void
  reload: () => void
}

export function useHelper(): HelperState {
  const [status, setStatus] = useState<HelperStatus | null>(null)
  const [error, setError] = useState<ApiError | null>(null)
  const [installing, setInstalling] = useState(false)
  const [installError, setInstallError] = useState<ApiError | null>(null)
  const [installed, setInstalled] = useState(false)

  const reload = useCallback(() => {
    api.helperStatus().then(
      (st) => {
        setStatus(st)
        setError(null)
      },
      (err: unknown) => setError(toApiError(err)),
    )
  }, [])

  const install = useCallback(() => {
    setInstalling(true)
    setInstallError(null)
    api
      .installHelper()
      .then(
        (st) => {
          setStatus(st)
          setInstalled(true)
        },
        (err: unknown) => setInstallError(toApiError(err)),
      )
      .finally(() => setInstalling(false))
  }, [])

  useEffect(reload, [reload])

  return { status, error, installing, installError, installed, install, reload }
}

export interface PublicIPState {
  status: PublicIPStatus | null
  error: ApiError | null
  busy: boolean
  actionError: ApiError | null
  /** Set after the user turned the lookup on: shows which service sees the IP. */
  justEnabled: boolean
  setEnabled: (on: boolean) => void
  check: () => void
  reload: () => void
}

/**
 * Opt-in public IP lookup. Reading the state never contacts an external
 * service; only check() (the user's "Revisar") asks the daemon for a lookup.
 */
export function usePublicIP(): PublicIPState {
  const [status, setStatus] = useState<PublicIPStatus | null>(null)
  const [error, setError] = useState<ApiError | null>(null)
  const [busy, setBusy] = useState(false)
  const [actionError, setActionError] = useState<ApiError | null>(null)
  const [justEnabled, setJustEnabled] = useState(false)

  const reload = useCallback(() => {
    api.publicIP().then(
      (st) => {
        setStatus(st)
        setError(null)
      },
      (err: unknown) => setError(toApiError(err)),
    )
  }, [])

  const run = useCallback(
    (action: () => Promise<unknown>, after?: () => void) => {
      setBusy(true)
      setActionError(null)
      action()
        .then(
          () => after?.(),
          (err: unknown) => setActionError(toApiError(err)),
        )
        .finally(() => {
          setBusy(false)
          reload()
        })
    },
    [reload],
  )

  const setEnabled = useCallback(
    (on: boolean) => run(() => api.setPublicIP(on), () => setJustEnabled(on)),
    [run],
  )
  const check = useCallback(() => run(() => api.checkPublicIP()), [run])

  useEffect(() => {
    reload()
    const poll = setInterval(reload, POLL_MS) // local state only
    return () => clearInterval(poll)
  }, [reload])

  return { status, error, busy, actionError, justEnabled, setEnabled, check, reload }
}

export interface DoctorState {
  report: DoctorReport | null
  error: ApiError | null
  running: boolean
  run: () => void
}

/**
 * Diagnostics on demand: the daemon runs the same checks as
 * `kongtrol doctor` (a few seconds, starts VPN clients' --version), so it
 * never runs on its own or on a timer.
 */
export function useDoctor(): DoctorState {
  const [report, setReport] = useState<DoctorReport | null>(null)
  const [error, setError] = useState<ApiError | null>(null)
  const [running, setRunning] = useState(false)

  const run = useCallback(() => {
    setRunning(true)
    setError(null)
    api
      .doctor()
      .then(setReport, (err: unknown) => setError(toApiError(err)))
      .finally(() => setRunning(false))
  }, [])

  return { report, error, running, run }
}

export interface ExportState {
  busy: boolean
  error: ApiError | null
  done: boolean
  download: () => void
}

/** Downloads kongtrol.yaml as kongtrol-export.yaml (no passwords inside). */
export function useConfigExport(): ExportState {
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<ApiError | null>(null)
  const [done, setDone] = useState(false)

  const download = useCallback(() => {
    setBusy(true)
    setError(null)
    setDone(false)
    api
      .exportConfig()
      .then(
        (blob) => {
          const url = URL.createObjectURL(blob)
          const a = document.createElement('a')
          a.href = url
          a.download = EXPORT_FILENAME
          document.body.append(a)
          a.click()
          a.remove()
          URL.revokeObjectURL(url)
          setDone(true)
        },
        (err: unknown) => setError(toApiError(err)),
      )
      .finally(() => setBusy(false))
  }, [])

  return { busy, error, done, download }
}

/** internal/configexport.Filename */
export const EXPORT_FILENAME = 'kongtrol-export.yaml'
