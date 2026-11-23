// A notice's offer buttons (ADR-0102): text, the API each one calls, and
// when they stop showing.
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, useLocation } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { DataProvider } from '../../app/DataProvider'
import { PreferencesProvider } from '../../app/PreferencesProvider'
import type { DaemonEvent } from '../../data/home'
import { offerLabel, offerOf, offerStillUseful } from '../../data/offers'
import { json, stubFetch, type Routes } from '../../test/fakes'
import { OfferActions } from './OfferActions'

class SilentSocket {
  onopen: (() => void) | null = null
  onmessage: ((e: MessageEvent) => void) | null = null
  onclose: (() => void) | null = null
  close() {}
}

let seq = 100
function ev(offer: string, extra: Partial<DaemonEvent> = {}): DaemonEvent {
  return { seq: seq++, time: '2026-11-23T10:00:00Z', kind: 'tunnel_watchdog', level: 'WARN', key: 'k', args: ['work'], profile: 'work', offer, ...extra }
}

function Where() {
  const loc = useLocation()
  return <output data-testid="where">{loc.search}</output>
}

function draw(event: DaemonEvent, routes: Routes, onDisconnect = vi.fn()) {
  const fetchMock = stubFetch({ 'GET /api/v1/preferences': () => json({ language: 'es', theme: '' }), ...routes })
  vi.stubGlobal('WebSocket', SilentSocket)
  render(
    <PreferencesProvider>
      <DataProvider>
        <MemoryRouter>
          <OfferActions event={event} onDisconnect={onDisconnect} />
          <Where />
        </MemoryRouter>
      </DataProvider>
    </PreferencesProvider>,
  )
  return { fetchMock, onDisconnect }
}

const ksOff = { kill_switch_enabled: false, kill_switch_state: 'disabled', dns_guard_enabled: false, dns_guard_state: 'disabled' }

beforeEach(() => {
  Object.defineProperty(navigator, 'languages', { value: ['es-AR'], configurable: true })
})

describe('OfferActions', () => {
  it('Activar protección turns the kill switch on through the security API', async () => {
    let enabled = false
    const { fetchMock } = draw(ev('enable_killswitch'), {
      'GET /api/v1/security/status': () => json({ ...ksOff, kill_switch_enabled: enabled }),
      'POST /api/v1/security/killswitch': () => {
        enabled = true
        return json({ status: 'updated', kill_switch_enabled: true })
      },
    })
    await userEvent.click(await screen.findByRole('button', { name: 'Activar protección' }))
    await waitFor(() => expect(fetchMock.mock.calls.some(([url]) => String(url) === '/api/v1/security/killswitch')).toBe(true))
    const post = fetchMock.mock.calls.find(([url, init]) => String(url) === '/api/v1/security/killswitch' && init?.method === 'POST')
    expect(JSON.parse(String(post?.[1]?.body))).toEqual({ enabled: true })
    await waitFor(() => expect(screen.queryByRole('button', { name: 'Activar protección' })).toBeNull())
  })

  it('No, gracias closes the offer without calling anything', async () => {
    const { fetchMock } = draw(ev('enable_killswitch'), { 'GET /api/v1/security/status': () => json(ksOff) })
    await userEvent.click(await screen.findByRole('button', { name: 'No, gracias' }))
    expect(screen.queryByRole('button', { name: 'Activar protección' })).toBeNull()
    expect(fetchMock.mock.calls.some(([, init]) => init?.method === 'POST' && init.body)).toBe(false)
  })

  it('Recuperar internet disconnects that profile as the user', async () => {
    const { onDisconnect } = draw(ev('release_block'), {
      'GET /api/v1/security/status': () => json({ ...ksOff, kill_switch_enabled: true, kill_switch_status: { state: 'active', blocking: ['work'], blocked_count: 3 } }),
    })
    await userEvent.click(await screen.findByRole('button', { name: 'Recuperar internet' }))
    expect(onDisconnect).toHaveBeenCalledWith('work')
    expect(screen.queryByRole('button', { name: 'No, gracias' })).toBeNull()
  })

  it('Agregar Proton opens the Add sheet with the provider chosen', async () => {
    draw(ev('add_provider', { args: ['work', 'protonvpn'] }), { 'GET /api/v1/security/status': () => json(ksOff) })
    await userEvent.click(await screen.findByRole('button', { name: 'Agregar Proton' }))
    expect(new URLSearchParams(screen.getByTestId('where').textContent ?? '')).toEqual(new URLSearchParams('add=vpn&type=protonvpn'))
  })
})

describe('offers model', () => {
  const t = (k: string, v?: Record<string, string | number>) => `${k}${v ? JSON.stringify(v) : ''}`
  it('reads only known offers; an unknown provider gives none', () => {
    expect(offerOf(ev('release_block', { profile: '' }))).toBeNull()
    expect(offerOf(ev('add_provider', { args: ['work', 'evil'] }))).toBeNull()
    expect(offerOf(ev('something_new'))).toBeNull()
    expect(offerOf(ev('add_provider', { args: ['work', 'nordvpn'] }))).toEqual({ kind: 'add_provider', profile: 'work', provider: 'nordvpn' })
    expect(offerLabel({ kind: 'add_provider', profile: 'w', provider: 'nordvpn' }, t as never)).toBe('offer.action.addProvider{"app":"NordVPN"}')
  })
  it('stops being useful once the kill switch is on or the profile is no longer blocked', () => {
    expect(offerStillUseful({ kind: 'enable_killswitch', profile: 'w' }, { kill_switch_enabled: true })).toBe(false)
    expect(offerStillUseful({ kind: 'enable_killswitch', profile: 'w' }, { kill_switch_enabled: false })).toBe(true)
    expect(offerStillUseful({ kind: 'release_block', profile: 'w' }, { kill_switch_enabled: true, kill_switch_status: { blocking: [] } })).toBe(false)
    expect(offerStillUseful({ kind: 'release_block', profile: 'w' }, null)).toBe(true)
  })
})
