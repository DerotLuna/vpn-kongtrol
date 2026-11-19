// Inicio (/): every state and interaction of docs/dashboard-parity.md "/ Inicio".
import { act, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { App } from '../App'
import type { TunnelMetrics } from '../api/types'
import { DataProvider } from '../app/DataProvider'
import { PreferencesProvider } from '../app/PreferencesProvider'
import { formatBytes, formatDuration, formatClock } from '../components/home/format'
import { attachKey, attachWaiting, routesOpen, securityView, sparkPoints } from '../components/home/model'
import { TunnelsCard } from '../components/home/TunnelsCard'
import {
  LATENCY_SAMPLES,
  bulkTargets,
  latencyHistory,
  pendingActions,
  recentNotices,
  recordLatency,
  type SecuritySummary,
} from '../data/home'
import { toasts } from '../data/toast'
import { FakeSocket, helperMissing, helperReady, json, stubFetch, tunnel, type Routes } from '../test/fakes'

const security = (extra: Partial<SecuritySummary> = {}): SecuritySummary => ({
  kill_switch: false,
  kill_switch_enabled: true,
  kill_switch_state: 'armed',
  dns_guard: true,
  dns_guard_enabled: true,
  dns_guard_state: 'active',
  public_ip_check_enabled: false,
  public_ip_check_host: 'api.ipify.org',
  leak_check: null,
  kill_switch_status: { state: 'armed', protected: ['work', 'home'] },
  ...extra,
})

const overview = {
  connected_tunnels: 1,
  default_routes: [{ destination: '0.0.0.0/0', gateway: '192.168.1.1', interface: 'Wi-Fi', metric: 25 }],
  local_ips: ['192.168.1.20'],
  public_ip_check_enabled: false,
}

const work = tunnel('work', 'connected', {
  AssignedIP: '10.8.0.2',
  UptimeSeconds: 3725,
  BytesSent: 1536,
  BytesReceived: 5 * 1024 * 1024,
  LatencyMS: 42,
})

function base(tunnels: TunnelMetrics[], extra: Routes = {}): Routes {
  return {
    'GET /api/v1/preferences': () => json({ language: '', theme: '' }),
    'GET /api/v1/helper/status': () => json(helperReady),
    'GET /api/v1/tunnels': () => json(tunnels),
    'GET /api/v1/security/status': () => json(security()),
    'GET /api/v1/network/overview': () => json(overview),
    'GET /api/v1/tunnels/routes': () =>
      json({
        rest_via: 'Wi-Fi',
        tunnels: [
          {
            name: 'work',
            connected: true,
            interface: 'wg0',
            all_traffic: false,
            total: 3,
            counts: { policy: 2, server: 1 },
            routes: [
              { destination: '10.0.0.0/8', origin: 'server' },
              { destination: '140.82.112.0/20', origin: 'policy', policy: 'github' },
              { destination: '192.30.252.0/22', origin: 'policy', policy: 'github' },
            ],
          },
        ],
      }),
    'GET /api/v1/groups': () => json([]),
    'GET /api/v1/events?after=-1': () => json({ events: [], seq: 0 }),
    ...extra,
  }
}

class SilentSocket {
  onopen: (() => void) | null = null
  onmessage: ((e: MessageEvent) => void) | null = null
  onclose: (() => void) | null = null
  close() {}
}

function stubApi(routes: Routes, socket: unknown = SilentSocket) {
  const fetchMock = stubFetch(routes)
  vi.stubGlobal('WebSocket', socket)
  return fetchMock
}

/** User actions posted (the live feed's ticket request is not one). */
function posts(fetchMock: ReturnType<typeof stubFetch>): string[] {
  return fetchMock.mock.calls
    .filter(([url, init]) => init?.method === 'POST' && String(url) !== '/api/v1/ws-ticket')
    .map(([url]) => String(url))
}

async function rowOf(name: string): Promise<HTMLElement> {
  const cell = await screen.findByText(name, { selector: '.home-name' })
  return cell.closest('tr')!
}

beforeEach(() => {
  Object.defineProperty(navigator, 'languages', { value: ['es-AR'], configurable: true })
  sessionStorage.clear()
  latencyHistory.set({})
  pendingActions.set({})
  routesOpen.set({})
  recentNotices.set([])
  FakeSocket.reset()
})

describe('Inicio: states', () => {
  it('shows skeletons first, then every tunnel field, in Spanish', async () => {
    stubApi(base([work, tunnel('home', 'disconnected')]))
    render(<App />)

    expect(screen.getByRole('heading', { level: 1, name: 'Inicio' })).toBeInTheDocument()
    expect(screen.getByRole('status', { name: 'Cargando tus VPN' })).toBeInTheDocument()

    const row = await rowOf('work')
    expect(within(row).getByText('Conectada')).toHaveClass('pill', 'ok')
    expect(within(row).getByText('10.8.0.2')).toBeInTheDocument()
    expect(within(row).getByText('1h 02m')).toBeInTheDocument()
    expect(within(row).getByText('1,5 KB')).toBeInTheDocument()
    expect(within(row).getByText('5 MB')).toBeInTheDocument()
    expect(within(row).getByText('42 ms')).toBeInTheDocument()
    // Accessible names carry the tunnel: "Desconectar work", not a bare "Desconectar".
    expect(within(row).getByRole('button', { name: 'Desconectar work' })).toBeEnabled()
    expect(within(row).getByRole('button', { name: 'Reconectar work' })).toBeEnabled()

    const off = await rowOf('home')
    expect(within(off).getByText('Desconectada')).toHaveClass('pill')
    expect(within(off).getAllByText('—').length).toBeGreaterThanOrEqual(5)
    expect(within(off).getByRole('button', { name: 'Conectar home' })).toBeEnabled()

    expect(screen.getByText('1 de 2 conectadas', { selector: '.home-count' })).toBeInTheDocument()
    // No helper banner when the helper is ready.
    expect(screen.queryByRole('button', { name: 'Instalar ayudante' })).toBeNull()
  })

  it('first run: no VPNs yet → one primary "Agregar VPN" that opens the add sheet + the quick start guide', async () => {
    stubApi(base([]))
    render(<App />)
    expect(await screen.findByRole('heading', { name: 'Agrega tu primera VPN' })).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Agregar VPN' }))
    expect(await screen.findByRole('dialog', { name: 'Agregar' })).toBeInTheDocument()
    expect(new URLSearchParams(window.location.search).get('add')).toBe('menu')
    expect(screen.getByRole('link', { name: /guía de inicio rápido/ })).toHaveAttribute('target', '_blank')
    expect(screen.queryByRole('button', { name: 'Conectar todas' })).toBeNull()
  })

  it('stopped daemon: says what to do at once, in the app banner and the tunnels card', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.reject(new TypeError('Failed to fetch'))))
    vi.stubGlobal('WebSocket', SilentSocket)
    render(<App />)
    const alerts = await screen.findAllByRole('alert')
    expect(alerts.some((a) => within(a).queryByText('No se pudo hablar con Kongtrol.'))).toBe(true)
    expect(alerts.some((a) => within(a).queryByText(/Qué hacer:/))).toBe(true)
  })

  it('helper missing or outdated: one banner on Inicio with the install / update action', async () => {
    stubApi(base([work], { 'GET /api/v1/helper/status': () => json({ ...helperMissing, installed: true, running: true, outdated: true }) }))
    render(<App />)
    expect(await screen.findByText('El ayudante está desactualizado')).toBeInTheDocument()
    expect(screen.getAllByRole('button', { name: 'Actualizar ayudante' })).toHaveLength(1)
    expect(screen.queryByRole('button', { name: 'Instalar ayudante' })).toBeNull()
  })

  it('a card that cannot load says what happened and what to do, with Reintentar', async () => {
    stubApi(base([work], { 'GET /api/v1/network/overview': () => json({ error: 'boom', hint: 'Qué hacer: reinicia.' }, 400) }))
    render(<App />)
    const card = (await screen.findByRole('heading', { name: 'Tu salida a internet' })).closest('section')!
    expect(await within(card).findByText('boom')).toBeInTheDocument()
    expect(within(card).getByText('Qué hacer: reinicia.')).toBeInTheDocument()
    expect(within(card).getByRole('button', { name: 'Reintentar' })).toBeEnabled()
  })
})

