import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { MemoryRouter } from 'react-router'
import { usePurgeData, useUpdates } from '../../api/lifecycle-hooks'
import { DataProvider } from '../../app/DataProvider'
import { PreferencesProvider } from '../../app/PreferencesProvider'
import { DataCard } from './DataCard'
import { StartupSection } from './StartupSection'
import { UpdatesCard } from './UpdatesCard'
import type { UpdateStatus } from '../../api/types'

function json(body: unknown, status = 200) {
  return Promise.resolve(new Response(JSON.stringify(body), { status }))
}

class SilentSocket {
  onopen: (() => void) | null = null
  onmessage: ((e: MessageEvent) => void) | null = null
  onclose: (() => void) | null = null
  close() {}
}

function stubApi(routes: Record<string, (init?: RequestInit) => Promise<Response>>) {
  const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
    const route = routes[`${init?.method ?? 'GET'} ${String(input)}`]
    return route ? route(init) : json({ error: 'not found' }, 404)
  })
  vi.stubGlobal('fetch', fetchMock)
  vi.stubGlobal('WebSocket', SilentSocket)
  return fetchMock
}

const base = {
  'GET /api/v1/preferences': () => json({ language: '', theme: '' }),
  'GET /api/v1/tunnels': () => json([]),
  'GET /api/v1/startup': () => json({ start_at_login: true, start_at_login_supported: true, restore_tunnels: true }),
}

// The cards on their own: Inicio no longer shows them (they move to /settings).
function Cards() {
  const updates = useUpdates()
  const purge = usePurgeData()
  return (
    <>
      <StartupSection />
      <UpdatesCard state={updates} />
      <DataCard state={purge} />
    </>
  )
}

function SettingsCards() {
  return (
    <PreferencesProvider>
      <DataProvider>
        <MemoryRouter>
          <Cards />
        </MemoryRouter>
      </DataProvider>
    </PreferencesProvider>
  )
}

const off: UpdateStatus ={ check_enabled: false, host: 'api.github.com', current: 'v1.0.0', newer: false }

describe('Settings', () => {
  beforeEach(() => {
    Object.defineProperty(navigator, 'languages', { value: ['es-AR'], configurable: true })
  })

  it('update notice is off by default, explains GitHub, and turns on', async () => {
    const fetchMock = stubApi({
      ...base,
      'GET /api/v1/updates': () => json(off),
      'POST /api/v1/updates': () =>
        json({ ...off, check_enabled: true, latest: 'v1.1.0', newer: true, checked_at: '2026-11-18T10:00:00Z', url: 'https://github.com/DerotLuna/vpn-kongtrol/releases/tag/v1.1.0' }),
    })
    render(<SettingsCards />)

    const card = (await screen.findByRole('heading', { name: 'Aviso de versión nueva' })).closest('section')!
    expect(await within(card).findByText('Apagado')).toBeInTheDocument()
    expect(within(card).getByText(/api\.github\.com una vez al día/)).toBeInTheDocument()

    await userEvent.click(within(card).getByRole('button', { name: 'Activar aviso' }))
    expect(await within(card).findByText(/Hay una versión nueva: v1\.1\.0 \(tienes v1\.0\.0\)/)).toBeInTheDocument()
    const post = fetchMock.mock.calls.find(([url, init]) => String(url) === '/api/v1/updates' && init?.method === 'POST')
    expect(JSON.parse(String(post?.[1]?.body))).toEqual({ enabled: true })
  })

  it('startup switches show the daemon state and send only their own field (ADR-0098)', async () => {
    let state = { start_at_login: true, start_at_login_supported: true, restore_tunnels: true }
    const fetchMock = stubApi({
      ...base,
      'GET /api/v1/updates': () => json(off),
      'GET /api/v1/startup': () => json(state),
      'PUT /api/v1/startup': (init) => {
        state = { ...state, ...(JSON.parse(String(init?.body)) as Partial<typeof state>) }
        return json(state)
      },
    })
    render(<SettingsCards />)

    const section = (await screen.findByRole('heading', { name: 'Al iniciar' })).closest('section')!
    const login = await within(section).findByRole('switch', { name: 'Abrir Kongtrol al iniciar sesión' })
    const restore = within(section).getByRole('switch', { name: 'Al iniciar, volver a conectar las VPN que tenías conectadas' })
    expect(login).toHaveAttribute('aria-checked', 'true')
    expect(restore).toHaveAttribute('aria-checked', 'true')

    await userEvent.click(restore)
    await vi.waitFor(() => expect(restore).toHaveAttribute('aria-checked', 'false'))
    const put = fetchMock.mock.calls.find(([url, init]) => String(url) === '/api/v1/startup' && init?.method === 'PUT')
    expect(JSON.parse(String(put?.[1]?.body))).toEqual({ restore_tunnels: false })
  })

  it('without a tray the sign-in switch is off, disabled and says what to do', async () => {
    stubApi({
      ...base,
      'GET /api/v1/updates': () => json(off),
      'GET /api/v1/startup': () => json({ start_at_login: false, start_at_login_supported: false, restore_tunnels: true }),
    })
    render(<SettingsCards />)
    const login = await screen.findByRole('switch', { name: 'Abrir Kongtrol al iniciar sesión' })
    expect(login).toBeDisabled()
    expect(screen.getByText(/no tiene el ícono de Kongtrol/)).toBeInTheDocument()
  })

  it('purge asks first, lists what is deleted, and cancel sends nothing', async () => {
    const fetchMock = stubApi({
      ...base,
      'GET /api/v1/updates': () => json(off),
      'POST /api/v1/data/purge': () => json({ status: 'purged', restarting: true }),
    })
    render(<SettingsCards />)

    await userEvent.click(await screen.findByRole('button', { name: 'Borrar todos mis datos' }))
    const dialog = screen.getByRole('alertdialog', { name: '¿Borrar todos tus datos?' })
    expect(within(dialog).getByText(/contraseñas guardadas en el llavero/)).toBeInTheDocument()
    expect(within(dialog).getByText(/Kongtrol sigue instalado/)).toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: 'Cancelar' })).toHaveFocus()

    await userEvent.click(within(dialog).getByRole('button', { name: 'Cancelar' }))
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    expect(fetchMock.mock.calls.some(([url]) => String(url) === '/api/v1/data/purge')).toBe(false)

    await userEvent.click(screen.getByRole('button', { name: 'Borrar todos mis datos' }))
    await userEvent.click(screen.getByRole('button', { name: 'Sí, borrar todo' }))
    expect(await screen.findByText(/tus datos se borraron/)).toBeInTheDocument()
    const post = fetchMock.mock.calls.find(([url]) => String(url) === '/api/v1/data/purge')
    expect(JSON.parse(String(post?.[1]?.body))).toEqual({ confirm: true })
  })
})
