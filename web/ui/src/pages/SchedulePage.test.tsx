import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { App } from '../App'
import { FakeSocket, helperReady, json, stubFetch, tunnel, type Routes } from '../test/fakes'

beforeEach(() => {
  Object.defineProperty(navigator, 'languages', { value: ['en-US'], configurable: true })
  FakeSocket.reset()
  vi.stubGlobal('WebSocket', FakeSocket)
  window.history.replaceState(null, '', '/rules/schedule')
})

const office = { name: 'Office', profiles: ['work'], weekdays: ['mon', 'tue', 'wed', 'thu', 'fri'], start: '09:00', end: '18:00' }

function routes(extra: Routes = {}, rules: unknown[] = [office]): Routes {
  return {
    'GET /api/v1/preferences': () => json({ language: 'en', theme: '' }),
    'GET /api/v1/version': () => json({ version: 'v0.7.0' }),
    'GET /api/v1/helper/status': () => json(helperReady),
    'GET /api/v1/tunnels': () => json([tunnel('work', 'connected')]),
    'GET /api/v1/policies/meta': () => json({ profiles: ['home', 'work'], config_path: '/x/kongtrol.yaml' }),
    'GET /api/v1/scheduler/rules': () => json(rules),
    'GET /api/v1/settings': () => json({ scheduler_enabled: true, scheduler_interval: '1m', dashboard_port: 9741 }),
    ...extra,
  }
}