describe('Inicio: tunnel lines', () => {
  it('attach, health and last error read as plain lines; the raw error is on demand', async () => {
    stubApi(
      base([
        tunnel('nord', 'connecting', { attach: { phase: 'waiting_user', app: 'NordVPN', since: '' } }),
        { ...tunnel('brainbox', 'error'), health: { state: 'no_response' } },
        { ...tunnel('lab', 'connected'), health: { state: 'unknown', reason_key: 'service.error.outdated' } },
        { ...tunnel('idle', 'connected'), health: { state: 'idle' } },
        tunnel('office', 'error', {
          last_error: 'privops: file /etc/x.conf owned by uid 0',
          last_error_key: 'privops.denied.file_owner',
          last_error_at: '2026-11-18T10:20:30Z',
        }),
      ]),
    )
    render(<App />)

    const nord = await rowOf('nord')
    expect(within(nord).getByText('Conectando')).toHaveClass('pill', 'connecting')
    expect(within(nord).getByText('Abre NordVPN y conéctate ahí; Kongtrol lo detecta solo.')).toHaveClass('home-line--act')
    expect(within(nord).getByRole('button', { name: 'Cancelar nord' })).toBeEnabled()

    const brainbox = await rowOf('brainbox')
    expect(within(brainbox).getByText('Sin respuesta')).toHaveClass('pill', 'caution')
    expect(within(brainbox).getByText(/Sin respuesta del servidor · Kongtrol reintenta solo/)).toBeInTheDocument()

    const lab = await rowOf('lab')
    expect(within(lab).getByText('· sin verificar')).toBeInTheDocument()
    expect(within(lab).getByText(/pulsa «Actualizar ayudante» arriba/)).toBeInTheDocument()

    expect(within(await rowOf('idle')).getByText(/Sin tráfico reciente/)).toBeInTheDocument()

    const office = await rowOf('office')
    expect(within(office).getByText(/Último error \(/)).toBeInTheDocument()
    expect(within(office).getByText(/no es tuyo/)).toBeInTheDocument()
    const detail = within(office).getByText('Detalle técnico')
    expect(within(office).getByText('privops: file /etc/x.conf owned by uid 0')).not.toBeVisible()
    await userEvent.click(detail)
    expect(within(office).getByText('privops: file /etc/x.conf owned by uid 0')).toBeVisible()
  })

  it('"Rutas" opens in place (aria-expanded) and groups routes by origin', async () => {
    stubApi(base([work]))
    render(<App />)
    const row = await rowOf('work')
    const toggle = await within(row).findByRole('button', { name: 'Rutas (3)' })
    expect(toggle).toHaveAttribute('aria-expanded', 'false')
    await userEvent.click(toggle)
    expect(toggle).toHaveAttribute('aria-expanded', 'true')
    const panel = document.getElementById(toggle.getAttribute('aria-controls')!)!
    expect(panel).toBeVisible()
    expect(within(panel).getByText(/Por tu regla «github»/)).toBeInTheDocument()
    expect(within(panel).getByText('140.82.112.0/20')).toBeInTheDocument()
    expect(within(panel).getByText(/Lo envió el servidor de la VPN/)).toBeInTheDocument()
    expect(within(panel).getByText('Todo lo demás sale por Wi-Fi.')).toBeInTheDocument()
    expect(within(panel).getByRole('link', { name: 'Ver todas las rutas en Tráfico' })).toHaveAttribute('href', '/traffic')
    // Full width: its own row under the tunnel, one cell spanning every column.
    const detail = panel.closest('tr')!
    expect(detail).not.toBe(row)
    expect(detail.previousElementSibling).toBe(row)
    expect(panel.closest('td')).toHaveAttribute('colspan', String(row.children.length))
    // The long question lives in the tooltip (aria-describedby), the label stays one short line.
    expect(toggle).toHaveAccessibleDescription('Qué sale por este túnel: los destinos que van por esta VPN')
    await userEvent.click(toggle)
    expect(toggle).toHaveAttribute('aria-expanded', 'false')
    expect(panel).not.toBeInTheDocument()
  })

  it('latency trend: an SVG with a text alternative, fed by live ticks', async () => {
    stubApi(base([work]), FakeSocket)
    render(<App />)
    const row = await rowOf('work')
    expect(within(row).getByText('Midiendo')).toBeInTheDocument()
    const ws = FakeSocket.last()
    act(() => ws.open())
    act(() => ws.push({ work: { ...work, LatencyMS: 40 } }))
    act(() => ws.push({ work: { ...work, LatencyMS: 40 } })) // same value: not a new sample
    act(() => ws.push({ work: { ...work, LatencyMS: 55 } }))
    await waitFor(() => expect(latencyHistory.get()).toEqual({ work: [42, 40, 55] }))
    expect(
      await within(row).findByRole('img', { name: 'Latencia de las últimas 3 medidas: entre 40 y 55 ms, ahora 55 ms.' }),
    ).toBeInTheDocument()
  })
})

describe('Inicio: tunnel actions', () => {
  it('connect: "Conectando" at once, the name is URL-encoded, routes refresh after', async () => {
    let resolve!: (r: Response) => void
    const fetchMock = stubApi(
      base([tunnel('mi vpn', 'disconnected')], {
        'POST /api/v1/tunnels/mi%20vpn/connect': () => new Promise<Response>((r) => (resolve = r)),
      }),
    )
    render(<App />)
    const row = await rowOf('mi vpn')
    await userEvent.click(within(row).getByRole('button', { name: 'Conectar mi vpn' }))
    expect(await within(row).findByText('Conectando', { selector: '.pill' })).toHaveClass('connecting')
    expect(within(row).getByRole('button', { name: 'Conectando… mi vpn' })).toHaveAttribute('aria-busy', 'true')
    expect(posts(fetchMock)).toEqual(['/api/v1/tunnels/mi%20vpn/connect'])
    resolve(new Response(JSON.stringify({ status: 'connecting', tunnel: 'mi vpn' }), { status: 202 }))
    await waitFor(() =>
      expect(fetchMock.mock.calls.filter(([u]) => String(u) === '/api/v1/network/overview').length).toBeGreaterThan(1),
    )
  })

  it('a refused connect rolls back and toasts the problem + what to do + Reintentar', async () => {
    stubApi(base([tunnel('work', 'disconnected')], { 'POST /api/v1/tunnels/work/connect': () => json({ error: 'tunnel not found: work' }, 404) }))
    render(<App />)
    const row = await rowOf('work')
    await userEvent.click(within(row).getByRole('button', { name: 'Conectar work' }))
    expect(await screen.findByText('Kongtrol no encuentra la VPN work.')).toBeInTheDocument()
    expect(screen.getByText(/Qué hacer: actualiza la lista/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Reintentar' })).toBeInTheDocument()
    expect(await within(row).findByText('Desconectada')).toBeInTheDocument()
  })

  it('a connect already in progress (409) is not an error', async () => {
    stubApi(base([tunnel('work', 'disconnected')], { 'POST /api/v1/tunnels/work/connect': () => json({ error: 'connect already in progress for work' }, 409) }))
    render(<App />)
    await userEvent.click(within(await rowOf('work')).getByRole('button', { name: 'Conectar work' }))
    await waitFor(() => expect(pendingActions.get()).toEqual({}))
    expect(screen.queryByText(/already in progress/)).toBeNull()
  })

  it('disconnect of a provider app without CLI (202 waiting_user) explains what to do', async () => {
    stubApi(base([work], { 'POST /api/v1/tunnels/work/disconnect': () => json({ status: 'waiting_user', tunnel: 'work' }, 202) }))
    render(<App />)
    await userEvent.click(within(await rowOf('work')).getByRole('button', { name: 'Desconectar work' }))
    expect(await screen.findByText('Desconecta work en su app; Kongtrol lo detecta solo.')).toBeInTheDocument()
    // Never flickers to "Desconectada": the daemon still says connected.
    expect(within(await rowOf('work')).getByText('Conectada')).toBeInTheDocument()
  })

  it('disconnect in flight: row buttons inert, spinner + "Desconectando…", pill in transition, one request on double click', async () => {
    let resolve!: (r: Response) => void
    const fetchMock = stubApi(base([work], { 'POST /api/v1/tunnels/work/disconnect': () => new Promise<Response>((r) => (resolve = r)) }))
    render(<App />)
    const row = await rowOf('work')
    await userEvent.dblClick(within(row).getByRole('button', { name: 'Desconectar work' }))

    const pressed = await within(row).findByRole('button', { name: 'Desconectando… work' })
    expect(pressed).toHaveAttribute('aria-busy', 'true')
    expect(pressed).toHaveAttribute('aria-disabled', 'true')
    expect(pressed.querySelector('.k-spinner')).not.toBeNull()
    expect(pressed).toHaveFocus() // aria-disabled, not disabled: focus stays where the user pressed
    const other = within(row).getByRole('button', { name: 'Reconectar work' })
    expect(other).toHaveAttribute('aria-disabled', 'true')
    expect(other).not.toHaveAttribute('aria-busy')
    expect(within(row).getByText('Desconectando', { selector: '.pill' })).toHaveClass('connecting')

    await userEvent.click(pressed)
    await userEvent.click(other)
    expect(posts(fetchMock)).toEqual(['/api/v1/tunnels/work/disconnect'])

    resolve(new Response(JSON.stringify({ status: 'disconnected', tunnel: 'work' }), { status: 200 }))
    await waitFor(() => expect(pendingActions.get()).toEqual({}))
  })

  it('reconnect in flight: the icon button spins, the pill says "Reconectando", Desconectar waits', async () => {
    let resolve!: (r: Response) => void
    const fetchMock = stubApi(base([work], { 'POST /api/v1/tunnels/work/reload': () => new Promise<Response>((r) => (resolve = r)) }))
    render(<App />)
    const row = await rowOf('work')
    const icon = within(row).getByRole('button', { name: 'Reconectar work' })
    expect(icon).toHaveAccessibleDescription('Reconectar: vuelve a leer su configuración guardada y reinicia la conexión')
    expect(icon.querySelector('svg')).not.toBeNull()
    await userEvent.dblClick(icon)

    const pressed = await within(row).findByRole('button', { name: 'Reconectando… work' })
    expect(pressed).toHaveAttribute('aria-busy', 'true')
    expect(pressed.querySelector('.k-spinner')).not.toBeNull()
    expect(within(row).getByRole('button', { name: 'Desconectar work' })).toHaveAttribute('aria-disabled', 'true')
    expect(within(row).getByText('Reconectando', { selector: '.pill' })).toHaveClass('connecting')
    expect(posts(fetchMock)).toEqual(['/api/v1/tunnels/work/reload'])

    resolve(new Response(JSON.stringify({ status: 'restarting', tunnel: 'work' }), { status: 202 }))
    await waitFor(() => expect(within(row).getByRole('button', { name: 'Desconectar work' })).not.toHaveAttribute('aria-disabled'))
  })

  it('provider app waiting for the user to disconnect (waiting_disconnect): in transition, with what to do', async () => {
    const waiting = tunnel('nord', 'connected', { attach: { phase: 'waiting_disconnect', app: 'NordVPN', since: '' } })
    const fetchMock = stubApi(base([waiting]))
    render(<App />)
    const row = await rowOf('nord')
    expect(within(row).getByText('Desconectando', { selector: '.pill' })).toBeInTheDocument()
    expect(within(row).getByText('Desconéctate en NordVPN; Kongtrol espera y lo detecta solo.')).toBeInTheDocument()
    const pressed = within(row).getByRole('button', { name: 'Desconectando… nord' })
    expect(pressed).toHaveAttribute('aria-busy', 'true')
    await userEvent.click(pressed)
    expect(posts(fetchMock)).toEqual([])
  })

  it('cancel a connecting tunnel', async () => {
    const fetchMock = stubApi(base([tunnel('work', 'connecting')], { 'POST /api/v1/tunnels/work/cancel_connect': () => json({ status: 'connect_cancelled' }) }))
    render(<App />)
    await userEvent.click(within(await rowOf('work')).getByRole('button', { name: 'Cancelar work' }))
    await waitFor(() => expect(posts(fetchMock)).toContain('/api/v1/tunnels/work/cancel_connect'))
  })

  it('reload: restarting, not connected (no false "Reconectando") and restart required', async () => {
    let answer: () => Promise<Response> = () => json({ status: 'restarting', tunnel: 'work' }, 202)
    stubApi(base([work], { 'POST /api/v1/tunnels/work/reload': () => answer() }))
    render(<App />)
    const row = await rowOf('work')

    await userEvent.click(within(row).getByRole('button', { name: 'Reconectar work' }))
    expect(await screen.findByText('Reconectando work con su configuración guardada…')).toBeInTheDocument()

    act(() => toasts.clear())
    answer = () => json({ status: 'skipped_not_connected', tunnel: 'work' }, 202)
    await userEvent.click(await within(row).findByRole('button', { name: 'Reconectar work' }))
    expect(await screen.findByText(/No estaba conectada, así que no hubo que reiniciarla/)).toBeInTheDocument()

    act(() => toasts.clear())
    answer = () => json({ status: 'restart_required', tunnel: 'work', message: 'english' }, 409)
    await userEvent.click(await within(row).findByRole('button', { name: 'Reconectar work' }))
    expect(await screen.findByText('Kongtrol todavía no conoce work (se agregó a mano).')).toBeInTheDocument()
    expect(screen.getByText(/cierra Kongtrol y vuelve a abrirlo/)).toBeInTheDocument()
  })

  it('Conectar todas / Desconectar todas act only on the tunnels that need it', async () => {
    const fetchMock = stubApi(
      base([work, tunnel('home', 'disconnected'), tunnel('lab', 'error'), tunnel('nord', 'connecting')], {
        'POST /api/v1/tunnels/home/connect': () => json({ status: 'connecting' }, 202),
        'POST /api/v1/tunnels/lab/connect': () => json({ status: 'connecting' }, 202),
        'POST /api/v1/tunnels/work/disconnect': () => json({ status: 'disconnected' }),
      }),
    )
    render(<App />)
    await rowOf('work')
    await userEvent.click(screen.getByRole('button', { name: 'Conectar todas' }))
    await waitFor(() => expect(posts(fetchMock).sort()).toEqual(['/api/v1/tunnels/home/connect', '/api/v1/tunnels/lab/connect']))
    await userEvent.click(screen.getByRole('button', { name: 'Desconectar todas' }))
    await waitFor(() => expect(posts(fetchMock)).toContain('/api/v1/tunnels/work/disconnect'))
  })
})

describe('Inicio: groups, egress, protection, notices', () => {
  it('groups: connect reads the answer (started / already / unknown) and a failure says what to do', async () => {
    stubApi(
      base([work], {
        'GET /api/v1/groups': () => json([{ name: 'oficina', profiles: ['work', 'vpn2', 'nueva'] }]),
        'POST /api/v1/groups/oficina/connect': () =>
          json({ status: 'connecting', group: 'oficina', started: ['vpn2'], already_connected: ['work'] }, 202),
        'POST /api/v1/groups/oficina/disconnect': () => json({ error: 'work: boom' }, 500),
      }),
    )
    render(<App />)
    const card = (await screen.findByRole('heading', { name: 'Grupos' })).closest('section')!
    expect(await within(card).findByText('oficina')).toBeInTheDocument()
    expect(within(card).getByText('3 VPN')).toBeInTheDocument()

    await userEvent.click(within(card).getByRole('button', { name: 'Conectar oficina' }))
    expect(await screen.findByText('Conectando el grupo oficina: 1 VPN.')).toBeInTheDocument()
    expect(screen.getByText(/todavía no las conoce \(reinícialo\): nueva/)).toBeInTheDocument()

    act(() => toasts.clear()) // one toast at a time: let the error show now
    await userEvent.click(within(card).getByRole('button', { name: 'Desconectar oficina' }))
    expect(await screen.findByText('No se pudo completar la acción sobre el grupo oficina.')).toBeInTheDocument()
  })

  it('groups: empty state teaches where to create one', async () => {
    stubApi(base([work]))
    render(<App />)
    expect(await screen.findByRole('heading', { name: 'Todavía no tienes grupos' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Crear un grupo' })).toHaveAttribute('href', '/profiles/groups')
  })

  it('egress: tunnels count, normal exit, gateway, local IPs; public IP lookup off says so', async () => {
    stubApi(base([work]))
    render(<App />)
    const card = (await screen.findByRole('heading', { name: 'Tu salida a internet' })).closest('section')!
    expect(await within(card).findByText('Wi-Fi · métrica 25')).toBeInTheDocument()
    expect(within(card).getByText('192.168.1.1')).toBeInTheDocument()
    expect(within(card).getByText('192.168.1.20')).toBeInTheDocument()
    expect(within(card).getByText('consulta apagada')).toBeInTheDocument()
    expect(within(card).getByText(/no coincide con ninguna regla/)).toBeInTheDocument()
  })

  it('protection: states as words; no "Revisar" and no lookup while the public IP check is off', async () => {
    const fetchMock = stubApi(base([work]))
    render(<App />)
    const card = (await screen.findByRole('heading', { name: 'Protección' })).closest('section')!
    expect(await within(card).findByText('Armado')).toHaveClass('pill', 'on')
    expect(within(card).getByText('Protege 2 VPN conectadas.')).toBeInTheDocument()
    expect(within(card).getByText('Activa')).toHaveClass('pill', 'on')
    expect(within(card).getByText(/nada sale de tu equipo/)).toBeInTheDocument()
    expect(within(card).queryByRole('button', { name: 'Revisar IP pública' })).toBeNull()
    expect(within(card).getByRole('link', { name: 'Gestionar en Seguridad' })).toHaveAttribute('href', '/security')
    expect(fetchMock.mock.calls.some(([u]) => String(u).includes('/publicip/check'))).toBe(false)
  })

  it('protection: with the lookup on, "Revisar" asks once and shows IP + time', async () => {
    let checked = false
    const status = () =>
      json(
        security({
          public_ip_check_enabled: true,
          leak_check: checked ? { state: 'clean', has_leak: false, public_ip: '203.0.113.7', checked_at: '2026-11-18T10:00:00Z' } : null,
        }),
      )
    const fetchMock = stubApi(
      base([work], {
        'GET /api/v1/security/status': status,
        'POST /api/v1/security/publicip/check': () => {
          checked = true
          return status()
        },
      }),
    )
    render(<App />)
    const card = (await screen.findByRole('heading', { name: 'Protección' })).closest('section')!
    expect(await within(card).findByText('Sin revisar')).toHaveClass('pill', 'unknown')
    await userEvent.click(within(card).getByRole('button', { name: 'Revisar IP pública' }))
    expect(await within(card).findByText('203.0.113.7')).toBeInTheDocument()
    expect(within(card).getByText('Sin fugas')).toHaveClass('pill', 'ok')
    expect(posts(fetchMock)).toEqual(['/api/v1/security/publicip/check'])
  })

  it('notices: a policy whose VPN is off toasts with "Conectar <perfil>" and stays in the list', async () => {
    const fetchMock = stubApi(
      base([tunnel('brainbox', 'disconnected')], {
        'GET /api/v1/events?after=-1': () =>
          json({
            seq: 7,
            events: [
              {
                seq: 7,
                time: '2026-11-18T10:00:00Z',
                kind: 'policy_tunnel_down',
                level: 'WARN',
                key: 'policy.event.allow',
                args: ['claude.ai', 'brainbox'],
                profile: 'brainbox',
                can_connect: true,
              },
              { seq: 6, time: '', kind: 'other', level: 'INFO', key: 'x' },
            ],
          }),
        'POST /api/v1/tunnels/brainbox/connect': () => json({ status: 'connecting' }, 202),
      }),
    )
    render(<App />)
    const card = (await screen.findByRole('heading', { name: 'Avisos' })).closest('section')!
    expect(await within(card).findByText('claude.ai sale por internet normal · brainbox está apagado')).toBeInTheDocument()
    expect(sessionStorage.getItem('kongtrol.noticeSeq')).toBe('7')
    // The toast too (with its own action button).
    expect(screen.getAllByText('claude.ai sale por internet normal · brainbox está apagado')).toHaveLength(2)
    await userEvent.click(within(card).getByRole('button', { name: 'Conectar brainbox' }))
    await waitFor(() => expect(posts(fetchMock)).toContain('/api/v1/tunnels/brainbox/connect'))
  })

  it('notices: empty state says what will show up here', async () => {
    stubApi(base([work]))
    render(<App />)
    expect(await screen.findByText(/Sin avisos nuevos/)).toBeInTheDocument()
  })

  it('English: the whole page follows the language', async () => {
    stubApi(base([work], { 'GET /api/v1/preferences': () => json({ language: 'en', theme: '' }) }))
    render(<App />)
    expect(await screen.findByRole('heading', { level: 1, name: 'Home' })).toBeInTheDocument()
    expect(await screen.findByRole('button', { name: 'Disconnect work' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Protection' })).toBeInTheDocument()
    expect(screen.getByText('1.5 KB')).toBeInTheDocument()
  })
})

describe('Inicio: only changed rows re-render', () => {
  function Harness({ list, onRow }: { list: TunnelMetrics[]; onRow: (k: string) => void }) {
    return (
      <PreferencesProvider>
        <DataProvider>
          <MemoryRouter>
            <TunnelsCard tunnels={list} error={null} onRefresh={() => {}} onRowRender={onRow} />
          </MemoryRouter>
        </DataProvider>
      </PreferencesProvider>
    )
  }

  it('a tick that changes one tunnel re-renders only its row', async () => {
    stubApi(base([]))
    const a = tunnel('a', 'connected')
    const b = tunnel('b', 'disconnected')
    const rendered: string[] = []
    const onRow = (k: string) => void rendered.push(k)
    const { rerender } = render(<Harness list={[a, b]} onRow={onRow} />)
    await screen.findByText('a', { selector: '.home-name' })
    rendered.length = 0
    rerender(<Harness list={[{ ...a, BytesReceived: 99 }, b]} onRow={onRow} />)
    expect(rendered).toEqual(['a'])
  })
})

describe('Inicio: pure helpers', () => {
  it('formats bytes, durations and clock times', () => {
    expect(formatBytes(0, 'es')).toBeNull()
    expect(formatBytes(512, 'en')).toBe('512 B')
    expect(formatBytes(1536, 'es')).toBe('1,5 KB')
    expect(formatDuration(0)).toBe('< 1s')
    expect(formatDuration(65)).toBe('1m 05s')
    expect(formatDuration(7260)).toBe('2h 01m')
    expect(formatClock('0001-01-01T00:00:00Z', 'es')).toBe('')
    expect(formatClock(undefined, 'es')).toBe('')
  })

  it('records a latency sample only when it changes, caps the history and clears it when down', () => {
    let h = recordLatency({}, [tunnel('a', 'connected', { LatencyMS: 10 })])
    expect(h.a).toEqual([10])
    expect(recordLatency(h, [tunnel('a', 'connected', { LatencyMS: 10 })])).toBe(h)
    for (let i = 0; i < LATENCY_SAMPLES + 5; i++) h = recordLatency(h, [tunnel('a', 'connected', { LatencyMS: 20 + i })])
    expect(h.a).toHaveLength(LATENCY_SAMPLES)
    expect(recordLatency(h, [tunnel('a', 'disconnected')]).a).toEqual([])
    expect(sparkPoints([1, 2], 10, 10, 0)).toBe('0.0,10.0 10.0,0.0')
  })

  it('maps attach phases like vpn.AttachState.MessageKey', () => {
    const a = (phase: string, extra = {}) => ({ phase, app: 'X', since: '', ...extra })
    expect(attachKey(a('auto', { location: 'AR' }))).toBe('home.attach.autoLocation')
    expect(attachKey(a('waiting_user', { reason: 'timeout' }))).toBe('home.attach.automationFailed')
    expect(attachKey(a('nope'))).toBeNull()
    expect(attachWaiting(a('dropped'))).toBe(true)
    expect(attachWaiting(a('attached'))).toBe(false)
  })

  it('maps the security status like `kongtrol status`', () => {
    expect(securityView(security({ kill_switch_status: { state: 'active', blocking: ['brainbox'] } })).killSwitch).toMatchObject({
      tone: 'caution',
      label: 'home.sec.ks.active',
      detail: { vars: { names: 'brainbox' } },
    })
    expect(securityView(security({ kill_switch_enabled: false, kill_switch_state: 'disabled', kill_switch_status: { state: 'off' } })).killSwitch.label).toBe(
      'home.sec.ks.off',
    )
    expect(securityView(security({ dns_guard: false, dns_guard_state: 'idle' })).dnsGuard.label).toBe('home.sec.dns.idle')
    expect(securityView(security({ leak_detection_configured: true, leak_check_available: false })).leak.label).toBe('home.sec.leak.unavailable')
    expect(
      securityView(security({ public_ip_check_enabled: true, leak_check: { state: 'leak', has_leak: true, checked_at: '' } })).leak.tone,
    ).toBe('bad')
  })

  it('bulk targets', () => {
    const list = [tunnel('a', 'connected'), tunnel('b', 'error'), tunnel('c', 'connecting'), tunnel('d', 'disconnected')]
    expect(bulkTargets(list, 'connect')).toEqual(['b', 'd'])
    expect(bulkTargets(list, 'disconnect')).toEqual(['a'])
  })
})
