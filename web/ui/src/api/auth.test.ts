import { describe, expect, it, vi } from 'vitest'
import { api } from './client'
import { authReady, authState, startAuth, takeBootstrapNonce } from './auth'

describe('auth', () => {
  it('takes #k= out of the URL (and keeps other fragment params)', () => {
    window.history.replaceState(null, '', '/app/rules?x=1#k=abc123&tab=2')
    expect(takeBootstrapNonce()).toBe('abc123')
    expect(location.pathname + location.search + location.hash).toBe('/app/rules?x=1#tab=2')
    expect(takeBootstrapNonce()).toBe('')
  })

  it('trades the nonce for a session before any API request', async () => {
    window.history.replaceState(null, '', '/app/#k=n1')
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
