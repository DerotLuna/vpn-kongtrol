import { QueryClientProvider } from '@tanstack/react-query'
import { act, render, renderHook, screen, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '../api/client'
import type { SecurityStatus, TunnelMetrics } from '../api/types'
import { PreferencesProvider } from '../app/PreferencesProvider'
import { DataList, type Column } from '../components/ui/DataList'
import { FakeSocket, helperMissing, helperReady, json, stubFetch, tunnel } from '../test/fakes'
import { backoffDelay } from './backoff'
import { connection, liveState } from './connection'
import { keys } from './keys'
import { parseFrame, startLiveFeed } from './live'
import { createQueryClient, shouldRetry } from './queryClient'
import {
  helperUiState,
  summarizeTunnels,
  useHelperReadyToast,
  useHelperStatus,
  useSecurityToggle,
  useTunnelAction,
  useTunnelsQuery,
} from './resources'
import { toasts } from './toast'

function setup() {
  const client = createQueryClient()
  const wrapper = ({ children }: { children: ReactNode }) => (
    <PreferencesProvider>
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    </PreferencesProvider>
  )
  return { client, wrapper }
}

beforeEach(() => {
  Object.defineProperty(navigator, 'languages', { value: ['en-US'], configurable: true })
  FakeSocket.reset()
})
afterEach(() => vi.useRealTimers())

describe('backoff and retry policy', () => {
  it('full jitter stays within [floor, min(cap, base·2^n)]', () => {
    expect(backoffDelay(0, { random: () => 1 })).toBe(1_000)
    expect(backoffDelay(3, { random: () => 1 })).toBe(8_000)
    expect(backoffDelay(20, { random: () => 1 })).toBe(30_000)
    expect(backoffDelay(5, { random: () => 0 })).toBe(250)
  })

  it('retries only "daemon not answering" and 5xx, at most twice', () => {
    expect(shouldRetry(0, new ApiError('offline', 0))).toBe(true)
    expect(shouldRetry(1, new ApiError('server', 503))).toBe(true)
    expect(shouldRetry(2, new ApiError('offline', 0))).toBe(false)
    expect(shouldRetry(0, new ApiError('server', 400))).toBe(false)
    expect(shouldRetry(0, new ApiError('unauthorized', 401))).toBe(false)
  })
})

describe('cache', () => {
  it('two readers of one resource share one request (de-dup)', async () => {
    const fetchMock = stubFetch({
      'GET /api/v1/preferences': () => json({ language: 'en', theme: '' }),
      'GET /api/v1/tunnels': () => json([tunnel('b', 'connected'), tunnel('a', 'disconnected')]),
    })
    const { wrapper } = setup()
    const { result } = renderHook(() => [useTunnelsQuery(), useTunnelsQuery()], { wrapper })
    await waitFor(() => expect(result.current[0]?.data).toBeDefined())
    expect(fetchMock.mock.calls.filter(([u]) => u === '/api/v1/tunnels')).toHaveLength(1)
    // sorted by name, whichever order the daemon sent
    expect(result.current[1]?.data?.map((t) => t.Name)).toEqual(['a', 'b'])
  })

  it('a network failure marks the daemon offline; any answer marks it online', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.reject(new TypeError('Failed to fetch'))))
    const { wrapper } = setup()
    renderHook(() => useHelperStatus(), { wrapper })
    await waitFor(() => expect(connection.get()).toBe('offline'))
    stubFetch({ 'GET /api/v1/helper/status': () => json(helperReady) })
    const second = setup()
    renderHook(() => useHelperStatus(), { wrapper: second.wrapper })
    await waitFor(() => expect(connection.get()).toBe('online'))
  })
})

