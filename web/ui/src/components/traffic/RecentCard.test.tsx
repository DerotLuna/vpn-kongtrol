// Recientes (ADR-0106): 10 rows + "Ver 20 más", search and "Salió por"
// filters, live updates held behind "N nuevas · Mostrar".
import { act, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router'
import { beforeEach, describe, expect, it } from 'vitest'
import { PreferencesProvider } from '../../app/PreferencesProvider'
import { createQueryClient } from '../../data/queryClient'
import type { RecentConnection } from '../../data/recent'
import { json, pickOption, stubFetch } from '../../test/fakes'
import { RecentCard } from './RecentCard'
import { matchesRecent, recentKey } from './recentView'

const URL_RECENT = 'GET /api/v1/connections/recent?limit=500'
const now = new Date().toISOString()

function conn(i: number, extra: Partial<RecentConnection> = {}): RecentConnection {
  return { host: `site${i}.example`, remote_ip: `10.0.0.${i}`, port: 443, first_seen: now, last_seen: now, count: 1, ...extra }
}

function body(connections: RecentConnection[]) {
  return { supported: true, enabled: true, max: 500, ttl_seconds: 3600, connections }
}

let list: RecentConnection[] = []

function renderCard() {
  stubFetch({
    'GET /api/v1/preferences': () => json({ language: '', theme: '' }),
    [URL_RECENT]: () => json(body(list)),
  })
  const client = createQueryClient()
  render(
    <PreferencesProvider>
      <QueryClientProvider client={client}>
        <MemoryRouter initialEntries={['/traffic']}>
          <RecentCard />
        </MemoryRouter>
      </QueryClientProvider>
    </PreferencesProvider>,
  )
  return client
}

const rowNames = () =>
  within(screen.getByRole('table'))
    .getAllByRole('rowheader')
    .map((th) => th.textContent)

describe('RecentCard', () => {
  beforeEach(() => {
    Object.defineProperty(navigator, 'languages', { value: ['es-AR'], configurable: true })
  })

  it('shows 10 rows, then 20 more per click, and hides the button when none remain', async () => {
    list = Array.from({ length: 35 }, (_, i) => conn(i))
    const user = userEvent.setup()
    renderCard()
    await screen.findByRole('table')
    expect(rowNames()).toHaveLength(10)
    await user.click(screen.getByRole('button', { name: 'Ver 20 más (quedan 25)' }))
    expect(rowNames()).toHaveLength(30)
    expect(rowNames()[0]).toBe('site0.example')
    await user.click(screen.getByRole('button', { name: 'Ver 5 más (quedan 5)' }))
    expect(rowNames()).toHaveLength(35)
    expect(screen.queryByRole('button', { name: /^Ver \d+ más/ })).not.toBeInTheDocument()
  })

  it('filters by site, IP or program and by the way out, with an empty state that says what to do', async () => {
    list = [
      conn(1, { host: 'github.com', via: 'office' }),
      conn(2, { host: '', process: 'claude.exe.old.1795215272638.13036' }),
      conn(3, { host: 'intranet.example', via: 'office' }),
      conn(4, { host: 'news.example', via: 'home' }),
    ]
    const user = userEvent.setup()
    renderCard()
    await screen.findByRole('table')

    const search = screen.getByRole('searchbox', { name: 'Buscar sitio, IP o programa' })
    await user.type(search, 'CLAUDE.exe')
    expect(rowNames()).toEqual(['10.0.0.2'])
    // The long program name keeps its full text in the tooltip.
    expect(screen.getByTitle('claude.exe.old.1795215272638.13036')).toBeInTheDocument()

    await user.clear(search)
    await user.type(search, '10.0.0.3')
    expect(rowNames()).toEqual(['intranet.example'])

    await user.clear(search)
    await pickOption(user, screen.getByRole('button', { name: /Salió por/ }), 'office')
    expect(rowNames()).toEqual(['github.com', 'intranet.example'])
    await pickOption(user, screen.getByRole('button', { name: /Salió por/ }), 'Conexión normal')
    expect(rowNames()).toEqual(['10.0.0.2'])

    await user.type(search, 'nada-parecido')
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
    expect(screen.getByText('Ningún destino coincide con el filtro.')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Quitar filtros' }))
    expect(rowNames()).toHaveLength(4)
  })

  it('each row offers "Agregar como regla" with a tooltip that names the destination', async () => {
    list = [conn(1), conn(2, { host: '' })]
    renderCard()
    await screen.findByRole('table')
    const buttons = screen.getAllByRole('button', { name: 'Agregar como regla' })
    expect(buttons).toHaveLength(2)
    expect(buttons[0]).toHaveAccessibleDescription('Crea una regla para que site1.example salga por el perfil VPN que elijas.')
    expect(buttons[1]).toHaveAccessibleDescription('Crea una regla para que 10.0.0.2 salga por el perfil VPN que elijas.')
    expect(buttons[0]).toHaveAttribute('aria-haspopup', 'dialog')
    expect(screen.queryByText(/Agregar regla de salida para esto/)).not.toBeInTheDocument()
  })

  it('keeps the rows on screen during live updates and merges new ones on top when asked', async () => {
    list = Array.from({ length: 3 }, (_, i) => conn(i))
    const user = userEvent.setup()
    const client = renderCard()
    await screen.findByRole('table')
    expect(rowNames()).toEqual(['site0.example', 'site1.example', 'site2.example'])
    const notice = screen.getByRole('status')
    expect(notice).toBeEmptyDOMElement()

    // Next tick: two new entries first, an old one seen again moves up.
    list = [conn(8), conn(9), conn(2, { count: 5 }), conn(0), conn(1)]
    await act(() => client.refetchQueries())
    await waitFor(() => expect(notice).toHaveTextContent('2 nuevas'))
    expect(rowNames()).toEqual(['site0.example', 'site1.example', 'site2.example'])

    await user.click(within(notice).getByRole('button', { name: 'Mostrar' }))
    expect(rowNames()).toEqual(['site8.example', 'site9.example', 'site2.example', 'site0.example', 'site1.example'])
    expect(notice).toBeEmptyDOMElement()
  })
})

describe('recentView', () => {
  it('matches like `kongtrol traffic recent --search/--via`', () => {
    const c = conn(1, { host: 'GitHub.com', via: 'office', process: 'git.exe' })
    expect(matchesRecent(c, 'github', 'all')).toBe(true)
    expect(matchesRecent(c, 'GIT.EXE', 'p:office')).toBe(true)
    expect(matchesRecent(c, '10.0.0.1', 'all')).toBe(true)
    expect(matchesRecent(c, '', 'direct')).toBe(false)
    expect(matchesRecent(c, '', 'p:home')).toBe(false)
    expect(matchesRecent(conn(2), '', 'direct')).toBe(true)
  })

  it('keys an entry like the daemon (destination, program, interface)', () => {
    expect(recentKey(conn(1, { process: 'a' }))).not.toBe(recentKey(conn(1, { process: 'b' })))
    expect(recentKey(conn(1, { interface: 'wg0' }))).not.toBe(recentKey(conn(1)))
  })
})
