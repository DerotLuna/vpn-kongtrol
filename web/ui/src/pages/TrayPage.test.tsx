import { render, screen, within, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { App } from '../App'
import { authState } from '../api/auth'
import type { SecurityStatus, TunnelMetrics } from '../api/types'

function tunnel(name: string, status: TunnelMetrics['Status']): TunnelMetrics {
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
  }
}

const security: SecurityStatus = {
  kill_switch_enabled: true,
  kill_switch_state: 'armed',
  dns_guard_enabled: true,
  dns_guard_state: 'active',
  public_ip_check_enabled: false,
  public_ip_check_host: 'api.ipify.org',
}

function json(body: unknown, status = 200) {
  return Promise.resolve(new Response(JSON.stringify(body), { status }))
}

class SilentSocket {
  onopen: (() => void) | null = null
  onmessage: ((e: MessageEvent) => void) | null = null
  onclose: (() => void) | null = null
  close() {}
}

type Routes = Record<string, () => Promise<Response>>

function stubApi(routes: Routes) {
  const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
    const key = `${init?.method ?? 'GET'} ${String(input)}`
    const route = routes[key]
    return route ? route() : json({ error: 'not found' }, 404)
  })
  vi.stubGlobal('fetch', fetchMock)
  vi.stubGlobal('WebSocket', SilentSocket)
  return fetchMock
}

function baseRoutes(tunnels: TunnelMetrics[]): Routes {
  return {
    'GET /api/v1/preferences': () => json({ language: '', theme: '' }),
    'GET /api/v1/tunnels': () => json(tunnels),
    'GET /api/v1/groups': () => json([{ name: 'trabajo', profiles: ['oficina', 'cliente'] }]),
    'GET /api/v1/security/status': () => json(security),
    'GET /api/v1/version': () => json({ version: 'v0.7.0' }),
  }
}

function setLang(lang: string) {
  Object.defineProperty(navigator, 'languages', { value: [lang], configurable: true })
}

