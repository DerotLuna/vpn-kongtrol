import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { App } from '../App'
import type { HelperStatus, TunnelMetrics } from '../api/types'
import { GUIDE_URL } from '../links'

const helperMissing: HelperStatus = {
  installed: false,
  running: false,
  available: false,
  elevated: false,
  helper_version: '',
  cli_version: 'v1.2.3',
  needs_install: true,
}

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

function json(body: unknown, status = 200) {
  return Promise.resolve(new Response(JSON.stringify(body), { status }))
}

// WebSocket that never opens: the page stays on the REST snapshot.
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

describe('WelcomePage', () => {
  beforeEach(() => {
    Object.defineProperty(navigator, 'languages', { value: ['es-AR'], configurable: true })
  })

  it('shows helper status, tunnel summary and guide links in Spanish', async () => {
    stubApi({
      'GET /api/v1/preferences': () => json({ language: '', theme: '' }),
      'GET /api/v1/helper/status': () => json(helperMissing),
      'GET /api/v1/tunnels': () => json([tunnel('work', 'connected'), tunnel('home', 'disconnected')]),
    })
    render(<App />)

    expect(await screen.findByRole('heading', { level: 1, name: 'Bienvenido a Kongtrol' })).toBeInTheDocument()
    expect(await screen.findByText('1 de 2 conectadas')).toBeInTheDocument()
    expect(await screen.findByRole('button', { name: 'Instalar ayudante' })).toBeEnabled()
    expect(await screen.findByText('Kongtrol v1.2.3')).toBeInTheDocument()

    const guideLinks = screen
      .getAllByRole('link')
      .filter((a) => a.getAttribute('href')?.startsWith(GUIDE_URL))
    expect(guideLinks.length).toBeGreaterThanOrEqual(4)
    for (const a of guideLinks) {
      expect(a).toHaveAttribute('target', '_blank')
      expect(a).toHaveAttribute('rel', 'noopener noreferrer')
    }
    expect(screen.getByRole('link', { name: /Agregar VPN/ })).toHaveAttribute('href', '/studio.html')
  })

  it('installs the helper with one click and confirms', async () => {
    const fetchMock = stubApi({
      'GET /api/v1/preferences': () => json({ language: 'en', theme: '' }),
      'GET /api/v1/helper/status': () => json(helperMissing),
      'GET /api/v1/tunnels': () => json([]),
      'POST /api/v1/helper/install': () =>
        json({ ...helperMissing, installed: true, running: true, available: true, needs_install: false }),
    })
    render(<App />)

    await userEvent.click(await screen.findByRole('button', { name: 'Install helper' }))
    expect(await screen.findByText('Done: the helper is installed.')).toBeInTheDocument()
    const install = fetchMock.mock.calls.find(([, init]) => init?.method === 'POST')
    expect(new Headers(install?.[1]?.headers).get('X-Kongtrol-Lang')).toBe('en')
    // No VPNs yet: "Add VPN" is the primary action.
    expect(screen.getByRole('link', { name: /Add VPN/ })).toHaveClass('primary')
  })

  // Issue #33: a WireGuard tunnel whose server stopped answering is not
  // shown as connected, and an unverifiable one says what to do.
  it('flags tunnels whose server does not answer', async () => {
    stubApi({
      'GET /api/v1/preferences': () => json({ language: '', theme: '' }),
      'GET /api/v1/helper/status': () => json({ ...helperMissing, installed: true, running: true, available: true, needs_install: false }),
      'GET /api/v1/tunnels': () =>
        json([
          { ...tunnel('brainbox', 'error'), health: { state: 'no_response' } },
          { ...tunnel('home', 'connected'), health: { state: 'unknown', reason_key: 'service.error.outdated' } },
        ]),
    })
    render(<App />)

    expect(await screen.findByText('Sin respuesta')).toBeInTheDocument()
    expect(screen.getByText(/Sin respuesta del servidor · Kongtrol reintenta solo/)).toBeInTheDocument()
    expect(screen.getByText(/el ayudante está desactualizado/)).toBeInTheDocument()
  })

  it('explains what to do when Kongtrol is not running', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.reject(new TypeError('Failed to fetch'))))
    vi.stubGlobal('WebSocket', SilentSocket)
    render(<App />)

    const alerts = await screen.findAllByRole('alert')
    expect(within(alerts[0]!).getByText('No se pudo hablar con Kongtrol.')).toBeInTheDocument()
    expect(within(alerts[0]!).getByText(/Qué hacer:/)).toBeInTheDocument()
  })
  it('keeps the public IP lookup off until the user turns it on', async () => {
    let enabled = false
    const status = () =>
      json({
        public_ip_check_enabled: enabled,
        public_ip_check_host: 'api.ipify.org',
        leak_check: enabled ? { state: 'clean', has_leak: false, public_ip: '203.0.113.7', checked_at: '' } : null,
      })
    const fetchMock = stubApi({
      'GET /api/v1/preferences': () => json({ language: 'es', theme: '' }),
      'GET /api/v1/helper/status': () => json(helperMissing),
      'GET /api/v1/tunnels': () => json([]),
      'GET /api/v1/security/status': status,
      'POST /api/v1/security/publicip': () => {
        enabled = true
        return status()
      },
    })
    render(<App />)

    expect(await screen.findByText('Actívalo para ver tu IP pública.')).toBeInTheDocument()
    expect(screen.getByText('—')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Revisar' })).toBeNull()
    expect(fetchMock.mock.calls.some(([url]) => String(url).includes('/publicip/check'))).toBe(false)

    await userEvent.click(screen.getByRole('button', { name: 'Activar' }))
    expect(await screen.findByText('Usa un servicio externo (api.ipify.org) que ve tu IP pública')).toBeInTheDocument()
    expect(await screen.findByText('203.0.113.7')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Revisar' })).toBeEnabled()
  })
})
