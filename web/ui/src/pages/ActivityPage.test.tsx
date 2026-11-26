import { act, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { App } from '../App'
import { activityKeys, auditToJSONL, auditURL, mergeEvents, nextCursor, signatureState, type AuditEvent, type DaemonEvent } from '../data/activity'
import { createQueryClient } from '../data/queryClient'
import { presetRange, toISODay } from '../components/ui/dates'
import { FakeSocket, helperReady, json, pickOption, stubFetch } from '../test/fakes'

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
  audit_log_path: '/home/ana/.kongtrol/audit.log',
  audit_log_max_size_mb: 100,
  audit_log_retention_days: 30,
  audit_log_max_total_mb: 500,
  audit_log_sign: true,
}

const now = new Date().toISOString()
const ev = (seq: number, key: string, args: string[], extra: Partial<DaemonEvent> = {}): DaemonEvent => ({
  seq,
  time: now,
  kind: 'tunnel_watchdog',
  level: 'INFO',
  key,
  args,
  ...extra,
})
const audit = (id: string, level: string, extra: Partial<AuditEvent> = {}): AuditEvent => ({
  id,
  timestamp: now,
  level,
  action: 'vpn.connect',
  profile: 'trabajo',
  message: `entry ${id}`,
  hmac: 'ab12',
  ...extra,
})

const base = {
  'GET /api/v1/preferences': () => json({ language: '', theme: '' }),
  'GET /api/v1/tunnels': () => json([]),
  'GET /api/v1/helper/status': () => json(helperReady),
  'GET /api/v1/settings': () => json(settings),
}

describe('Actividad · avisos', () => {
  it('offers Reintentar once the watchdog gave up (ADR-0098)', async () => {
    const fetchMock = stubFetch({
      ...base,
      'GET /api/v1/events?after=0': () =>
        json({
          seq: 1,
          events: [ev(1, 'watchdog.event.gave_up_blocking_full', ['trabajo', '3'], { level: 'WARN', profile: 'trabajo', can_connect: true })],
        }),
      'POST /api/v1/tunnels/trabajo/connect': () => json({ status: 'connecting', tunnel: 'trabajo' }, 202),
    })
    window.history.replaceState(null, '', '/activity')
    render(<App />)

    const list = await screen.findByRole('region', { name: 'Avisos, del más nuevo al más viejo' })
    const [item] = within(list).getAllByRole('listitem')
    expect(item).toHaveTextContent('No se pudo reconectar trabajo tras 3 intentos')
    expect(item).toHaveTextContent('Para volver a tener internet, desconéctala')
    await userEvent.click(within(item!).getByRole('button', { name: 'Reintentar trabajo' }))
    await waitFor(() =>
      expect(fetchMock.mock.calls.some(([u, i]) => String(u) === '/api/v1/tunnels/trabajo/connect' && i?.method === 'POST')).toBe(true),
    )
  })

  it('renders daemon notices in plain language, newest first, and offers Conectar', async () => {
    const fetchMock = stubFetch({
      ...base,
      'GET /api/v1/events?after=0': () =>
        json({
          seq: 2,
          events: [
            ev(1, 'watchdog.event.reconnected', ['casa']),
            ev(2, 'policy.event.connect_exhausted', ['intranet.example', 'trabajo'], { level: 'WARN', profile: 'trabajo', can_connect: true }),
          ],
        }),
      'POST /api/v1/tunnels/trabajo/connect': () => json({ status: 'connecting', tunnel: 'trabajo' }, 202),
    })
    window.history.replaceState(null, '', '/activity')
    render(<App />)

    const list = await screen.findByRole('region', { name: 'Avisos, del más nuevo al más viejo' })
    const items = within(list).getAllByRole('listitem')
    expect(items[0]).toHaveTextContent('No se pudo conectar trabajo para intranet.example tras tres intentos automáticos. Conecta el perfil manualmente.')
    expect(items[0]).toHaveTextContent('Aviso')
    expect(items[1]).toHaveTextContent('casa se reconectó')
    expect(within(list).getByRole('heading', { name: 'Hoy' })).toBeInTheDocument()

    await userEvent.click(within(items[0]!).getByRole('button', { name: 'Conectar trabajo' }))
    await waitFor(() =>
      expect(fetchMock.mock.calls.some(([u, i]) => String(u) === '/api/v1/tunnels/trabajo/connect' && i?.method === 'POST')).toBe(true),
    )
  })

  it('live tail asks only for what is new and appends it', async () => {
    const client = createQueryClient()
    const fetchMock = stubFetch({
      ...base,
      'GET /api/v1/events?after=0': () => json({ seq: 1, events: [ev(1, 'watchdog.event.reconnected', ['casa'])] }),
      'GET /api/v1/events?after=1': () => json({ seq: 2, events: [ev(2, 'watchdog.event.no_response', ['trabajo'], { level: 'WARN' })] }),
    })
    window.history.replaceState(null, '', '/activity')
    render(<App client={client} />)
    expect(await screen.findByText('casa se reconectó')).toBeInTheDocument()

    await act(() => client.refetchQueries({ queryKey: activityKeys.events }))
    expect(await screen.findByText(/trabajo dejó de responder/)).toBeInTheDocument()
    expect(screen.getByText('casa se reconectó')).toBeInTheDocument()
    expect(fetchMock.mock.calls.some(([u]) => String(u) === '/api/v1/events?after=1')).toBe(true)
  })

  it('empty feed teaches what will show up, centred with an icon', async () => {
    stubFetch({ ...base, 'GET /api/v1/events?after=0': () => json({ seq: 0, events: [] }) })
    window.history.replaceState(null, '', '/activity')
    render(<App />)
    const heading = await screen.findByRole('heading', { name: 'Sin avisos por ahora' })
    expect(heading.closest('.k-empty')).toHaveClass('k-empty--center')
    expect(heading.closest('.k-empty')!.querySelector('.k-empty__icon svg')).not.toBeNull()
  })
})