describe('TrayPage (/tray)', () => {
  beforeEach(() => {
    window.history.pushState({}, '', `${import.meta.env.BASE_URL}tray`)
    localStorage.clear()
    setLang('es-AR')
  })
  afterEach(() => {
    delete window.kongtrolTray
    vi.unstubAllGlobals()
    authState.set('ok')
  })

  it('shows each VPN with its state word, the aggregate, protection and version', async () => {
    stubApi(baseRoutes([tunnel('oficina', 'connected'), tunnel('cliente', 'disconnected'), tunnel('casa', 'error')]))
    render(<App />)

    expect(await screen.findByText('oficina')).toBeInTheDocument()
    expect(screen.getByText('Conectada')).toBeInTheDocument()
    expect(screen.getByText('Desconectada')).toBeInTheDocument()
    expect(screen.getByText('Con error')).toBeInTheDocument()
    expect(screen.getByRole('status')).toHaveTextContent('1 con error')

    const sw = screen.getByRole('switch', { name: 'Desconectar oficina' })
    expect(sw).toHaveAttribute('aria-checked', 'true')
    expect(screen.getByRole('switch', { name: 'Conectar cliente' })).toHaveAttribute('aria-checked', 'false')

    expect(await screen.findByText('Armado')).toBeInTheDocument()
    expect(screen.getByText('Activa')).toBeInTheDocument()
    expect(await screen.findByText('Kongtrol v0.7.0')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Conectar el grupo trabajo' })).toBeInTheDocument()
  })

  it('connects from the switch and shows "Conectando…" until the feed confirms', async () => {
    const fetchMock = stubApi({
      ...baseRoutes([tunnel('oficina', 'disconnected')]),
      'POST /api/v1/tunnels/oficina/connect': () => json({ status: 'connecting', tunnel: 'oficina' }, 202),
    })
    render(<App />)
    const sw = await screen.findByRole('switch', { name: 'Conectar oficina' })
    await userEvent.click(sw)

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/tunnels/oficina/connect', expect.objectContaining({ method: 'POST' }))
    const row = sw.closest('li') as HTMLElement
    expect(within(row).getByText('Conectando…')).toBeInTheDocument()
    expect(sw).toHaveAttribute('aria-checked', 'true')
    expect(sw).toHaveAttribute('aria-busy', 'true')
  })

  it('says what to do when a connect fails', async () => {
    stubApi({
      ...baseRoutes([tunnel('oficina', 'disconnected')]),
      'POST /api/v1/tunnels/oficina/connect': () => json({ error: 'boom' }, 500),
    })
    render(<App />)
    await userEvent.click(await screen.findByRole('switch', { name: 'Conectar oficina' }))
    expect(await screen.findByText('No se pudo. Inténtalo otra vez o abre el panel.')).toBeInTheDocument()
  })

  it('shows a what-to-do line when the daemon is unreachable', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.reject(new TypeError('Failed to fetch'))))
    vi.stubGlobal('WebSocket', SilentSocket)
    render(<App />)
    expect(await screen.findByText('No se pudo hablar con Kongtrol.')).toBeInTheDocument()
    expect(screen.getByRole('alert')).toHaveTextContent('Qué hacer')
  })

  it('in a normal tab: "Abrir panel" is a link and there is no Salir', async () => {
    stubApi(baseRoutes([]))
    render(<App />)
    expect(await screen.findByText('Todavía no tienes VPN. Agrega la primera en un minuto.')).toBeInTheDocument()
    // with no VPN the one primary action is "Agregar VPN"; "Abrir panel" steps back
    expect(screen.getByRole('button', { name: 'Agregar VPN' })).toHaveClass('primary')
    expect(screen.getByRole('link', { name: 'Abrir panel' })).not.toHaveClass('primary')
    expect(screen.getByRole('link', { name: 'Abrir panel' })).toHaveAttribute('href', '/')
    expect(screen.queryByRole('button', { name: 'Salir' })).toBeNull()
  })

  it('after a daemon restart (401) asks the tray once to reload, and hides stale rows', async () => {
    const post = vi.fn()
    window.kongtrolTray = { post }
    // every request: the new daemon does not know this page's session
    const routes = Object.fromEntries(Object.keys(baseRoutes([])).map((k) => [k, () => json({ error: 'unauthorized' }, 401)]))
    stubApi(routes)
    render(<App />)
    expect(await screen.findByText(/Kongtrol se reinició/)).toBeInTheDocument()
    await waitFor(() => expect(post).toHaveBeenCalledWith('reauth'))
    expect(post.mock.calls.filter(([m]) => m === 'reauth')).toHaveLength(1)
    expect(screen.queryByRole('switch')).toBeNull()
  })

  it('inside the flyout: talks to the tray only through the bridge', async () => {
    const post = vi.fn()
    window.kongtrolTray = { post }
    stubApi(baseRoutes([tunnel('oficina', 'connected')]))
    render(<App />)
    await screen.findByText('oficina')
    // ready goes out on the next animation frame (announceReady): wait for it.
    await waitFor(() => expect(post).toHaveBeenCalledWith('ready'))

    await userEvent.click(screen.getByRole('button', { name: 'Abrir panel' }))
    expect(post).toHaveBeenCalledWith('open-dashboard')

    await userEvent.click(screen.getByRole('button', { name: 'Salir' }))
    const quitTray = screen.getByRole('button', { name: /Cerrar solo el ícono/ })
    expect(quitTray).toHaveFocus()
    await userEvent.keyboard('{Escape}')
    expect(screen.queryByRole('button', { name: /Cerrar solo el ícono/ })).toBeNull()
    expect(post).not.toHaveBeenCalledWith('close')

    await userEvent.keyboard('{Escape}')
    expect(post).toHaveBeenCalledWith('close')

    await userEvent.click(screen.getByRole('button', { name: 'Salir' }))
    await userEvent.click(screen.getByRole('button', { name: 'Salir y desconectar todo' }))
    expect(post).toHaveBeenCalledWith('quit-all')
  })

  it('renders in English', async () => {
    setLang('en-US')
    stubApi(baseRoutes([tunnel('office', 'connecting')]))
    render(<App />)
    expect(await screen.findByText('office')).toBeInTheDocument()
    expect(screen.getAllByText('Connecting…').length).toBeGreaterThan(0)
    expect(screen.getByRole('link', { name: 'Open panel' })).toBeInTheDocument()
    expect(await screen.findByText('Kill switch')).toBeInTheDocument()
  })
})
