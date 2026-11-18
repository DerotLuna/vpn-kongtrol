import { describe, expect, it } from 'vitest'
import type { TunnelMetrics } from '../api/types'
import { aggregate, groupState, pendingDone, rowState } from './model'

function tn(name: string, Status: TunnelMetrics['Status'], extra: Partial<TunnelMetrics> = {}): TunnelMetrics {
  return {
    Name: name,
    Status,
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

describe('rowState — Four States (DESIGN.md)', () => {
  it('maps each state to its tone and word', () => {
    expect(rowState(tn('a', 'connected'))).toMatchObject({ tone: 'live', word: 'tray.status.connected', on: true })
    expect(rowState(tn('a', 'connecting'))).toMatchObject({ tone: 'steel', word: 'tray.status.connecting', on: true, busy: true })
    expect(rowState(tn('a', 'error'))).toMatchObject({ tone: 'danger', word: 'tray.status.error', on: false })
    expect(rowState(tn('a', 'disconnected'))).toMatchObject({ tone: 'idle', word: 'tray.status.disconnected', on: false })
    expect(rowState(tn('a', 'connected', { health: { state: 'no_response' } }))).toMatchObject({
      tone: 'caution',
      word: 'tray.status.noResponse',
    })
  })

  it('shows what the user asked for until the feed confirms it', () => {
    expect(rowState(tn('a', 'disconnected'), 'connect')).toMatchObject({ tone: 'steel', on: true, busy: true })
    expect(rowState(tn('a', 'connected'), 'disconnect')).toMatchObject({ word: 'tray.status.disconnecting', on: false })
    expect(pendingDone(tn('a', 'connected'), 'connect')).toBe(true)
    expect(pendingDone(tn('a', 'disconnected'), 'connect')).toBe(false)
    expect(pendingDone(tn('a', 'disconnected'), 'disconnect')).toBe(true)
    expect(pendingDone(undefined, 'connect')).toBe(true)
  })

  it('tells what to do while a provider app waits for the user', () => {
    const attach = { phase: 'waiting_user', app: 'FortiClient', since: '' }
    expect(rowState(tn('w', 'connecting', { attach })).note).toEqual({
      key: 'tray.note.waitingUser',
      vars: { app: 'FortiClient' },
    })
    const drop = { phase: 'waiting_disconnect', app: 'NordVPN', since: '' }
    expect(rowState(tn('w', 'connected', { attach: drop })).note?.key).toBe('tray.note.waitingDisconnect')
  })
})

describe('aggregate', () => {
  it('errors first, then work in progress, then the count', () => {
    const up = rowState(tn('a', 'connected'))
    const off = rowState(tn('b', 'disconnected'))
    expect(aggregate([up, off])).toEqual({ tone: 'live', key: 'tray.agg.some', vars: { connected: 1, total: 2 } })
    expect(aggregate([off])).toEqual({ tone: 'idle', key: 'tray.agg.none' })
    expect(aggregate([up, rowState(tn('c', 'connecting'))]).key).toBe('tray.agg.connecting')
    const silent = rowState(tn('s', 'connected', { health: { state: 'no_response' } }))
    expect(aggregate([up, silent])).toEqual({ tone: 'caution', key: 'tray.agg.noResponse', vars: { count: 1 } })
    expect(aggregate([silent, rowState(tn('d', 'error'))])).toEqual({ tone: 'danger', key: 'tray.agg.error', vars: { count: 1 } })
  })
})

describe('groupState', () => {
  it('counts only existing members; all up means the button disconnects', () => {
    const tunnels = [tn('a', 'connected'), tn('b', 'connected'), tn('c', 'disconnected')]
    expect(groupState({ name: 'g', profiles: ['a', 'b', 'gone'] }, tunnels)).toEqual({ connected: 2, total: 2, allUp: true })
    expect(groupState({ name: 'g', profiles: ['a', 'c'] }, tunnels)).toEqual({ connected: 1, total: 2, allUp: false })
    expect(groupState({ name: 'g', profiles: [] }, tunnels).allUp).toBe(false)
  })
})
