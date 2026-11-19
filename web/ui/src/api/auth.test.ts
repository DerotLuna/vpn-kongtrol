import { describe, expect, it, vi } from 'vitest'
import { api } from './client'
import {
  authHeaders,
  authReady,
  authState,
  liveTicket,
  resetSessionForTests,
  startAuth,
  takeBootstrapNonce,
} from './auth'

describe('auth', () => {
  it('takes #k= out of the URL (and keeps other fragment params)', () => {
    window.history.replaceState(null, '', '/rules?x=1#k=abc123&tab=2')
    expect(takeBootstrapNonce()).toBe('abc123')
    expect(location.pathname + location.search + location.hash).toBe('/rules?x=1#tab=2')
    expect(takeBootstrapNonce()).toBe('')
  })

  it('trades the nonce for a session before any API request', async () => {
    window.history.replaceState(null, '', '/#k=n1')
    const order: string[] = []
    const fetchMock = vi.fn((input: RequestInfo | URL, _init?: RequestInit) => {
      order.push(String(input))
      return Promise.resolve(new Response('{}', { status: 200 }))
    })
    vi.stubGlobal('fetch', fetchMock)
    void startAuth()
    await api.preferences()
    await authReady()
    expect(order).toEqual(['/api/v1/session', '/api/v1/preferences'])
    expect(JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body))).toEqual({ nonce: 'n1' })
    expect(location.hash).toBe('')
  })

  it('keeps the page token in sessionStorage and sends it on every request (ADR-0062)', async () => {
    resetSessionForTests()
    window.history.replaceState(null, '', '/#k=n2')
    const headers: Array<string | null> = []
    vi.stubGlobal(
      'fetch',
      vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
        if (String(input) === '/api/v1/session') {
          return Promise.resolve(new Response(JSON.stringify({ session_token: 'page-tok' }), { status: 200 }))
        }
        headers.push(new Headers(init?.headers).get('X-Kongtrol-Session'))
        return Promise.resolve(new Response('{}', { status: 200 }))
      }),
    )
    await startAuth()
    await api.preferences()
    expect(headers).toEqual(['page-tok'])
    expect(sessionStorage.getItem('kongtrol.session')).toBe('page-tok')
    expect(authHeaders()).toEqual({ 'X-Kongtrol-Session': 'page-tok' })
    // Never in localStorage (shared by every tab, kept after closing).
    expect(Object.values({ ...localStorage })).not.toContain('page-tok')
  })

  it('a page opened without a nonce sends no session header and gets the "open from Kongtrol" state', async () => {
    resetSessionForTests()
    window.history.replaceState(null, '', '/')
    const fetchMock = vi.fn((_input: RequestInfo | URL, _init?: RequestInit) =>
      Promise.resolve(new Response('unauthorized', { status: 401 })),
    )
    vi.stubGlobal('fetch', fetchMock)
    await startAuth()
    await expect(api.preferences()).rejects.toMatchObject({ kind: 'unauthorized' })
    expect(new Headers(fetchMock.mock.calls[0]?.[1]?.headers).has('X-Kongtrol-Session')).toBe(false)
    expect(authState.get()).toBe('unauthorized')
  })

  it('asks for a one-time WebSocket ticket with the session header', async () => {
    resetSessionForTests()
    window.history.replaceState(null, '', '/#k=n3')
    const fetchMock = vi.fn((input: RequestInfo | URL, _init?: RequestInit) =>
      Promise.resolve(
        String(input) === '/api/v1/session'
          ? new Response(JSON.stringify({ session_token: 'tok3' }), { status: 200 })
          : new Response(JSON.stringify({ ticket: 'tk', expires_in: 30 }), { status: 200 }),
      ),
    )
    await startAuth(fetchMock)
    expect(await liveTicket(fetchMock)).toBe('tk')
    const [url, init] = fetchMock.mock.calls[1]!
    expect(url).toBe('/api/v1/ws-ticket')
    expect(init?.method).toBe('POST')
    expect(new Headers(init?.headers).get('X-Kongtrol-Session')).toBe('tok3')

    const denied = vi.fn(() => Promise.resolve(new Response('unauthorized', { status: 401 })))
    await expect(liveTicket(denied)).rejects.toThrow()
    expect(authState.get()).toBe('unauthorized')
  })

  it('a 401 marks the page unauthorized; a later answer clears it', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve(new Response('unauthorized', { status: 401 }))))
    await expect(api.preferences()).rejects.toMatchObject({ kind: 'unauthorized' })
    expect(authState.get()).toBe('unauthorized')
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve(new Response('{}', { status: 200 }))))
    await api.preferences()
    expect(authState.get()).toBe('ok')
  })

  it('a translated 403 (declined prompt) is an ordinary failure, not "unauthorized"', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(new Response(JSON.stringify({ error: 'x', error_key: 'service.elevation.cancelled' }), { status: 403 }))),
    )
    await expect(api.installHelper()).rejects.toMatchObject({ kind: 'server', status: 403 })
    expect(authState.get()).toBe('ok')
  })
})
