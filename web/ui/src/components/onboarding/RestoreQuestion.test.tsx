import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { MemoryRouter } from 'react-router'
import { DataProvider } from '../../app/DataProvider'
import { PreferencesProvider } from '../../app/PreferencesProvider'
import { RestoreQuestion } from './RestoreQuestion'

function json(body: unknown, status = 200) {
  return Promise.resolve(new Response(JSON.stringify(body), { status }))
}

class SilentSocket {
  onopen: (() => void) | null = null
  onmessage: ((e: MessageEvent) => void) | null = null
  onclose: (() => void) | null = null
  close() {}
}

type State = { start_at_login: boolean; start_at_login_supported: boolean; restore_tunnels: boolean; ask: boolean }

function stubStartup(initial: State) {
  let state = initial
  const puts: unknown[] = []
  const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
    const route = `${init?.method ?? 'GET'} ${String(input)}`
    if (route === 'GET /api/v1/preferences') return json({ language: '', theme: '' })
    if (route === 'GET /api/v1/tunnels') return json([])
    if (route === 'GET /api/v1/startup') return json(state)
    if (route === 'PUT /api/v1/startup') {
      const body = JSON.parse(String(init?.body)) as { restore_answer: boolean }
      puts.push(body)
      const on = body.restore_answer
      state = { ...state, ask: false, restore_tunnels: on, start_at_login: on || state.start_at_login }
      return json(state)
    }
    return json({ error: 'not found' }, 404)
  })
  vi.stubGlobal('fetch', fetchMock)
  vi.stubGlobal('WebSocket', SilentSocket)
  return puts
}

function Harness({ variant = 'panel' }: { variant?: 'panel' | 'tray' }) {
  return (
    <PreferencesProvider>
      <DataProvider>
        <MemoryRouter>
          <RestoreQuestion variant={variant} />
        </MemoryRouter>
      </DataProvider>
    </PreferencesProvider>
  )
}

const pending: State = { start_at_login: false, start_at_login_supported: true, restore_tunnels: false, ask: true }

describe('RestoreQuestion (ADR-0101)', () => {
  beforeEach(() => {
    Object.defineProperty(navigator, 'languages', { value: ['es-AR'], configurable: true })
  })
  afterEach(() => vi.unstubAllGlobals())

  it('shows nothing while the daemon has no question', async () => {
    stubStartup({ ...pending, ask: false })
    const { container } = render(<Harness />)
    await new Promise((r) => setTimeout(r, 50))
    expect(container.textContent).toBe('')
  })

  it.each([
    ['panel', 'Sí, activar', true],
    ['panel', 'Ahora no', false],
    ['tray', 'Sí, activar', true],
    ['tray', 'Ahora no', false],
  ] as const)('%s: "%s" answers with restore_answer=%s and hides the question', async (variant, button, on) => {
    const puts = stubStartup(pending)
    render(<Harness variant={variant} />)
    expect(await screen.findByText('¿Volver a conectar tus VPN al encender el equipo?')).toBeInTheDocument()
    expect(screen.getByText(/Puedes cambiarlo en Ajustes › Al iniciar\./)).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: button }))
    await vi.waitFor(() => expect(puts).toEqual([{ restore_answer: on }]))
    await vi.waitFor(() => expect(screen.queryByText('¿Volver a conectar tus VPN al encender el equipo?')).toBeNull())
  })
})
