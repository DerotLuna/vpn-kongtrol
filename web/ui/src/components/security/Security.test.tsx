// Seguridad (/security): toggles (optimistic + rollback), confirmations,
// the public IP consent notice before enabling, states and settings forms.
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { App } from '../../App'
import { messages } from '../../i18n/messages'
import { helperReady, json, stubFetch, type Routes } from '../../test/fakes'
import type { SecuritySettings, SecurityState } from '../../data/security'
import { dnsView, isDuration, isIP, killSwitchView, leakPhase } from './model'
import { invalidEndpoints, lanPatch, parseEndpoints } from '../../data/security'

class SilentSocket {
  onopen: (() => void) | null = null
  onmessage: ((e: MessageEvent) => void) | null = null
  onclose: (() => void) | null = null
  close() {}
}

const armed: SecurityState = {
  kill_switch: false,
  kill_switch_enabled: true,
  kill_switch_state: 'armed',
  kill_switch_status: { state: 'armed', protected: ['oficina', 'casa'], blocked_count: 0 },
  dns_guard: true,
  dns_guard_enabled: true,
  dns_guard_state: 'active',
  public_ip_check_enabled: false,
  public_ip_check_host: 'api.ipify.org',
  leak_check: null,
}

const settings: SecuritySettings = {
  split_dns_enabled: false,
  split_dns_interval: '60s',
  kill_switch_mode: '',
  kill_switch_allow_lan: false,
  dns_guard_fallback_dns: '',
  public_ip_check_enabled: false,
  public_ip_endpoints: [],
}

function deferred() {
  let resolve!: (r: Response) => void
  const promise = new Promise<Response>((r) => (resolve = r))
  return { promise, resolve }
}

function setup(extra: Routes = {}, status: SecurityState = armed) {
  vi.stubGlobal('WebSocket', SilentSocket)
  const fetchMock = stubFetch({
    'GET /api/v1/preferences': () => json({ language: 'es', theme: '' }),
    'GET /api/v1/tunnels': () => json([]),
    'GET /api/v1/helper/status': () => json(helperReady),
    'GET /api/v1/security/status': () => json(status),
    'GET /api/v1/settings': () => json(settings),
    'GET /api/v1/vpns': () => json([{ name: 'oficina', type: 'forticlient', kill_switch_override: 'inherit' }]),
    'GET /api/v1/policies': () =>
      json([{ name: 'intranet', via: 'oficina', domains: ['*.corp'], ip_ranges: [], apps: [], on_tunnel_down: 'block' }]),
    'GET /api/v1/events?after=0': () => json({ events: [], seq: 0 }),
    ...extra,
  })
  window.history.replaceState(null, '', '/security')
  render(<App />)
  return fetchMock
}

const posts = (fetchMock: ReturnType<typeof stubFetch>, url: string) =>
  fetchMock.mock.calls.filter(([u, init]) => String(u) === url && init?.method && init.method !== 'GET')

const ksSwitch = () => screen.findByRole('switch', { name: 'Bloquear si una VPN se cae' })