/** An audit page: the JSON array plus the paging header the daemon sends. */
const page = (events: AuditEvent[], more = false) =>
  Promise.resolve(new Response(JSON.stringify(events), { status: 200, headers: { 'X-Kongtrol-Audit-More': String(more) } }))

describe('Actividad · registro de auditoría', () => {
  const all = [
    audit('1', 'INFO', { signed: true, verified: true }),
    audit('2', 'WARN', { hmac: '', signed: false, verified: false }),
    audit('3', 'SECURITY', { profile: 'casa', signed: true, verified: false }),
  ]
  const auditRoutes = {
    ...base,
    'GET /api/v1/events?after=0': () => json({ seq: 0, events: [] }),
    'GET /api/v1/audit?limit=50': () => page(all),
    'GET /api/v1/audit?limit=50&level=WARN': () => page([all[1]!]),
    'GET /api/v1/audit?limit=50&profile=casa': () => page([all[2]!]),
  }

  it('shows the verified signature state per entry and filters by level at once (no Apply)', async () => {
    const fetchMock = stubFetch(auditRoutes)
    window.history.replaceState(null, '', '/activity?tab=audit')
    render(<App />)

    expect(await screen.findByText('Mostrando 3 entradas')).toBeInTheDocument()
    expect(screen.getByText('Firma activada')).toBeInTheDocument()
    expect(screen.getByText('Firma válida')).toBeInTheDocument()
    expect(screen.getByText('Firma no válida')).toBeInTheDocument()
    expect(screen.getByText('Sin firma')).toBeInTheDocument()
    expect(screen.getByText('1 entradas con firma no válida en lo que ves')).toBeInTheDocument()
    // translated level names, never raw INFO/WARN
    expect(screen.getAllByText('Seguridad').length).toBeGreaterThan(0)

    await pickOption(userEvent.setup(), screen.getByRole('button', { name: /Nivel/ }), 'Aviso')
    expect(await screen.findByText('Mostrando 1 entradas')).toBeInTheDocument()
    expect(fetchMock.mock.calls.some(([u]) => String(u) === '/api/v1/audit?limit=50&level=WARN')).toBe(true)
  })

  it('pages with the before cursor while the daemon says there is more', async () => {
    const older = audit('0', 'INFO', { timestamp: '2026-11-01T08:00:00Z', message: 'older entry' })
    const first = [audit('9', 'INFO', { timestamp: '2026-11-17T10:00:00Z' })]
    stubFetch({
      ...auditRoutes,
      'GET /api/v1/audit?limit=50': () => page(first, true),
      'GET /api/v1/audit?limit=50&before=2026-11-17T10%3A00%3A00Z': () => page([older], false),
    })
    window.history.replaceState(null, '', '/activity?tab=audit')
    render(<App />)
    expect(await screen.findByText('Mostrando las 1 más nuevas · hay más antiguas')).toBeInTheDocument()
    // The loaded entries ran out: "Ver 20 más" asks the daemon for the next older page.
    await userEvent.click(screen.getByRole('button', { name: 'Ver 20 más' }))
    expect(await screen.findByText('older entry')).toBeInTheDocument()
    expect(screen.getByText('Mostrando 2 entradas')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /^Ver \d+ más$/ })).not.toBeInTheDocument()
  })

  it('shows 10, then 20 more; live entries wait behind "N nuevas · Mostrar"', async () => {
    const client = createQueryClient()
    const at = (i: number) => new Date(Date.UTC(2026, 10, 20, 12, 0, 0) - i * 60_000).toISOString()
    let entries = Array.from({ length: 25 }, (_, i) => audit(String(i), 'INFO', { timestamp: at(i), message: `entry ${i}` }))
    stubFetch({ ...auditRoutes, 'GET /api/v1/audit?limit=50': () => page(entries) })
    window.history.replaceState(null, '', '/activity?tab=audit')
    render(<App client={client} />)
    expect(await screen.findByText('Mostrando 10 de 25 entradas')).toBeInTheDocument()
    expect(screen.getByText('entry 9')).toBeInTheDocument()
    expect(screen.queryByText('entry 10')).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Ver 15 más' }))
    expect(await screen.findByText('entry 24')).toBeInTheDocument()
    expect(screen.getByText('Mostrando 25 entradas')).toBeInTheDocument()

    // A new entry arrives on a live read: it waits, the rows on screen stay.
    entries = [audit('new', 'WARN', { timestamp: at(-1), message: 'brand new entry' }), ...entries]
    await act(() => client.refetchQueries({ queryKey: ['audit'] }))
    expect(await screen.findByText('1 nueva')).toBeInTheDocument()
    expect(screen.queryByText('brand new entry')).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Mostrar' }))
    expect(await screen.findByText('brand new entry')).toBeInTheDocument()
    expect(screen.getByText('entry 24')).toBeInTheDocument()
    expect(screen.queryByText('1 nueva')).not.toBeInTheDocument()
  })

  it('filters dates on the daemon with one range control: a preset at once, then Clear', async () => {
    const week = presetRange('last7', toISODay(new Date()))
    const fetchMock = stubFetch({
      ...auditRoutes,
      [`GET /api/v1/audit?limit=50&from=${week.from}&to=${week.to}`]: () => page([all[0]!]),
    })
    window.history.replaceState(null, '', '/activity?tab=audit')
    render(<App />)
    await screen.findByText('Mostrando 3 entradas')
    const dates = screen.getByRole('button', { name: 'Fechas' })
    expect(dates).toHaveTextContent('Todas las fechas')
    await userEvent.click(dates)
    await userEvent.click(within(screen.getByRole('dialog', { name: 'Elegir fechas' })).getByRole('button', { name: 'Últimos 7 días' }))
    expect(await screen.findByText('Mostrando 1 entradas')).toBeInTheDocument()
    expect(fetchMock.mock.calls.some(([u]) => String(u) === `/api/v1/audit?limit=50&from=${week.from}&to=${week.to}`)).toBe(true)
    expect(dates).toHaveTextContent('Últimos 7 días')
    await userEvent.click(dates)
    await userEvent.click(screen.getByRole('button', { name: 'Borrar fechas' }))
    expect(await screen.findByText('Mostrando 3 entradas')).toBeInTheDocument()
  })

  it('filters by several profiles: search, pick, chips, clear', async () => {
    const user = userEvent.setup()
    const fetchMock = stubFetch({
      ...auditRoutes,
      'GET /api/v1/audit?limit=50&profile=casa&profile=trabajo': () => page(all),
    })
    window.history.replaceState(null, '', '/activity?tab=audit')
    render(<App />)
    await screen.findByText('Mostrando 3 entradas')
    const field = screen.getByRole('button', { name: /^Perfil/ })
    expect(field).toHaveTextContent('Todos los perfiles')
    await user.click(field)
    // The search field takes focus; typing filters the list.
    const search = screen.getByRole('searchbox', { name: 'Buscar perfil' })
    await waitFor(() => expect(search).toHaveFocus())
    await user.keyboard('cas')
    const list = screen.getByRole('listbox', { name: 'Buscar perfil' })
    expect(within(list).getAllByRole('option').map((o) => o.textContent)).toEqual(['casa'])
    await user.keyboard('{ArrowDown}{Enter}')
    expect(await screen.findByText('Mostrando 1 entradas')).toBeInTheDocument()
    expect(screen.getByText('entry 3')).toBeInTheDocument()
    // A second profile: an entry of any of them matches (profile repeats).
    await user.clear(search)
    await user.click(within(screen.getByRole('listbox', { name: 'Buscar perfil' })).getByRole('option', { name: 'trabajo' }))
    expect(await screen.findByText('Mostrando 3 entradas')).toBeInTheDocument()
    expect(fetchMock.mock.calls.some(([u]) => String(u) === '/api/v1/audit?limit=50&profile=casa&profile=trabajo')).toBe(true)
    await user.keyboard('{Escape}')
    expect(field).toHaveTextContent('2 elegidos')
    const chips = screen.getByRole('grid', { name: 'Perfiles elegidos' })
    expect(within(chips).getAllByRole('row').map((r) => r.textContent)).toEqual(['casa', 'trabajo'])
    await user.click(within(chips).getByRole('button', { name: /Quitar trabajo/ }))
    expect(await screen.findByText('Mostrando 1 entradas')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Quitar todos' }))
    expect(await screen.findByText('Mostrando 3 entradas')).toBeInTheDocument()
    expect(field).toHaveTextContent('Todos los perfiles')
  })

  it('explains an unconfigured log instead of "no events"', async () => {
    stubFetch({ ...auditRoutes, 'GET /api/v1/settings': () => json({ ...settings, audit_log_path: '' }) })
    window.history.replaceState(null, '', '/activity?tab=audit')
    render(<App />)
    expect(await screen.findByRole('heading', { name: 'El registro de auditoría está apagado' })).toBeInTheDocument()
  })

  it('reports a read error with what to do, not "not configured"', async () => {
    stubFetch({
      ...auditRoutes,
      'GET /api/v1/audit?limit=50': () => json({ error: 'permiso denegado', hint: 'Revisa los permisos del archivo.' }, 400),
    })
    window.history.replaceState(null, '', '/activity?tab=audit')
    render(<App />)
    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('permiso denegado')
    expect(alert).toHaveTextContent('Revisa los permisos del archivo.')
  })

  it('refuses an empty max size or bad days instead of saving them; saves size and retention', async () => {
    const fetchMock = stubFetch({ ...auditRoutes, 'PUT /api/v1/settings': () => json(settings) })
    window.history.replaceState(null, '', '/activity?tab=audit')
    render(<App />)
    const size = await screen.findByLabelText('Tamaño máximo por archivo (MB)')
    const days = screen.getByLabelText('Días que se guardan')
    expect(days).toHaveValue('30')
    expect(screen.getByLabelText('Espacio máximo en total (MB)')).toHaveValue('500')
    expect(screen.getByText('Se guardan los últimos 30 días, contando hoy; los más viejos se borran solos cada medianoche.')).toBeInTheDocument()
    await userEvent.clear(size)
    await userEvent.clear(days)
    await userEvent.type(days, '400')
    await userEvent.click(screen.getByRole('button', { name: 'Guardar' }))
    expect(await screen.findByText('Escribe un tamaño.')).toBeInTheDocument()
    expect(screen.getByText('Esa cantidad de días no vale.')).toBeInTheDocument()
    expect(size).toHaveAttribute('aria-invalid', 'true')
    expect(fetchMock.mock.calls.some(([, i]) => i?.method === 'PUT')).toBe(false)

    await userEvent.clear(days)
    await userEvent.type(days, '0')
    await userEvent.click(screen.getByRole('button', { name: 'Guardar' }))
    expect(await screen.findByText('Esa cantidad de días no vale.')).toBeInTheDocument()

    // The total can never be smaller than one file.
    const total = screen.getByLabelText('Espacio máximo en total (MB)')
    await userEvent.type(size, '50')
    await userEvent.clear(total)
    await userEvent.type(total, '20')
    await userEvent.click(screen.getByRole('button', { name: 'Guardar' }))
    expect(await screen.findByText('El espacio total es menor que un solo archivo.')).toBeInTheDocument()
    expect(fetchMock.mock.calls.some(([, i]) => i?.method === 'PUT')).toBe(false)

    await userEvent.clear(total)
    await userEvent.type(total, '400')
    await userEvent.clear(days)
    await userEvent.type(days, '1')
    expect(screen.getByText('Solo se guarda hoy; a medianoche se borra el día anterior.')).toBeInTheDocument()
    await userEvent.clear(days)
    await userEvent.type(days, '30')
    expect(screen.getByText('Se guardan los últimos 30 días, contando hoy; los más viejos se borran solos cada medianoche.')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Guardar' }))
    await waitFor(() => expect(fetchMock.mock.calls.some(([, i]) => i?.method === 'PUT')).toBe(true))
    const put = fetchMock.mock.calls.find(([, i]) => i?.method === 'PUT')
    expect(JSON.parse(String(put?.[1]?.body))).toEqual({
      audit_log_path: settings.audit_log_path,
      audit_log_max_size_mb: 50,
      audit_log_retention_days: 30,
      audit_log_max_total_mb: 400,
      audit_log_sign: true,
    })
  })
})

