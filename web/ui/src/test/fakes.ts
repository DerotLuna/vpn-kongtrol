// Shared test fakes: daemon API routes and a controllable WebSocket.
import type { QueryClient } from '@tanstack/react-query'
import { screen, within } from '@testing-library/react'
import type { UserEvent } from '@testing-library/user-event'
import { vi } from 'vitest'
import type { HelperStatus, TunnelMetrics } from '../api/types'
import { createQueryClient } from '../data/queryClient'

/**
 * The app's cache without automatic retries, for tests that answer a read
 * with a 5xx: the real client retries with a random backoff (up to seconds),
 * which made "shows the error" race the findBy timeout.
 */
export function noRetryClient(): QueryClient {
  const client = createQueryClient()
  client.setDefaultOptions({ ...client.getDefaultOptions(), queries: { ...client.getDefaultOptions().queries, retry: false } })
  return client
}

export function json(body: unknown, status = 200): Promise<Response> {
  return Promise.resolve(new Response(JSON.stringify(body), { status }))
}

export type Routes = Record<string, (init?: RequestInit) => Promise<Response>>

/** Stubs fetch with "METHOD path" routes; anything else answers 404. */
export function stubFetch(routes: Routes) {
  const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
    const route = routes[`${init?.method ?? 'GET'} ${String(input)}`]
    return route ? route(init) : json({ error: 'not found' }, 404)
  })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

/** WebSocket double: tests open, push frames and close it by hand. */
export class FakeSocket {
  static instances: FakeSocket[] = []
  onopen: (() => void) | null = null
  onmessage: ((e: { data: string }) => void) | null = null
  onclose: (() => void) | null = null
  closed = false
  url: string
  constructor(url: string) {
    this.url = url
    FakeSocket.instances.push(this)
  }
  open() {
    this.onopen?.()
  }
  push(map: Record<string, TunnelMetrics>) {
    this.onmessage?.({ data: JSON.stringify(map) })
  }
  drop() {
    this.onclose?.()
  }
  close() {
    this.closed = true
  }
  static reset() {
    FakeSocket.instances = []
  }
  static last(): FakeSocket {
    const s = FakeSocket.instances.at(-1)
    if (!s) throw new Error('no socket')
    return s
  }
}

export function tunnel(name: string, status: TunnelMetrics['Status'], extra: Partial<TunnelMetrics> = {}): TunnelMetrics {
  return {
    Name: name,
    Status: status,
    InterfaceName: '',
    AssignedIP: '',
    ConnectedAt: '',
    UptimeSeconds: 0,
    BytesSent: 0,
    BytesReceived: 0,
    LatencyMS: 0,
    UpdatedAt: '',
    ...extra,
  }
}

export const helperMissing: HelperStatus = {
  installed: false,
  running: false,
  available: false,
  elevated: false,
  helper_version: '',
  cli_version: 'v1.2.3',
  needs_install: true,
}

export const helperReady: HelperStatus = {
  ...helperMissing,
  installed: true,
  running: true,
  available: true,
  needs_install: false,
  helper_version: 'v1.2.3',
}

/** Chooses an option of a kit Select / Combobox (own listbox, not a native <select>). */
export async function pickOption(user: UserEvent, combobox: HTMLElement, option: string | RegExp): Promise<void> {
  await user.click(combobox)
  const list = await screen.findByRole('listbox')
  await user.click(within(list).getByRole('option', { name: option }))
}
