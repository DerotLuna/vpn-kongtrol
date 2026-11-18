import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { App } from '../App'
import { parseDuration, validateDuration, validateMaxSize } from '../data/settings'
import { FakeSocket, helperReady, json, stubFetch } from '../test/fakes'

beforeEach(() => {
  Object.defineProperty(navigator, 'languages', { value: ['es-AR'], configurable: true })
  FakeSocket.reset()
  vi.stubGlobal('WebSocket', FakeSocket)
})

const settings = {
  dashboard_bind: '127.0.0.1',
  dashboard_port: 7777,
  health_check_interval: '30s',
  health_check_timeout: '10s',
  audit_log_path: '',
  audit_log_max_size_mb: 100,
  audit_log_sign: false,
}

const base = {
  'GET /api/v1/preferences': () => json({ language: '', theme: '' }),
  'GET /api/v1/tunnels': () => json([]),
  'GET /api/v1/version': () => json({ version: 'v1.4.0' }),
  'GET /api/v1/helper/status': () => json(helperReady),
  'GET /api/v1/settings': () => json(settings),
  'GET /api/v1/setup': () =>
    json({ setup_mode: false, needs_setup: false, restart_pending: false, config_path: 'C:\\Users\\ana\\.kongtrol\\kongtrol.yaml' }),
  'GET /api/v1/updates': () => json({ check_enabled: false, host: 'api.github.com', current: 'v1.4.0', newer: false }),
  'GET /api/v1/security/status': () =>
    json({ public_ip_check_enabled: false, public_ip_check_host: 'api.ipify.org', kill_switch_enabled: false, kill_switch_state: 'disabled', dns_guard_enabled: false, dns_guard_state: 'disabled' }),
}

function openSettings() {
  window.history.replaceState(null, '', '/settings')
  render(<App />)
}

describe('Ajustes: cambios sin guardar', () => {
  it('asks before leaving with unsaved edits; Seguir editando stays, Descartar leaves', async () => {
    stubFetch(base)
    openSettings()
    const interval = await screen.findByLabelText('Revisar cada')
    await userEvent.clear(interval)
    await userEvent.type(interval, '45s')
    const nav = screen.getAllByRole('navigation', { name: 'Secciones' })[0]!
    await userEvent.click(within(nav).getByRole('link', { name: 'Inicio' }))
    const dialog = await screen.findByRole('alertdialog', { name: 'Tienes cambios sin guardar' })
    await userEvent.click(within(dialog).getByRole('button', { name: 'Seguir editando' }))
    expect(screen.getByRole('heading', { level: 1, name: 'Ajustes' })).toBeInTheDocument()
    expect(screen.getByLabelText('Revisar cada')).toHaveValue('45s')

    await userEvent.click(within(nav).getByRole('link', { name: 'Inicio' }))
    await userEvent.click(
      within(await screen.findByRole('alertdialog', { name: 'Tienes cambios sin guardar' })).getByRole('button', { name: 'Descartar' }),
    )
    expect(await screen.findByRole('heading', { level: 1, name: 'Inicio' })).toBeInTheDocument()
  })

  it('leaves without asking when nothing changed', async () => {
    stubFetch(base)
    openSettings()
    await screen.findByLabelText('Revisar cada')
    const nav = screen.getAllByRole('navigation', { name: 'Secciones' })[0]!
    await userEvent.click(within(nav).getByRole('link', { name: 'Inicio' }))
    expect(await screen.findByRole('heading', { level: 1, name: 'Inicio' })).toBeInTheDocument()
    expect(screen.queryByRole('alertdialog')).toBeNull()
  })
})

