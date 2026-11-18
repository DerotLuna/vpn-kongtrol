import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { App } from '../App'
import type { DoctorReport, HelperStatus, TunnelMetrics } from '../api/types'
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
    expect(await screen.findByText('Helper ready · no need to install it again.')).toBeInTheDocument()
    const install = fetchMock.mock.calls.find(([, init]) => init?.method === 'POST')
    expect(new Headers(install?.[1]?.headers).get('X-Kongtrol-Lang')).toBe('en')
    // No VPNs yet: "Add VPN" is the primary action.
    expect(screen.getByRole('link', { name: /Add VPN/ })).toHaveClass('primary')
  })

  it('offers to update an outdated helper instead of installing it', async () => {
    stubApi({
      'GET /api/v1/preferences': () => json({ language: '', theme: '' }),
      'GET /api/v1/helper/status': () =>
        json({ ...helperMissing, installed: true, running: true, helper_version: 'v0.5.0', outdated: true }),
      'GET /api/v1/tunnels': () => json([]),
    })
    render(<App />)

    expect(await screen.findByRole('button', { name: 'Actualizar ayudante' })).toBeEnabled()
    expect(screen.getByText('Desactualizado')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Instalar ayudante' })).not.toBeInTheDocument()
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

    // caution (orange), not the red of an error nor the green of "connected"
    expect(await screen.findByText('Sin respuesta')).toHaveClass('pill', 'caution')
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

const doctorReport = (helperOK: boolean): DoctorReport => ({
  failures: 1,
  warnings: helperOK ? 0 : 2,
  checks: [
    { section: 'Configuración', label: 'archivo de config', status: 'ok', detail: '/home/ana/.kongtrol/kongtrol.yaml' },
    ...(helperOK
      ? [{ section: 'Permisos', label: 'ayudante', status: 'ok' as const, detail: 'instalado y respondiendo (versión v1.2.3)' }]
      : [
          {
            section: 'Permisos',
            label: 'ayudante',
            status: 'warn' as const,
            detail: 'no instalado',
            hint: 'Instala el ayudante (una vez): botón Reparar en el panel, o kongtrol service install.',
            fix: 'helper_install' as const,
          },
          {
            section: 'Permisos',
            label: 'kill switch',
            status: 'warn' as const,
            detail: 'necesita el ayudante de Kongtrol (o permisos de administrador)',
            hint: 'Instala el ayudante (una vez): botón Reparar en el panel, o kongtrol service install.',
            fix: 'helper_install' as const,
          },
        ]),
    {
      section: 'Credenciales en keychain',
      label: 'office: password',
      status: 'fail',
      detail: 'no está en el llavero del sistema',
      hint: 'Vuelve a guardar la credencial: edita el perfil en el panel o ejecuta kongtrol init.',
    },
  ],
})

describe('DiagnosticsCard', () => {
  beforeEach(() => {
    Object.defineProperty(navigator, 'languages', { value: ['es-AR'], configurable: true })
  })

  it('runs only on demand, shows state + what to do, and offers one Reparar for the helper', async () => {
    let installed = false
    const fetchMock = stubApi({
      'GET /api/v1/preferences': () => json({ language: '', theme: '' }),
      'GET /api/v1/helper/status': () => json(helperMissing),
      'GET /api/v1/tunnels': () => json([]),
      'GET /api/v1/doctor': () => json(doctorReport(installed)),
      'POST /api/v1/helper/install': () => {
        installed = true
        return json({ ...helperMissing, installed: true, running: true, available: true, needs_install: false })
      },
    })
    render(<App />)

    const card = (await screen.findByRole('heading', { name: 'Diagnóstico' })).closest('section')!
    expect(within(card).getByText('Pulsa «Revisar ahora»; tarda unos segundos.')).toBeInTheDocument()
    expect(fetchMock.mock.calls.some(([url]) => String(url) === '/api/v1/doctor')).toBe(false)

    await userEvent.click(within(card).getByRole('button', { name: 'Revisar ahora' }))
    expect(await within(card).findByText('1 con error · 2 con aviso')).toBeInTheDocument()
    expect(within(card).getByText('Con errores')).toHaveClass('pill', 'bad')
    expect(within(card).getAllByText('Aviso')[0]).toHaveClass('pill', 'caution')
    expect(within(card).getByText('Error')).toHaveClass('pill', 'bad')
    expect(within(card).getByText(/Vuelve a guardar la credencial/)).toBeInTheDocument()
    expect(within(card).getAllByText('Qué hacer:')).toHaveLength(3)
    // Two checks carry the helper fix; one button. The credential failure has none.
    const fix = within(card).getAllByRole('button', { name: 'Reparar' })
    expect(fix).toHaveLength(1)
    expect(within(card).getByText('Ver las 1 comprobaciones correctas')).toBeInTheDocument()

    await userEvent.click(fix[0]!)
    expect(await within(card).findByText('Listo: el ayudante quedó instalado. Revisamos otra vez.')).toBeInTheDocument()
    expect(fetchMock.mock.calls.filter(([url]) => String(url) === '/api/v1/doctor')).toHaveLength(2)
    expect(within(card).queryByRole('button', { name: 'Reparar' })).toBeNull()
  })

  it('exports the configuration as a file, and explains when there is none', async () => {
    let hasConfig = false
    stubApi({
      'GET /api/v1/preferences': () => json({ language: 'en', theme: '' }),
      'GET /api/v1/helper/status': () => json(helperMissing),
      'GET /api/v1/tunnels': () => json([]),
      'GET /api/v1/config/export': () =>
        hasConfig
          ? Promise.resolve(new Response('vpns: {}\n', { status: 200, headers: { 'Content-Type': 'application/yaml' } }))
          : json(
              {
                error: 'There is no configuration to export yet.',
                hint: 'Create your first VPN from the dashboard or with kongtrol init.',
                error_key: 'export.err.no_config',
              },
              404,
            ),
    })
    const createObjectURL = vi.fn(() => 'blob:x')
    vi.stubGlobal('URL', Object.assign(URL, { createObjectURL, revokeObjectURL: vi.fn() }))
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    render(<App />)

    const exportBtn = await screen.findByRole('button', { name: 'Export configuration' })
    await userEvent.click(exportBtn)
    expect(await screen.findByText('There is no configuration to export yet.')).toBeInTheDocument()
    expect(screen.getByText('Create your first VPN from the dashboard or with kongtrol init.')).toBeInTheDocument()
    expect(click).not.toHaveBeenCalled()

    hasConfig = true
    await userEvent.click(exportBtn)
    expect(await screen.findByText('Done: kongtrol-export.yaml was downloaded.')).toBeInTheDocument()
    expect(createObjectURL).toHaveBeenCalledOnce()
    expect(click).toHaveBeenCalledOnce()
    click.mockRestore()
  })
})
