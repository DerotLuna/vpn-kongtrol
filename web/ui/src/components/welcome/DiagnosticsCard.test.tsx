// DiagnosticsCard on its own (Inicio no longer shows it; it moves to /settings).
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useConfigExport, useDoctor, useHelper } from '../../api/hooks'
import type { DoctorReport } from '../../api/types'
import { DataProvider } from '../../app/DataProvider'
import { PreferencesProvider } from '../../app/PreferencesProvider'
import { helperMissing, json, stubFetch, type Routes } from '../../test/fakes'
import { DiagnosticsCard } from './DiagnosticsCard'

class SilentSocket {
  onopen: (() => void) | null = null
  onmessage: ((e: MessageEvent) => void) | null = null
  onclose: (() => void) | null = null
  close() {}
}

function stubApi(routes: Routes) {
  const fetchMock = stubFetch(routes)
  vi.stubGlobal('WebSocket', SilentSocket)
  return fetchMock
}

function Card() {
  const doctor = useDoctor()
  const helper = useHelper()
  const exporter = useConfigExport()
  return <DiagnosticsCard doctor={doctor} helper={helper} exporter={exporter} />
}

function App() {
  return (
    <PreferencesProvider>
      <DataProvider>
        <MemoryRouter>
          <Card />
        </MemoryRouter>
      </DataProvider>
    </PreferencesProvider>
  )
}

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
