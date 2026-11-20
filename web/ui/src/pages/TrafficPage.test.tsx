import { act, render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { App } from '../App'
import type { TunnelMetrics } from '../api/types'
import { PreferencesProvider } from '../app/PreferencesProvider'
import { keys } from '../data/keys'
import { createQueryClient } from '../data/queryClient'
import { trafficHistory, type Policy, type TunnelRoutesResponse } from '../data/traffic'
import { json, pickOption, stubFetch, tunnel, type Routes } from '../test/fakes'
import { TrafficPage } from './TrafficPage'

const at = (s: number) => new Date(Date.UTC(2026, 10, 18, 12, 0, s)).toISOString()

const policies: Policy[] = [
  {
    name: 'ai',
    via: 'work',
    domains: ['claude.ai'],
    ip_ranges: null,
    apps: null,
    on_tunnel_down: 'allow',
    resolved_cidrs: ['1.1.1.1/32'],
    resolved: [{ cidr: '1.1.1.1/32', host: 'claude.ai', last_seen: at(0), expires_at: '2099-01-01T00:00:00Z' }],
  },
  {
    name: 'corp',
    via: 'office',
    domains: null,
    ip_ranges: ['10.0.0.0/8'],
    apps: null,
    on_tunnel_down: 'block',
    resolved_cidrs: null,
  },
]

const tunnelRoutes: TunnelRoutesResponse = {
  rest_via: 'Wi-Fi',
  tunnels: [
    {
      name: 'work',
      connected: true,
      interface: 'tun0',
      all_traffic: false,
      total: 2,
      counts: { server: 1, policy: 1 },
      routes: [
        { destination: '10.8.0.0/24', origin: 'server' },
        { destination: '1.1.1.1/32', origin: 'policy', policy: 'ai' },
      ],
    },
    { name: 'home', connected: true, interface: 'wg0', all_traffic: false, total: 0, counts: {}, routes: [] },
  ],
}

const work = (s: number, rx = s * 1000) =>
  tunnel('work', 'connected', { UpdatedAt: at(s), BytesReceived: rx, BytesSent: rx / 10, InterfaceName: 'tun0' })
const home = (s: number) => tunnel('home', 'connected', { UpdatedAt: at(s), InterfaceName: 'wg0' })

function api(extra: Routes = {}): Routes {
  return {
    'GET /api/v1/preferences': () => json({ language: '', theme: '' }),
    'GET /api/v1/tunnels': () => json([work(0), home(0), tunnel('tv', 'disconnected')]),
    'GET /api/v1/tunnels/routes': () => json(tunnelRoutes),
    'GET /api/v1/policies': () => json(policies),
    'GET /api/v1/metrics/history': () =>
      json({ work: { name: 'work', samples: 9, connected_samples: 9, uptime_seconds: 3725, reconnects: 2, jitter_ms: 4, last_latency_ms: 30 } }),
    ...extra,
  }
}

function renderPage(onCardRender?: (name: string) => void) {
  const client = createQueryClient()
  render(
    <PreferencesProvider>
      <QueryClientProvider client={client}>
        <MemoryRouter initialEntries={['/traffic']}>
          <TrafficPage {...(onCardRender ? { onCardRender } : {})} />
        </MemoryRouter>
      </QueryClientProvider>
    </PreferencesProvider>,
  )
  return client
}

describe('TrafficPage', () => {
  beforeEach(() => {
    Object.defineProperty(navigator, 'languages', { value: ['es-AR'], configurable: true })
    trafficHistory.reset()
  })
  afterEach(() => trafficHistory.reset())

  it('is the /traffic route, with what goes through each VPN and the "everything else" line', async () => {
    stubFetch(api())
    window.history.replaceState(null, '', '/traffic')
    render(<App />)
    expect(await screen.findByRole('heading', { level: 1, name: 'Tráfico' })).toBeInTheDocument()
    const card = await screen.findByRole('region', { name: 'work' })
    expect(within(card).getByText('Conectada')).toBeInTheDocument()
    const toggle = await within(card).findByRole('button', { name: 'Qué sale por este túnel (2)' })
    expect(toggle).toHaveAttribute('aria-expanded', 'true')
    expect(within(card).getByText('El servidor de la VPN pidió 1')).toBeInTheDocument()
    expect(within(card).getByText('Por tu regla ai: 1')).toBeInTheDocument()
    expect(within(card).getByText('Todo lo demás → tu conexión normal (Wi-Fi).')).toBeInTheDocument()
    expect(within(card).getByText('1 h 2 min')).toBeInTheDocument()
    expect(screen.getByText('Desconectadas: tv.')).toBeInTheDocument()
  })

  it('answers "¿por dónde sale?" in plain words; Enter runs the check and filters the rules', async () => {
    const fetchMock = stubFetch(
      api({
        'GET /api/v1/resolve?target=claude.ai': () => json({ target: 'claude.ai', via: 'work', rule: 'ai', matched: true }),
        'GET /api/v1/resolve?app=teams.exe': () => json({ target: '', app: 'teams.exe', via: '', rule: '', matched: false }),
        'GET /api/v1/dns/resolve?domain=claude.ai&via=work': () => json({ domain: 'claude.ai', via: 'work', ips: ['160.79.104.10'] }),
      }),
    )
    renderPage()
    const user = userEvent.setup()
    expect(await screen.findByText('2 reglas')).toBeInTheDocument()
    const input = screen.getByRole('textbox', { name: 'Sitio, IP o programa' })
    await user.type(input, 'claude.ai{Enter}')
    expect(await screen.findByText('claude.ai sale por work.')).toBeInTheDocument()
    expect(screen.getByText('Lo decide la regla ai.')).toBeInTheDocument()
    expect(screen.getByText('Mostrando 1 de 2 reglas que coinciden con «claude.ai»')).toBeInTheDocument()

    // Technical detail on demand: the IPs through the VPN's DNS.
    await user.click(screen.getByRole('button', { name: 'Ver a qué IPs apunta por work' }))
    expect(await screen.findByText('160.79.104.10')).toBeInTheDocument()

    // app: switches the query to ?app= and no rule → normal connection.
    await user.clear(input)
    await user.type(input, 'app:teams.exe{Enter}')
    expect(await screen.findByText('app:teams.exe sale por tu conexión normal.')).toBeInTheDocument()
    expect(screen.getByText('Ninguna regla lo manda por una VPN. Sale por Wi-Fi.')).toBeInTheDocument()
    expect(fetchMock.mock.calls.some(([u]) => String(u) === '/api/v1/resolve?app=teams.exe')).toBe(true)

    await user.click(screen.getByRole('button', { name: 'Limpiar' }))
    expect(input).toHaveValue('')
    expect(screen.queryByText('app:teams.exe sale por tu conexión normal.')).not.toBeInTheDocument()
    expect(screen.getByText('2 reglas')).toBeInTheDocument()
  })

  it('says a rule’s traffic is blocked while its VPN is off (on_tunnel_down: block)', async () => {
    stubFetch(api({ 'GET /api/v1/resolve?target=10.1.2.3': () => json({ target: '10.1.2.3', via: 'office', rule: 'corp', matched: true }) }))
    renderPage()
    const user = userEvent.setup()
    await user.type(await screen.findByRole('textbox', { name: 'Sitio, IP o programa' }), '10.1.2.3{Enter}')
    expect(await screen.findByText('10.1.2.3 está bloqueado mientras office esté desconectada.')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Ir a Inicio' })).toHaveAttribute('href', '/')
  })

  it('shows each rule’s resolved IPs with their expiry, on demand', async () => {
    stubFetch(api())
    renderPage()
    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: 'Ver 1 IP de ai' }))
    const rules = screen.getByRole('table', { name: 'Tus reglas' })
    expect(within(rules).getByText('1.1.1.1/32')).toBeInTheDocument()
    expect(within(rules).getByText(/caduca/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Ocultar IPs de ai' })).toHaveAttribute('aria-expanded', 'true')
  })

  it('live ticks re-render only the tunnel that changed; open/closed panels survive', async () => {
    stubFetch(api())
    const renders: string[] = []
    const client = renderPage((name) => renders.push(name))
    const user = userEvent.setup()
    const toggle = await screen.findByRole('button', { name: 'Qué sale por este túnel (2)' })
    await user.click(toggle)
    expect(toggle).toHaveAttribute('aria-expanded', 'false')

    renders.length = 0
    const homeSame = (client.getQueryData<TunnelMetrics[]>(keys.tunnels) ?? []).find((t) => t.Name === 'home')!
    act(() => {
      client.setQueryData(keys.tunnels, [home(0), tunnel('tv', 'disconnected'), work(5)].sort((a, b) => a.Name.localeCompare(b.Name)))
    })
    expect(renders).toContain('work')
    expect(renders).not.toContain('home')
    expect(homeSame.UpdatedAt).toBe(at(0))
    expect(screen.getByRole('button', { name: 'Qué sale por este túnel (2)' })).toHaveAttribute('aria-expanded', 'false')

    // Two frames give a chart with a text alternative.
    act(() => {
      client.setQueryData(keys.tunnels, [home(0), tunnel('tv', 'disconnected'), work(10)].sort((a, b) => a.Name.localeCompare(b.Name)))
    })
    const chart = await screen.findByRole('slider', { name: 'Recibido' })
    expect(chart).toHaveAttribute('aria-valuetext', expect.stringContaining('1 kB/s'))
    expect(screen.getByText(/Recibido: ahora 1 kB\/s; máximo 1 kB\/s/)).toBeInTheDocument()
  })

  it('empty states teach the next step', async () => {
    stubFetch(
      api({
        'GET /api/v1/tunnels': () => json([tunnel('tv', 'disconnected')]),
        'GET /api/v1/policies': () => json([]),
        'GET /api/v1/tunnels/routes': () => json({ rest_via: 'Wi-Fi', tunnels: [] }),
      }),
    )
    renderPage()
    expect(await screen.findByRole('heading', { name: 'Ninguna VPN conectada' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Ir a Inicio' })).toHaveAttribute('href', '/')
    expect(await screen.findByRole('heading', { name: 'Todavía no tienes reglas' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Ir a Reglas' })).toHaveAttribute('href', '/rules')
  })

  it('a section the daemon refuses says what to do and can retry', async () => {
    stubFetch(api({ 'GET /api/v1/policies': () => json({ error: 'config unreadable', hint: 'Qué hacer: revisa kongtrol.yaml.' }, 400) }))
    renderPage()
    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('config unreadable')
    expect(alert).toHaveTextContent('Qué hacer: revisa kongtrol.yaml.')
    expect(within(alert).getByRole('button', { name: 'Reintentar' })).toBeInTheDocument()
  })

  it('system routes stay collapsed (and unread) until opened; the filter keeps a valid choice', async () => {
    const fetchMock = stubFetch(
      api({
        'GET /api/v1/routes': () =>
          json([
            { destination: '0.0.0.0/0', gateway: '192.168.1.1', interface: 'Wi-Fi', metric: 25, is_default: true, policy_name: 'default', policy_via: 'system' },
            { destination: '10.8.0.0/24', gateway: '', interface: 'tun0', metric: 1, is_default: false },
          ]),
        'GET /api/v1/network/overview': () =>
          json({ connected_tunnels: 2, default_routes: [{ destination: '0.0.0.0/0', gateway: '192.168.1.1', interface: 'Wi-Fi', metric: 25 }], local_ips: ['192.168.1.20'], public_ip_check_enabled: false }),
      }),
    )
    renderPage()
    const user = userEvent.setup()
    const toggle = await screen.findByRole('button', { name: 'Ver rutas del sistema' })
    expect(toggle).toHaveAttribute('aria-expanded', 'false')
    expect(fetchMock.mock.calls.some(([u]) => String(u) === '/api/v1/routes')).toBe(false)

    await user.click(toggle)
    expect(await screen.findByText('2 de 2 rutas')).toBeInTheDocument()
    expect(screen.getByText('Wi-Fi (métrica 25)')).toBeInTheDocument()
    expect(screen.getByText('Desactivada (actívala en Seguridad)')).toBeInTheDocument()
    await pickOption(user, screen.getByRole('button', { name: /Mostrar/ }), /work/)
    expect(screen.getByText('1 de 2 rutas')).toBeInTheDocument()
    expect(screen.getByText('work · tun0')).toBeInTheDocument()
  })
})
