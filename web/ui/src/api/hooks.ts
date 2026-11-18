// Kongtrol UI — page-facing data hooks · v0.4.0 · 2026-11-18
//
// Resource reads go through the shared cache (src/data, ADR-0038): these
// hooks only shape it for the existing cards. On-demand actions (doctor,
// export) keep local state: they are not resources and never run by
// themselves.
import { useCallback, useState } from 'react'
import { useDaemonState, useLiveState } from '../data/connection'
import {
  useHelperInstall,
  useHelperInstalling,
  useHelperStatus,
  useTunnelsQuery,
} from '../data/resources'
import { ApiError, api } from './client'
import { toApiError } from './errors'
import type { DoctorReport, HelperStatus, TunnelMetrics } from './types'

function errorOf(err: unknown): ApiError | null {
  return err ? toApiError(err) : null
}

const OFFLINE = new ApiError('offline', 0)

export interface TunnelsState {
  tunnels: TunnelMetrics[] | null
  error: ApiError | null
  live: boolean
  reload: () => void
}

/** REST snapshot, then the WebSocket feed; polls while the feed is down. */
export function useTunnels(): TunnelsState {
  const query = useTunnelsQuery()
  const live = useLiveState() === 'live'
  const daemon = useDaemonState()
  const { refetch } = query
  const reload = useCallback(() => void refetch(), [refetch])
  // Say "Kongtrol is not answering" at the first failed request, not after
  // the retries: with nothing cached there is nothing better to show.
  const error = errorOf(query.error) ?? (daemon === 'offline' && !query.data ? OFFLINE : null)
  return { tunnels: query.data ?? null, error, live, reload }
}

export interface HelperState {
  status: HelperStatus | null
  error: ApiError | null
  installing: boolean
  installed: boolean
  install: () => void
  reload: () => void
}

export function useHelper(): HelperState {
  const query = useHelperStatus()
  const mutation = useHelperInstall()
  const installing = useHelperInstalling()
  const { refetch } = query
  const { mutate } = mutation
  return {
    status: query.data ?? null,
    error: errorOf(query.error),
    installing,
    installed: mutation.isSuccess,
    install: useCallback(() => mutate(), [mutate]),
    reload: useCallback(() => void refetch(), [refetch]),
  }
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