describe('Seguridad page', () => {
  beforeEach(() => {
    Object.defineProperty(navigator, 'languages', { value: ['es-AR'], configurable: true })
  })

  it('explains the kill switch state and the tunnels it protects', async () => {
    setup()
    expect(await screen.findByText('Protegiendo')).toBeInTheDocument()
    expect(screen.getByText('Vigila tus VPN conectadas (2). Si una se corta, bloquea lo que iba por ella.')).toBeInTheDocument()
    expect(screen.getByText('casa')).toBeInTheDocument()
    // Rule overview: on_tunnel_down: block
    expect(await screen.findByText('Se bloquea')).toBeInTheDocument()
  })

  it('turning the kill switch off asks first; Cancel sends nothing', async () => {
    const fetchMock = setup({ 'POST /api/v1/security/killswitch': () => json({ status: 'updated' }) })
    const sw = await ksSwitch()
    expect(sw).toHaveAttribute('aria-checked', 'true')
    await userEvent.click(sw)
    const dialog = screen.getByRole('alertdialog', { name: '¿Apagar el kill switch?' })
    await userEvent.click(within(dialog).getByRole('button', { name: 'Cancelar' }))
    expect(posts(fetchMock, '/api/v1/security/killswitch')).toHaveLength(0)
    expect(sw).toHaveAttribute('aria-checked', 'true')
  })

  it('kill switch off is optimistic and rolls back with a toast when refused', async () => {
    const pending = deferred()
    const fetchMock = setup({ 'POST /api/v1/security/killswitch': () => pending.promise })
    const sw = await ksSwitch()
    await userEvent.click(sw)
    await userEvent.click(screen.getByRole('button', { name: 'Sí, apagar' }))
    // Moves at once, before the daemon answers.
    await waitFor(() => expect(sw).toHaveAttribute('aria-checked', 'false'))
    const call = posts(fetchMock, '/api/v1/security/killswitch')[0]
    expect(JSON.parse(String(call?.[1]?.body))).toEqual({ enabled: false })

    pending.resolve(new Response(JSON.stringify({ error: 'no se pudo guardar', hint: 'Revisa el archivo' }), { status: 500 }))
    await waitFor(() => expect(sw).toHaveAttribute('aria-checked', 'true'))
    expect(await screen.findByText('no se pudo guardar')).toBeInTheDocument()
    expect(screen.getByText('Revisa el archivo')).toBeInTheDocument()
  })

  it('turning the kill switch on needs no confirmation', async () => {
    const fetchMock = setup(
      { 'POST /api/v1/security/killswitch': () => json({ status: 'updated' }) },
      { ...armed, kill_switch_enabled: false, kill_switch_state: 'disabled', kill_switch_status: { state: 'off', blocked_count: 0 } },
    )
    const sw = await ksSwitch()
    await userEvent.click(sw)
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    await waitFor(() => expect(posts(fetchMock, '/api/v1/security/killswitch')).toHaveLength(1))
  })

  it('shows the external-service notice and asks BEFORE enabling the public IP check', async () => {
    const fetchMock = setup({
      'POST /api/v1/security/publicip': () => json({ status: 'updated', public_ip_check_enabled: true, public_ip_check_host: 'api.ipify.org' }),
    })
    const sw = await screen.findByRole('switch', { name: 'Revisar mi IP pública' })
    await userEvent.click(sw)
    const dialog = screen.getByRole('alertdialog', { name: '¿Activar la revisión de IP pública?' })
    expect(within(dialog).getByText(/Usa un servicio externo \(api\.ipify\.org\) que ve tu IP pública/)).toBeInTheDocument()
    expect(posts(fetchMock, '/api/v1/security/publicip')).toHaveLength(0)

    await userEvent.click(within(dialog).getByRole('button', { name: 'Cancelar' }))
    expect(posts(fetchMock, '/api/v1/security/publicip')).toHaveLength(0)
    expect(sw).toHaveAttribute('aria-checked', 'false')

    await userEvent.click(sw)
    await userEvent.click(screen.getByRole('button', { name: 'Activar' }))
    await waitFor(() => expect(posts(fetchMock, '/api/v1/security/publicip')).toHaveLength(1))
    expect(JSON.parse(String(posts(fetchMock, '/api/v1/security/publicip')[0]?.[1]?.body))).toEqual({ enabled: true })
  })

  it('per-profile override saves at once and rolls back if refused', async () => {
    const fetchMock = setup({ 'PUT /api/v1/vpns/oficina/killswitch': () => json({ error: 'vpn profile not found' }, 404) })
    const select = await screen.findByRole('combobox', { name: 'Kill switch de oficina' })
    await userEvent.selectOptions(select, 'off')
    const call = posts(fetchMock, '/api/v1/vpns/oficina/killswitch')[0]
    expect(JSON.parse(String(call?.[1]?.body))).toEqual({ override: 'off' })
    await waitFor(() => expect(select).toHaveValue('inherit'))
    expect(await screen.findByText('vpn profile not found')).toBeInTheDocument()
  })

  it('rejects a non-https lookup service inline without saving', async () => {
    const fetchMock = setup()
    const field = await screen.findByRole('textbox', { name: 'Servicios de consulta' })
    await userEvent.type(field, 'http://ip.example')
    const form = field.closest('form')!
    await userEvent.click(within(form).getByRole('button', { name: 'Guardar' }))
    expect(screen.getByText(/no es una dirección https:\/\/ válida: http:\/\/ip\.example/)).toBeInTheDocument()
    expect(field).toHaveAttribute('aria-invalid', 'true')
    expect(posts(fetchMock, '/api/v1/settings')).toHaveLength(0)
  })
})

describe('Seguridad model', () => {
  it('maps every kill switch phase to a pill and a sentence', () => {
    expect(killSwitchView({ ...armed, kill_switch_enabled: false }).pill).toBe('security.ks.pill.off')
    expect(killSwitchView({ ...armed, kill_switch_status: { state: 'armed', blocked_count: 0 } }).pill).toBe('security.ks.pill.armedNone')
    expect(killSwitchView(armed)).toMatchObject({ tone: 'ok', vars: { n: 2 } })
    const active = killSwitchView({
      ...armed,
      kill_switch_state: 'active',
      kill_switch_status: { state: 'active', blocking: ['oficina'], blocked_count: 12 },
    })
    expect(active).toMatchObject({ tone: 'caution', pill: 'security.ks.pill.active', vars: { names: 'oficina', count: 12 } })
  })

  it('DNS guard and leak states', () => {
    expect(dnsView(armed).tone).toBe('ok')
    expect(dnsView({ ...armed, dns_guard_state: 'idle' }).pill).toBe('security.dns.pill.idle')
    expect(dnsView({ ...armed, dns_guard_enabled: false }).pill).toBe('security.dns.pill.disabled')
    expect(leakPhase(armed)).toBe('pending')
    expect(leakPhase({ ...armed, leak_check: { state: 'leak', has_leak: true, checked_at: '' } })).toBe('leak')
  })

  it('validates settings input like the daemon', () => {
    expect(parseEndpoints(' https://a.example \n\nhttps://b.example')).toEqual(['https://a.example', 'https://b.example'])
    expect(invalidEndpoints(['https://ok.example', 'http://no.example', 'nope'])).toEqual(['http://no.example', 'nope'])
    expect(isIP('1.1.1.1')).toBe(true)
    expect(isIP('2606:4700::1111')).toBe(true)
    expect(isIP('999.1.1.1')).toBe(false)
    expect(isDuration('1m30s')).toBe(true)
    expect(isDuration('60')).toBe(false)
    expect(lanPatch(false, { kill_switch_mode: 'loose' })).toEqual({ kill_switch_allow_lan: false, kill_switch_mode: 'strict' })
    expect(lanPatch(true, { kill_switch_mode: '' })).toEqual({ kill_switch_allow_lan: true })
  })

  it('every security string exists in ES and EN', () => {
    const es = Object.keys(messages.es).filter((k) => k.startsWith('security.'))
    expect(es.length).toBeGreaterThan(50)
    for (const k of es) expect(messages.en[k as keyof typeof messages.en]).toBeTruthy()
  })
})
