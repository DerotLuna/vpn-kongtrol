import { afterEach, describe, expect, it, vi } from 'vitest'
import { startSession, takeNonce } from './session'

afterEach(() => {
  vi.unstubAllGlobals()
  history.replaceState(null, '', '/')
})

describe('takeNonce', () => {
  it('returns the nonce and removes it from the address bar', () => {
    history.replaceState(null, '', '/app/tray#k=abc_-1')
    expect(takeNonce()).toBe('abc_-1')
    expect(location.hash).toBe('')
    expect(location.pathname).toBe('/app/tray')
  })

  it('returns empty without a nonce', () => {
    history.replaceState(null, '', '/app/')
    expect(takeNonce()).toBe('')
  })
})

describe('startSession', () => {
  it('posts the nonce once', async () => {
    history.replaceState(null, '', '/app/#k=n1')
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }))
    vi.stubGlobal('fetch', fetchMock)
    await startSession()
    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/session')
    expect(init.method).toBe('POST')
    expect(init.body).toBe(JSON.stringify({ nonce: 'n1' }))
  })

  it('does nothing without a nonce', async () => {
    history.replaceState(null, '', '/app/')
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    await startSession()
    expect(fetchMock).not.toHaveBeenCalled()
  })
})
