import { act, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { App } from '../App'
import { activityKeys, auditToJSONL, auditURL, inRange, mergeEvents, signatureState, type AuditEvent, type DaemonEvent } from '../data/activity'
import { createQueryClient } from '../data/queryClient'
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
  audit_log_path: '/home/ana/.kongtrol/audit.log',
  audit_log_max_size_mb: 100,
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
  it('renders daemon notices in plain language, newest first, and offers Conectar', async () => {
    const fetchMock = stubFetch({
      ...base,
      'GET /api/v1/events?after=0': () =>
        json({
          seq: 2,
          events: [
            ev(1, 'watchdog.event.reconnected', ['casa']),
            ev(2, 'policy.event.allow', ['intranet.example', 'trabajo'], { level: 'WARN', profile: 'trabajo', can_connect: true }),
          ],
        }),
      'POST /api/v1/tunnels/trabajo/connect': () => json({ status: 'connecting', tunnel: 'trabajo' }, 202),
    })
    window.history.replaceState(null, '', '/activity')
    render(<App />)

    const list = await screen.findByRole('region', { name: 'Avisos, del más nuevo al más viejo' })
    const items = within(list).getAllByRole('listitem')
    expect(items[0]).toHaveTextContent('intranet.example sale por internet normal · trabajo está apagado')
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

  it('empty feed teaches what will show up', async () => {
    stubFetch({ ...base, 'GET /api/v1/events?after=0': () => json({ seq: 0, events: [] }) })
    window.history.replaceState(null, '', '/activity')
    render(<App />)
    expect(await screen.findByRole('heading', { name: 'Sin avisos por ahora' })).toBeInTheDocument()
  })
})

describe('Actividad · registro de auditoría', () => {
  const auditRoutes = {
    ...base,
    'GET /api/v1/events?after=0': () => json({ seq: 0, events: [] }),
    'GET /api/v1/audit?limit=300': () =>
      json([audit('1', 'INFO'), audit('2', 'WARN', { hmac: '' }), audit('3', 'SECURITY', { profile: 'casa' })]),
    'GET /api/v1/audit?limit=300&level=WARN': () => json([audit('2', 'WARN', { hmac: '' })]),
    'GET /api/v1/audit?limit=300&profile=casa': () => json([audit('3', 'SECURITY', { profile: 'casa' })]),
  }

  it('shows signing state per entry and filters by level at once (no Apply)', async () => {
    const fetchMock = stubFetch(auditRoutes)
    window.history.replaceState(null, '', '/activity?tab=audit')
    render(<App />)

    expect(await screen.findByText('Mostrando 3 entradas')).toBeInTheDocument()
    expect(screen.getByText('Firma activada')).toBeInTheDocument()
    expect(screen.getAllByText('Firmada · sin comprobar')).toHaveLength(2)
    expect(screen.getByText('Sin firma')).toBeInTheDocument()
    // translated level names, never raw INFO/WARN
    expect(screen.getAllByText('Seguridad').length).toBeGreaterThan(0)

    await userEvent.selectOptions(screen.getByLabelText('Nivel'), 'WARN')
    expect(await screen.findByText('Mostrando 1 entradas')).toBeInTheDocument()
    expect(fetchMock.mock.calls.some(([u]) => String(u) === '/api/v1/audit?limit=300&level=WARN')).toBe(true)
  })

  it('filters by profile on Enter', async () => {
    stubFetch(auditRoutes)
    window.history.replaceState(null, '', '/activity?tab=audit')
    render(<App />)
    await screen.findByText('Mostrando 3 entradas')
    await userEvent.type(screen.getByLabelText('Perfil'), 'casa{Enter}')
    expect(await screen.findByText('Mostrando 1 entradas')).toBeInTheDocument()
    expect(screen.getByText('entry 3')).toBeInTheDocument()
  })

  it('explains an unconfigured log instead of "no events"', async () => {
    stubFetch({ ...auditRoutes, 'GET /api/v1/settings': () => json({ ...settings, audit_log_path: '' }), 'GET /api/v1/audit?limit=300': () => json([]) })
    window.history.replaceState(null, '', '/activity?tab=audit')
    render(<App />)
    expect(await screen.findByRole('heading', { name: 'El registro de auditoría está apagado' })).toBeInTheDocument()
  })

  it('reports a read error with what to do, not "not configured"', async () => {
    stubFetch({
      ...auditRoutes,
      'GET /api/v1/audit?limit=300': () => json({ error: 'permiso denegado', hint: 'Revisa los permisos del archivo.' }, 400),
    })
    window.history.replaceState(null, '', '/activity?tab=audit')
    render(<App />)
    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('permiso denegado')
    expect(alert).toHaveTextContent('Revisa los permisos del archivo.')
  })

  it('refuses an empty max size instead of saving 0', async () => {
    const fetchMock = stubFetch({ ...auditRoutes, 'PUT /api/v1/settings': () => json(settings) })
    window.history.replaceState(null, '', '/activity?tab=audit')
    render(<App />)
    const size = await screen.findByLabelText('Tamaño máximo (MB)')
    await userEvent.clear(size)
    await userEvent.click(screen.getByRole('button', { name: 'Guardar' }))
    expect(await screen.findByText('Escribe un tamaño.')).toBeInTheDocument()
    expect(size).toHaveAttribute('aria-invalid', 'true')
    expect(fetchMock.mock.calls.some(([, i]) => i?.method === 'PUT')).toBe(false)

    await userEvent.type(size, '50')
    await userEvent.click(screen.getByRole('button', { name: 'Guardar' }))
    await waitFor(() => expect(fetchMock.mock.calls.some(([, i]) => i?.method === 'PUT')).toBe(true))
    const put = fetchMock.mock.calls.find(([, i]) => i?.method === 'PUT')
    expect(JSON.parse(String(put?.[1]?.body))).toEqual({
      audit_log_path: settings.audit_log_path,
      audit_log_max_size_mb: 50,
      audit_log_sign: true,
    })
  })
})

describe('activity data helpers', () => {
  it('builds the audit URL with only the filters in use', () => {
    expect(auditURL({ profile: ' ', level: '', limit: 300 })).toBe('/api/v1/audit?limit=300')
    expect(auditURL({ profile: 'a b', level: 'ERROR', limit: 600 })).toBe('/api/v1/audit?limit=600&profile=a+b&level=ERROR')
  })

  it('date ranges', () => {
    const at = new Date(2026, 10, 18, 12, 0, 0)
    const iso = (d: Date) => ({ timestamp: d.toISOString() })
    expect(inRange(iso(new Date(2026, 10, 18, 11, 30)), 'hour', at)).toBe(true)
    expect(inRange(iso(new Date(2026, 10, 18, 10, 0)), 'hour', at)).toBe(false)
    expect(inRange(iso(new Date(2026, 10, 18, 0, 1)), 'today', at)).toBe(true)
    expect(inRange(iso(new Date(2026, 10, 17, 23, 59)), 'today', at)).toBe(false)
    expect(inRange(iso(new Date(2026, 10, 17, 23, 59)), 'yesterday', at)).toBe(true)
    expect(inRange(iso(new Date(2026, 10, 16, 23, 59)), 'yesterday', at)).toBe(false)
    expect(inRange({ timestamp: 'nope' }, 'all', at)).toBe(true)
  })

  it('signature states never claim "valid" without a verification', () => {
    expect(signatureState({ hmac: '' })).toBe('unsigned')
    expect(signatureState({ hmac: 'x' })).toBe('signed')
    expect(signatureState({ hmac: 'x', verified: true })).toBe('verified')
    expect(signatureState({ hmac: 'x', verified: false })).toBe('tampered')
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