describe('Schedule page', () => {
  it('reads each schedule as a sentence, with no restart note (the daemon reloads it)', async () => {
    stubFetch(routes())
    render(<App />)
    const table = await screen.findByRole('table', { name: 'Connection schedules' })
    expect(
      within(table).getByText('Monday to Friday, from 9:00 to 18:00: connects “work”. Outside that time, Kongtrol disconnects it.'),
    ).toBeInTheDocument()
    expect(screen.queryByText(/restart/i)).not.toBeInTheDocument()
  })

  it('pauses one rule at once (optimistic) and rolls it back if the daemon refuses', async () => {
    let body: unknown = null
    let refuse = false
    let stored = true
    stubFetch(
      routes({
        'GET /api/v1/scheduler/rules': () => json([{ ...office, enabled: stored }]),
        'PUT /api/v1/scheduler/rules/Office': (init) => {
          body = JSON.parse(String(init?.body))
          if (refuse) return json({ error: 'could not save', hint: 'Check that kongtrol.yaml can be written.' }, 500)
          stored = (body as { enabled: boolean }).enabled
          return json({ status: 'updated', rule: 'Office' })
        },
      }),
    )
    render(<App />)
    const sw = await screen.findByRole('switch', { name: 'Apply the schedule “Office”' })
    expect(sw).toHaveAttribute('aria-checked', 'true')
    await userEvent.click(sw)
    expect(sw).toHaveAttribute('aria-checked', 'false')
    expect(await screen.findAllByText('Paused')).not.toHaveLength(0)
    await waitFor(() => expect(body).toEqual({ enabled: false }))

    refuse = true
    await userEvent.click(sw)
    expect(await screen.findByText('Check that kongtrol.yaml can be written.')).toBeInTheDocument()
    await waitFor(() => expect(sw).toHaveAttribute('aria-checked', 'false'))
  })

  it('the switch saves only scheduler_enabled, at once', async () => {
    let body: unknown = null
    let release = () => {}
    const answered = new Promise<void>((r) => (release = r))
    stubFetch(
      routes({
        'GET /api/v1/settings': () => json({ scheduler_enabled: body === null, scheduler_interval: '1m' }),
        'PUT /api/v1/settings': async (init) => {
          body = JSON.parse(String(init?.body))
          await answered
          return json({ scheduler_enabled: false, scheduler_interval: '1m' })
        },
      }),
    )
    render(<App />)
    const sw = await screen.findByRole('switch', { name: 'Use schedules' })
    expect(sw).toHaveAttribute('aria-checked', 'true')
    await userEvent.click(sw)
    // Flips before the daemon answers (optimistic).
    await waitFor(() => expect(sw).toHaveAttribute('aria-checked', 'false'))
    expect(body).toEqual({ scheduler_enabled: false })
    release()
    await waitFor(() => expect(sw).not.toHaveAttribute('aria-busy'))
    expect(sw).toHaveAttribute('aria-checked', 'false')
  })

  it('rejects a bad interval inline and saves a good one on its own', async () => {
    let body: unknown = null
    stubFetch(
      routes({
        'PUT /api/v1/settings': (init) => {
          body = JSON.parse(String(init?.body))
          return json({ scheduler_enabled: true, scheduler_interval: '30s' })
        },
      }),
    )
    render(<App />)
    const field = await screen.findByRole('textbox', { name: 'Check every' })
    await userEvent.clear(field)
    await userEvent.type(field, '5 minutes')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(screen.getByText('“5 minutes” is not a valid interval.')).toBeInTheDocument()
    expect(body).toBeNull()
    await userEvent.clear(field)
    await userEvent.type(field, '30s')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(body).toEqual({ scheduler_interval: '30s' }))
  })

  it('creates a schedule with a live preview, from the empty state', async () => {
    let body: unknown = null
    stubFetch(
      routes(
        {
          'POST /api/v1/scheduler/rules': (init) => {
            body = JSON.parse(String(init?.body))
            return json({ status: 'created' }, 201)
          },
        },
        [],
      ),
    )
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: 'Create an office schedule' }))
    expect(screen.getByRole('textbox', { name: /Name/ })).toHaveValue('Office')
    expect(screen.getByRole('checkbox', { name: 'home' })).toBeChecked()
    expect(screen.getByText(/Monday to Friday, from 9:00 to 18:00: connects “home”/)).toBeInTheDocument()

    await userEvent.click(screen.getByRole('checkbox', { name: 'work' }))
    await userEvent.click(screen.getByRole('checkbox', { name: 'Saturday' }))
    fireEvent.change(screen.getByLabelText('From'), { target: { value: '22:00' } })
    fireEvent.change(screen.getByLabelText('Until'), { target: { value: '06:00' } })
    expect(
      screen.getByText(/Monday to Saturday, from 22:00 to 6:00 the next day: connects “home” and “work”\. Outside that time, Kongtrol disconnects them\./),
    ).toBeInTheDocument()
    expect(screen.getByText(/After midnight the next day already counts/)).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Create schedule' }))
    expect(await screen.findByText('Schedule “Office” created.')).toBeInTheDocument()
    expect(body).toEqual({
      name: 'Office',
      profiles: ['home', 'work'],
      weekdays: ['mon', 'tue', 'wed', 'thu', 'fri', 'sat'],
      start: '22:00',
      end: '06:00',
    })
  })

  it('validates before sending: a VPN and different start and end times', async () => {
    const post = vi.fn(() => json({}, 201))
    stubFetch(routes({ 'POST /api/v1/scheduler/rules': post }, []))
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: 'New schedule' }))
    await userEvent.type(screen.getByRole('textbox', { name: /Name/ }), 'Night')
    fireEvent.change(screen.getByLabelText('Until'), { target: { value: '09:00' } })
    await userEvent.click(screen.getByRole('button', { name: 'Create schedule' }))
    expect(screen.getByText('Choose a VPN profile.')).toBeInTheDocument()
    expect(screen.getByText('It starts and ends at the same time, so it would never turn on.')).toBeInTheDocument()
    expect(post).not.toHaveBeenCalled()
  })

  it('edits and deletes with confirmation', async () => {
    let gone = false
    let put: unknown = null
    stubFetch(
      routes({
        'GET /api/v1/scheduler/rules': () => json(gone ? [] : [office]),
        'PUT /api/v1/scheduler/rules/Office': (init) => {
          put = JSON.parse(String(init?.body))
          return json({ status: 'updated' })
        },
        'DELETE /api/v1/scheduler/rules/Office': () => {
          gone = true
          return json({ status: 'deleted' })
        },
      }),
    )
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: 'Edit “Office”' }))
    await userEvent.click(screen.getByRole('checkbox', { name: 'All day' }))
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(put).toMatchObject({ name: 'Office', start: '', end: '' }))

    await userEvent.click(await screen.findByRole('button', { name: 'Delete “Office”' }))
    await userEvent.click(within(screen.getByRole('alertdialog')).getByRole('button', { name: 'Delete' }))
    expect(await screen.findByText('No schedules yet')).toBeInTheDocument()
  })
})