describe('live feed', () => {
  it('parses frames and rejects malformed ones', () => {
    expect(parseFrame('{"b":{"Name":"b"},"a":{"Name":"a"}}')?.map((t) => t.Name)).toEqual(['a', 'b'])
    expect(parseFrame('null')).toEqual([])
    expect(parseFrame('not json')).toBeNull()
    expect(parseFrame('42')).toBeNull()
  })

  it('opens each socket with a fresh one-time ticket', async () => {
    const client = createQueryClient()
    let n = 0
    const ticket = vi.fn(() => Promise.resolve(`t${++n}`))
    const stop = startLiveFeed(client, {
      WebSocketImpl: FakeSocket as unknown as typeof WebSocket,
      url: 'ws://x/api/v1/ws/metrics',
      ticket,
      delay: () => 0,
    })
    await vi.waitFor(() => expect(FakeSocket.instances).toHaveLength(1))
    expect(FakeSocket.last().url).toBe('ws://x/api/v1/ws/metrics?ticket=t1')
    FakeSocket.last().drop()
    await vi.waitFor(() => expect(FakeSocket.instances).toHaveLength(2))
    expect(FakeSocket.last().url).toBe('ws://x/api/v1/ws/metrics?ticket=t2')
    stop()
  })

  it('retries with backoff when no ticket is given (no socket without one)', async () => {
    vi.useFakeTimers()
    const client = createQueryClient()
    const ticket = vi.fn().mockRejectedValueOnce(new Error('unauthorized')).mockResolvedValue('ok')
    const stop = startLiveFeed(client, {
      WebSocketImpl: FakeSocket as unknown as typeof WebSocket,
      url: 'ws://x',
      ticket,
      delay: () => 100,
    })
    await vi.advanceTimersByTimeAsync(0)
    expect(FakeSocket.instances).toHaveLength(0)
    expect(liveState.get()).toBe('down')
    await vi.advanceTimersByTimeAsync(100)
    expect(FakeSocket.instances).toHaveLength(1)
    expect(FakeSocket.last().url).toBe('ws://x?ticket=ok')
    stop()
  })

  it('writes frames into the cache and keeps unchanged tunnels identical', async () => {
    const client = createQueryClient()
    const stop = startLiveFeed(client, {
      WebSocketImpl: FakeSocket as unknown as typeof WebSocket,
      url: 'ws://x',
      ticket: () => Promise.resolve('t'),
    })
    await vi.waitFor(() => expect(FakeSocket.instances).toHaveLength(1))
    const ws = FakeSocket.last()
    act(() => ws.open())
    expect(liveState.get()).toBe('live')

    const a = tunnel('a', 'connected', { BytesReceived: 10 })
    const b = tunnel('b', 'disconnected')
    ws.push({ a, b })
    const first = client.getQueryData<TunnelMetrics[]>(keys.tunnels)!
    ws.push({ a: { ...a, BytesReceived: 20 }, b: { ...b } })
    const second = client.getQueryData<TunnelMetrics[]>(keys.tunnels)!

    expect(second[0]).not.toBe(first[0]) // a changed
    expect(second[0]?.BytesReceived).toBe(20)
    expect(second[1]).toBe(first[1]) // b is the very same object
    stop()
    expect(ws.closed).toBe(true)
  })

  it('reconnects with backoff, then refreshes every resource after the outage', async () => {
    vi.useFakeTimers()
    const client = createQueryClient()
    const invalidate = vi.spyOn(client, 'invalidateQueries')
    const stop = startLiveFeed(client, {
      WebSocketImpl: FakeSocket as unknown as typeof WebSocket,
      url: 'ws://x',
      delay: (n) => 100 * 2 ** n,
      ticket: () => Promise.resolve('t'),
    })
    await vi.advanceTimersByTimeAsync(0)
    FakeSocket.last().open()
    FakeSocket.last().drop()
    expect(liveState.get()).toBe('down')
    expect(FakeSocket.instances).toHaveLength(1)
    await vi.advanceTimersByTimeAsync(100)
    expect(FakeSocket.instances).toHaveLength(2)
    FakeSocket.last().drop()
    await vi.advanceTimersByTimeAsync(199)
    expect(FakeSocket.instances).toHaveLength(2) // second delay is 200ms
    await vi.advanceTimersByTimeAsync(1)
    FakeSocket.last().open()
    expect(liveState.get()).toBe('live')
    expect(invalidate).toHaveBeenCalledOnce()
    stop()
  })
})

describe('only changed rows re-render', () => {
  const columns: Column<TunnelMetrics>[] = [
    { id: 'name', header: 'Name', cell: (t) => t.Name },
    { id: 'rx', header: 'Received', numeric: true, cell: (t) => t.BytesReceived },
  ]

  function Live({ onRow }: { onRow: (k: string) => void }) {
    const { data } = useTunnelsQuery()
    return <DataList caption="t" columns={columns} rows={data ?? []} rowKey={(t) => t.Name} onRowRender={onRow} />
  }

  it('a WebSocket tick that changes one tunnel re-renders only its row', async () => {
    stubFetch({
      'GET /api/v1/preferences': () => json({ language: 'en', theme: '' }),
      'GET /api/v1/tunnels': () => json([tunnel('a', 'connected'), tunnel('b', 'connected')]),
    })
    const { client, wrapper } = setup()
    const rendered: string[] = []
    render(<Live onRow={(k) => rendered.push(k)} />, { wrapper })
    await screen.findByText('a')
    rendered.length = 0

    act(() => {
      client.setQueryData(keys.tunnels, [tunnel('a', 'connected', { BytesReceived: 99 }), tunnel('b', 'connected')])
    })
    expect(await screen.findByText('99')).toBeInTheDocument()
    expect(rendered).toEqual(['a'])
  })
})