describe('Ajustes', () => {
  it('shows version, config file and the read-only panel address with its CLI commands', async () => {
    stubFetch(base)
    openSettings()
    const system = (await screen.findByRole('heading', { name: 'Kongtrol en este equipo' })).closest('section')!
    expect(await within(system).findByText('v1.4.0')).toBeInTheDocument()
    expect(await within(system).findByText('C:\\Users\\ana\\.kongtrol\\kongtrol.yaml')).toBeInTheDocument()
    expect(await within(system).findByText('127.0.0.1:7777')).toBeInTheDocument()
    expect(within(system).getByText('kongtrol config dashboard set-port')).toBeInTheDocument()
    // read-only: no input for the address
    expect(within(system).queryByRole('textbox')).not.toBeInTheDocument()
  })

  it('validates health check times before saving and sends only that section', async () => {
    const fetchMock = stubFetch({ ...base, 'PUT /api/v1/settings': () => json({ ...settings, health_check_interval: '1m' }) })
    openSettings()
    const interval = await screen.findByLabelText('Revisar cada')
    await userEvent.clear(interval)
    await userEvent.type(interval, '30 segundos')
    const health = interval.closest('section')!
    await userEvent.click(within(health).getByRole('button', { name: 'Guardar' }))
    expect(await within(health).findByText('Ese tiempo no se entiende.')).toBeInTheDocument()
    expect(within(health).getByText('Cambios sin guardar')).toBeInTheDocument()
    expect(fetchMock.mock.calls.some(([, i]) => i?.method === 'PUT')).toBe(false)

    await userEvent.clear(interval)
    await userEvent.type(interval, '1m')
    await userEvent.click(within(health).getByRole('button', { name: 'Guardar' }))
    await waitFor(() => expect(fetchMock.mock.calls.some(([, i]) => i?.method === 'PUT')).toBe(true))
    const put = fetchMock.mock.calls.find(([, i]) => i?.method === 'PUT')
    expect(JSON.parse(String(put?.[1]?.body))).toEqual({ health_check_interval: '1m', health_check_timeout: '10s' })
    expect(await screen.findByText('Ajustes guardados.')).toBeInTheDocument()
  })

  it('Deshacer restores the saved values', async () => {
    stubFetch(base)
    openSettings()
    const timeout = await screen.findByLabelText('Esperar respuesta')
    await userEvent.clear(timeout)
    await userEvent.type(timeout, '99s')
    await userEvent.click(screen.getByRole('button', { name: 'Deshacer cambios' }))
    expect(timeout).toHaveValue('10s')
  })

  it('quit asks first; Cancelar sends nothing, confirming stops Kongtrol', async () => {
    const fetchMock = stubFetch({ ...base, 'POST /api/v1/shutdown': () => json({ status: 'shutting_down' }, 202) })
    openSettings()
    await userEvent.click(await screen.findByRole('button', { name: 'Salir y desconectar todo' }))
    const dialog = screen.getByRole('alertdialog', { name: '¿Salir y desconectar todo?' })
    await userEvent.click(within(dialog).getByRole('button', { name: 'Cancelar' }))
    expect(fetchMock.mock.calls.some(([u]) => String(u) === '/api/v1/shutdown')).toBe(false)

    await userEvent.click(screen.getByRole('button', { name: 'Salir y desconectar todo' }))
    await userEvent.click(within(screen.getByRole('alertdialog', { name: '¿Salir y desconectar todo?' })).getByRole('button', { name: 'Sí, salir' }))
    expect(await screen.findByText(/Kongtrol se cerró y tus VPN están desconectadas/)).toBeInTheDocument()
  })

  it('public IP lives in Seguridad: state + link, no second switch', async () => {
    stubFetch(base)
    openSettings()
    const privacy = (await screen.findByRole('heading', { name: 'Privacidad' })).closest('section')!
    expect(await within(privacy).findByText('Apagada')).toBeInTheDocument()
    expect(within(privacy).getByRole('link', { name: 'Abrir Seguridad' })).toHaveAttribute('href', '/security')
    expect(within(privacy).queryByRole('switch')).not.toBeInTheDocument()
  })

  it('language and theme apply at once and help links follow the language', async () => {
    const fetchMock = stubFetch({ ...base, 'PUT /api/v1/preferences': () => json({ language: 'en', theme: '' }) })
    openSettings()
    const about = (await screen.findByRole('heading', { name: 'Ayuda y documentos' })).closest('section')!
    expect(within(about).getByRole('link', { name: /Privacidad/ })).toHaveAttribute('href', 'https://kongtrol.app/privacidad')
    await userEvent.click(screen.getByRole('radio', { name: 'English' }))
    expect(await screen.findByRole('heading', { level: 1, name: 'Settings' })).toBeInTheDocument()
    expect(within(about).getByRole('link', { name: /Security policy/ })).toHaveAttribute('href', 'https://kongtrol.app/en/security')
    await userEvent.click(screen.getByRole('radio', { name: 'Dark' }))
    expect(document.documentElement.dataset.theme).toBe('dark')
    expect(fetchMock.mock.calls.filter(([u, i]) => String(u) === '/api/v1/preferences' && i?.method === 'PUT')).toHaveLength(2)
  })

  it('purge stays behind a confirmation on this page too', async () => {
    const fetchMock = stubFetch(base)
    openSettings()
    await userEvent.click(await screen.findByRole('button', { name: 'Borrar todos mis datos' }))
    expect(screen.getByRole('alertdialog', { name: '¿Borrar todos tus datos?' })).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Cancelar' }))
    expect(fetchMock.mock.calls.some(([u]) => String(u) === '/api/v1/data/purge')).toBe(false)
  })
})

describe('settings validation', () => {
  it('parses Go durations', () => {
    expect(parseDuration('30s')).toBe(30_000)
    expect(parseDuration('1h30m')).toBe(5_400_000)
    expect(parseDuration('1.5m')).toBe(90_000)
    expect(parseDuration('30')).toBeNull()
    expect(parseDuration('-1s')).toBeNull()
  })

  it('rejects empty, unparsable and sub-second times', () => {
    expect(validateDuration('')?.message).toBe('settings.err.durationEmpty')
    expect(validateDuration('soon')?.message).toBe('settings.err.duration')
    expect(validateDuration('500ms')?.message).toBe('settings.err.durationShort')
    expect(validateDuration(' 10s ')).toBeNull()
  })

  it('max size is a whole number 1–1024, never empty or 0', () => {
    expect(validateMaxSize('')?.message).toBe('settings.err.sizeEmpty')
    expect(validateMaxSize('0')?.message).toBe('settings.err.size')
    expect(validateMaxSize('1.5')?.message).toBe('settings.err.size')
    expect(validateMaxSize('1025')?.message).toBe('settings.err.size')
    expect(validateMaxSize('100')).toBeNull()
  })
})
