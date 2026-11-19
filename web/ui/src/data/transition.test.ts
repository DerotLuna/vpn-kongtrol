// A tunnel action stays in progress until the daemon reports its end (transition.ts).
import { afterEach, describe, expect, it, vi } from 'vitest'
import { tunnel } from '../test/fakes'
import { keys } from './keys'
import { createQueryClient } from './queryClient'
import { SETTLE_TIMEOUT_MS, pendingActions, resetTransitions, transitionSettled, watchTransition } from './transition'

const fresh = { connectedAt: '', sawProgress: false }

describe('transitionSettled', () => {
  it('connect ends on connected, an error, a failed attempt or a provider app waiting for the user', () => {
    expect(transitionSettled(tunnel('a', 'connected'), 'connect', fresh)).toBe(true)
    expect(transitionSettled(tunnel('a', 'error'), 'connect', fresh)).toBe(true)
    expect(transitionSettled(tunnel('a', 'disconnected', { last_error_key: 'vpn.err.timeout' }), 'connect', fresh)).toBe(true)
    expect(transitionSettled(tunnel('a', 'connecting', { attach: { phase: 'waiting_user', app: 'NordVPN', since: '' } }), 'connect', fresh)).toBe(true)
    expect(transitionSettled(tunnel('a', 'connecting'), 'connect', fresh)).toBe(false)
    // A stale "disconnected" right after the 202 is not the end…
    expect(transitionSettled(tunnel('a', 'disconnected'), 'connect', fresh)).toBe(false)
    // …but after it was seen connecting, it is (cancelled elsewhere).
    expect(transitionSettled(tunnel('a', 'disconnected'), 'connect', { ...fresh, sawProgress: true })).toBe(true)
    expect(transitionSettled(undefined, 'connect', fresh)).toBe(true)
  })

  it('disconnect / cancel end when it is down; reload when it is back with a new ConnectedAt', () => {
    expect(transitionSettled(tunnel('a', 'connected'), 'disconnect', fresh)).toBe(false)
    expect(transitionSettled(tunnel('a', 'disconnected'), 'disconnect', fresh)).toBe(true)
    expect(transitionSettled(tunnel('a', 'connected', { attach: { phase: 'waiting_disconnect', app: 'NordVPN', since: '' } }), 'disconnect', fresh)).toBe(true)
    expect(transitionSettled(tunnel('a', 'disconnected'), 'cancel', fresh)).toBe(true)
    const base = { connectedAt: 't1', sawProgress: false }
    expect(transitionSettled(tunnel('a', 'connected', { ConnectedAt: 't1' }), 'reload', base)).toBe(false)
    expect(transitionSettled(tunnel('a', 'connected', { ConnectedAt: 't2' }), 'reload', base)).toBe(true)
    expect(transitionSettled(tunnel('a', 'connected', { ConnectedAt: 't1' }), 'reload', { ...base, sawProgress: true })).toBe(true)
  })
})

describe('watchTransition', () => {
  afterEach(() => {
    resetTransitions()
    vi.useRealTimers()
  })

  it('follows the tunnels cache (live frames or re-reads) and times out once with the callback', () => {
    vi.useFakeTimers()
    const client = createQueryClient()
    client.setQueryData(keys.tunnels, [tunnel('a', 'disconnected'), tunnel('b', 'disconnected')])
    const onTimeout = vi.fn()
    watchTransition(client, 'a', 'connect', { onTimeout })
    watchTransition(client, 'b', 'connect', { onTimeout })
    expect(pendingActions.get()).toEqual({ a: 'connect', b: 'connect' })

    client.setQueryData(keys.tunnels, [tunnel('a', 'connecting'), tunnel('b', 'connecting')])
    expect(pendingActions.get()).toEqual({ a: 'connect', b: 'connect' })
    client.setQueryData(keys.tunnels, [tunnel('a', 'connected'), tunnel('b', 'connecting')])
    expect(pendingActions.get()).toEqual({ b: 'connect' })

    vi.advanceTimersByTime(SETTLE_TIMEOUT_MS.connect)
    expect(pendingActions.get()).toEqual({})
    expect(onTimeout).toHaveBeenCalledTimes(1)
    expect(onTimeout).toHaveBeenCalledWith('b', 'connect')
  })

  it('a newer action on the same tunnel (Cancelar during a connect) replaces the watch', () => {
    vi.useFakeTimers()
    const client = createQueryClient()
    client.setQueryData(keys.tunnels, [tunnel('a', 'connecting')])
    const onTimeout = vi.fn()
    watchTransition(client, 'a', 'connect', { onTimeout })
    watchTransition(client, 'a', 'cancel')
    expect(pendingActions.get()).toEqual({ a: 'cancel' })
    client.setQueryData(keys.tunnels, [tunnel('a', 'disconnected')])
    expect(pendingActions.get()).toEqual({})
    vi.advanceTimersByTime(SETTLE_TIMEOUT_MS.connect + 1)
    expect(onTimeout).not.toHaveBeenCalled()
  })
})