describe('activity data helpers', () => {
  it('builds the audit URL with only the filters in use', () => {
    expect(auditURL({ profiles: [' '], level: '', from: '', to: '' })).toBe('/api/v1/audit?limit=50')
    expect(auditURL({ profiles: ['a b', 'c'], level: 'ERROR', from: '2026-11-01', to: '2026-11-02' }, '2026-11-02T10:00:00Z')).toBe(
      '/api/v1/audit?limit=50&profile=a+b&profile=c&level=ERROR&from=2026-11-01&to=2026-11-02&before=2026-11-02T10%3A00%3A00Z',
    )
  })

  it('cursor', () => {
    expect(nextCursor({ events: [audit('1', 'INFO', { timestamp: 't1' })], more: true })).toBe('t1')
    expect(nextCursor({ events: [audit('1', 'INFO')], more: false })).toBeUndefined()
    expect(nextCursor({ events: [], more: true })).toBeUndefined()
  })

  it('signature states: valid only when the daemon verified it', () => {
    expect(signatureState({ hmac: '', signed: false })).toBe('unsigned')
    expect(signatureState({ hmac: 'x', signed: true, verified: true })).toBe('verified')
    expect(signatureState({ hmac: 'x', signed: true, verified: false })).toBe('tampered')
    expect(signatureState({ hmac: 'x' })).toBe('tampered')
  })


  it('merges notices without duplicates, newest first, capped', () => {
    const a = ev(1, 'k', [])
    const b = ev(2, 'k', [])
    const merged = mergeEvents([a], [a, b])
    expect(merged.map((e) => e.seq)).toEqual([2, 1])
    expect(merged[1]).toBe(a)
    expect(mergeEvents([a, b], [ev(3, 'k', [])], 2).map((e) => e.seq)).toEqual([3, 2])
  })

  it('exports one JSON object per line', () => {
    const text = auditToJSONL([audit('1', 'INFO'), audit('2', 'WARN')])
    expect(text.trim().split('\n').map((l) => JSON.parse(l).id)).toEqual(['1', '2'])
    expect(auditToJSONL([])).toBe('')
  })
})