describe('optimistic mutations', () => {
  it('shows the expected state at once and rolls back + toasts when the daemon refuses', async () => {
    let release: (r: Response) => void = () => {}
    stubFetch({
      'GET /api/v1/preferences': () => json({ language: 'en', theme: '' }),
      'GET /api/v1/tunnels': () => json([tunnel('work', 'disconnected')]),
      'POST /api/v1/tunnels/work/connect': () =>
        new Promise<Response>((r) => {
          release = r
        }),
    })
    const { client, wrapper } = setup()
    const { result } = renderHook(() => ({ q: useTunnelsQuery(), m: useTunnelAction() }), { wrapper })
    await waitFor(() => expect(result.current.q.data?.[0]?.Status).toBe('disconnected'))

    act(() => result.current.m.mutate({ name: 'work', action: 'connect' }))
    await waitFor(() => expect(client.getQueryData<TunnelMetrics[]>(keys.tunnels)?.[0]?.Status).toBe('connecting'))

    act(() =>
      release(
        new Response(JSON.stringify({ error: 'Profile not found', hint: 'What to do: check the name.' }), { status: 404 }),
      ),
    )
    await waitFor(() => expect(toasts.get()[0]?.message).toBe('Profile not found'))
    expect(toasts.get()[0]?.detail).toBe('What to do: check the name.')
    expect(toasts.get()[0]?.action?.label).toBe('Retry')
    expect(client.getQueryData<TunnelMetrics[]>(keys.tunnels)?.[0]?.Status).toBe('disconnected')
  })

  it('a switch flips at once and keeps the new value when accepted', async () => {
    const status: SecurityStatus = {
      kill_switch: false,
      kill_switch_enabled: false,
      kill_switch_state: 'disabled',
      dns_guard: false,
      dns_guard_enabled: false,
      dns_guard_state: 'disabled',
      public_ip_check_enabled: false,
      public_ip_check_host: '',
    }
    let enabled = false
    stubFetch({
      'GET /api/v1/preferences': () => json({ language: 'en', theme: '' }),
      'GET /api/v1/security/status': () => json({ ...status, kill_switch_enabled: enabled }),
      'POST /api/v1/security/killswitch': (init) => {
        enabled = (JSON.parse(String(init?.body)) as { enabled: boolean }).enabled
        return json({ status: 'updated', kill_switch_enabled: enabled })
      },
    })
    const { client, wrapper } = setup()
    const { result } = renderHook(() => useSecurityToggle('kill_switch'), { wrapper })
    client.setQueryData(keys.security, status)
    act(() => result.current.mutate(true))
    // Applied before the daemon answers (onMutate), then kept after it accepts.
    await waitFor(() => expect(client.getQueryData<SecurityStatus>(keys.security)?.kill_switch_enabled).toBe(true))
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    await waitFor(() => expect(client.getQueryData<SecurityStatus>(keys.security)?.kill_switch_enabled).toBe(true))
    expect(toasts.get()).toHaveLength(0)
  })
})

describe('helper', () => {
  it('maps the status to one UI state', () => {
    expect(helperUiState(undefined)).toBe('unknown')
    expect(helperUiState({ ...helperMissing, elevated: true })).toBe('not_needed')
    expect(helperUiState(helperReady)).toBe('ready')
    expect(helperUiState(helperMissing)).toBe('missing')
    expect(helperUiState({ ...helperMissing, installed: true, running: true })).toBe('outdated')
    expect(helperUiState({ ...helperMissing, outdated: true })).toBe('outdated')
  })

  it('toasts once when the helper becomes ready, even if installed elsewhere (CLI, classic panel)', async () => {
    let status = helperMissing
    stubFetch({
      'GET /api/v1/preferences': () => json({ language: 'en', theme: '' }),
      'GET /api/v1/helper/status': () => json(status),
    })
    const { client, wrapper } = setup()
    renderHook(() => useHelperReadyToast(), { wrapper })
    await waitFor(() => expect(client.getQueryData(keys.helper)).toEqual(helperMissing))
    status = helperReady
    await act(() => client.refetchQueries({ queryKey: keys.helper }))
    await waitFor(() => expect(toasts.get().map((t) => t.message)).toEqual(['Helper ready · no need to install it again']))
    await act(() => client.refetchQueries({ queryKey: keys.helper }))
    expect(toasts.get()).toHaveLength(1)
  })
})

describe('summarizeTunnels', () => {
  it('counts connected and picks the worst state (no response beats connected)', () => {
    expect(summarizeTunnels([])).toEqual({ total: 0, connected: 0, worst: 'none' })
    const s = summarizeTunnels([
      tunnel('a', 'connected'),
      tunnel('b', 'connected', { health: { state: 'no_response' } }),
      tunnel('c', 'disconnected'),
    ])
    expect(s).toEqual({ total: 3, connected: 1, worst: 'no_response' })
    expect(summarizeTunnels([tunnel('a', 'error'), tunnel('b', 'connecting')]).worst).toBe('error')
  })
})
