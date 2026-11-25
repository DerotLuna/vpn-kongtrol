// One tunnel's charts (ADR-0114): a same-size skeleton until two samples,
// a latency loader only while a first reading is still expected, then a
// static "no reading" with why — never an endless spinner.
import { act, render, screen, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { PreferencesProvider } from '../../app/PreferencesProvider'
import { trafficHistory } from '../../data/traffic'
import { json, stubFetch, tunnel } from '../../test/fakes'
import { LATENCY_WAIT_MS, TunnelTrafficCard } from './TunnelTrafficCard'

const t0 = Date.UTC(2026, 10, 24, 12, 0, 0)
const frame = (ms: number, latency = 0) =>
  tunnel('office', 'connected', { UpdatedAt: new Date(t0 + ms).toISOString(), BytesReceived: ms, BytesSent: ms / 10, LatencyMS: latency })

function renderCard() {
  stubFetch({ 'GET /api/v1/preferences': () => json({ language: '', theme: '' }) })
  render(
    <PreferencesProvider>
      <TunnelTrafficCard tunnel={frame(0)} routes={undefined} allRoutes={undefined} history={undefined} open={false} onToggle={() => {}} />
    </PreferencesProvider>,
  )
  return screen.getByRole('region', { name: 'office' })
}

const record = (...frames: ReturnType<typeof frame>[]) => act(() => frames.forEach((f) => trafficHistory.record([f])))

describe('TunnelTrafficCard', () => {
  beforeEach(() => {
    Object.defineProperty(navigator, 'languages', { value: ['es-AR'], configurable: true })
    trafficHistory.reset()
  })
  afterEach(() => trafficHistory.reset())

  it('shows a skeleton of the three charts, busy and named for screen readers, until two samples exist', () => {
    const card = renderCard()
    record(frame(0))
    const skeleton = card.querySelector('.tr-charts')!
    expect(skeleton).toHaveAttribute('aria-busy', 'true')
    expect(within(card).getByRole('status')).toHaveTextContent('Cargando gráficos')
    expect(skeleton.querySelectorAll('.tr-chart--skeleton')).toHaveLength(3)
    expect(card.querySelectorAll('.k-skeleton__block[aria-hidden="true"]').length).toBeGreaterThan(0)
    expect(within(card).queryByText(/Juntando datos/)).not.toBeInTheDocument()

    // The first frame is only a baseline: two more give two samples.
    record(frame(5000))
    expect(skeleton).toHaveAttribute('aria-busy', 'true')
    record(frame(10_000))
    expect(card.querySelector('[aria-busy="true"]')).toBeNull()
    expect(within(card).getByRole('slider', { name: 'Recibido' })).toBeInTheDocument()
  })

  it('latency: a loader while a first reading is expected, then a static "no reading" with why', () => {
    const card = renderCard()
    record(frame(0), frame(5000), frame(10_000))
    expect(within(card).getByRole('status')).toHaveTextContent('Midiendo la latencia…')
    expect(card.querySelector('.k-spinner')).not.toBeNull()

    // Timed from the first sample (the second frame).
    record(frame(5000 + LATENCY_WAIT_MS - 1))
    expect(card.querySelector('.k-spinner')).not.toBeNull()
    record(frame(5000 + LATENCY_WAIT_MS))
    expect(card.querySelector('.k-spinner')).toBeNull()
    expect(within(card).getByText('Sin medición para esta VPN')).toBeInTheDocument()
    const why = within(card).getByRole('button', { name: 'Por qué no hay latencia' })
    expect(why).toHaveAccessibleDescription(/abriendo una conexión con el servidor de la VPN o con el DNS/)
  })

  it('latency: one reading shows its value without a spinner; two draw the chart', () => {
    const card = renderCard()
    record(frame(0, 30), frame(5000, 30), frame(10_000, 30))
    expect(card.querySelector('.k-spinner')).toBeNull()
    expect(within(card).getByText('30 ms')).toBeInTheDocument()
    expect(within(card).getByText(/Última medición/)).toBeInTheDocument()

    record(frame(15_000, 42))
    expect(within(card).getByRole('slider', { name: 'Latencia' })).toBeInTheDocument()
  })
})
