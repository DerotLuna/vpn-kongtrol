import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { App } from '../App'
import { FakeSocket, helperReady, json, noRetryClient, stubFetch, tunnel, type Routes } from '../test/fakes'

beforeEach(() => {
  Object.defineProperty(navigator, 'languages', { value: ['en-US'], configurable: true })
  FakeSocket.reset()
  vi.stubGlobal('WebSocket', FakeSocket)
  window.history.replaceState(null, '', '/rules')
})

const work = {
  name: 'Work',
  via: 'office',
  domains: ['*.corp.example', 'corp.example'],
  ip_ranges: ['10.0.0.0/8'],
  apps: null,
  on_tunnel_down: 'block',
}
const lab = { name: 'Lab', via: 'home', domains: null, ip_ranges: ['10.2.0.0/16'], apps: ['code.exe'], on_tunnel_down: 'allow' }

function routes(extra: Routes = {}, policies: unknown[] = [work, lab]): Routes {
  return {
    'GET /api/v1/preferences': () => json({ language: 'en', theme: '' }),
    'GET /api/v1/version': () => json({ version: 'v0.7.0' }),
    'GET /api/v1/helper/status': () => json(helperReady),
    'GET /api/v1/tunnels': () => json([tunnel('office', 'connected'), tunnel('home', 'disconnected')]),
    'GET /api/v1/policies': () => json(policies),
    'GET /api/v1/policies/meta': () => json({ profiles: ['home', 'office'], config_path: '/home/u/.kongtrol/kongtrol.yaml' }),
    ...extra,
  }
}

describe('Rules page', () => {
  it('shows one recovery notice when authorization is refused', async () => {
    stubFetch(routes({
      'GET /api/v1/policies': () => Promise.resolve(new Response('forbidden origin', { status: 403 })),
    }))
    render(<App />)
    expect(await screen.findByText('This page is not allowed to talk to Kongtrol.')).toBeInTheDocument()
    expect(screen.getAllByRole('alert')).toHaveLength(1)
    expect(screen.getByText(/open the dashboard from Kongtrol/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Retry' })).not.toBeInTheDocument()
  })

  it('lists rules in evaluation order with their VPN state, what happens when off, and overlaps', async () => {
    stubFetch(routes())
    render(<App />)
    const table = await screen.findByRole('table', { name: 'Rules by destination, in the order they are checked' })
    const rows = within(table).getAllByRole('row').slice(1)
    expect(rows.map((r) => within(r).getByRole('rowheader').textContent)).toEqual(['Work', 'Lab'])
    expect(within(rows[0]!).getByText('Connected')).toBeInTheDocument()
    expect(within(rows[0]!).getByText('Blocked')).toBeInTheDocument()
    expect(within(rows[1]!).getByText('Disconnected')).toBeInTheDocument()
    expect(within(rows[1]!).getByText('code.exe')).toBeInTheDocument()
    expect(screen.getByText(/10\.2\.0\.0\/16 \(“Lab”\) is inside 10\.0\.0\.0\/8 \(“Work”\)/)).toBeInTheDocument()
    expect(screen.getByText('/home/u/.kongtrol/kongtrol.yaml')).toBeInTheDocument()
  })

  it('teaches with an empty state; "Create the first rule" opens the add sheet and saves one sentence as a rule', async () => {
    let body: unknown = null
    stubFetch(
      routes(
        {
          'GET /api/v1/vpns': () => json([{ name: 'home', type: 'nordvpn' }, { name: 'office', type: 'forticlient' }]),
          'POST /api/v1/policies': (init) => {
            body = JSON.parse(String(init?.body))
            return json({ status: 'created' }, 201)
          },
        },
        [],
      ),
    )
    render(<App />)
    expect(await screen.findByText('No rules yet')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Create the first rule' }))

    const sheet = await screen.findByRole('dialog', { name: 'Send a site through a VPN' })
    expect(new URLSearchParams(window.location.search).get('add')).toBe('rule')
    // Nothing typed: the error names the problem and says what to do.
    await userEvent.click(await within(sheet).findByRole('button', { name: 'Save rule' }))
    expect(within(sheet).getByText('Type a site, an IP or a program.')).toBeInTheDocument()

    await userEvent.type(within(sheet).getByRole('textbox', { name: /Site, IP or program/ }), 'https://git.acme.test/x')
    // "Goes through" starts on the connected profile.
    expect(within(sheet).getByRole('button', { name: /Goes through/ })).toHaveTextContent(/office/)
    await userEvent.click(within(sheet).getByRole('radio', { name: 'Block it' }))
    await userEvent.click(within(sheet).getByRole('button', { name: 'Save rule' }))

    expect(await within(sheet).findByText('The rule “git.acme.test via office” is saved.')).toBeInTheDocument()
    expect(body).toEqual({
      name: 'git.acme.test via office',
      via: 'office',
      on_tunnel_down: 'block',
      match: { domains: ['git.acme.test', '*.git.acme.test'], ip_ranges: [], apps: [] },
    })
  })

  it('keeps the form and shows the daemon’s refusal next to it', async () => {
    stubFetch(
      routes({
        'PUT /api/v1/policies/Lab': () =>
          json({ error: 'Policy “Lab”: block needs destinations.', hint: 'Add a site or a network.', error_key: 'config.err.x' }, 400),
      }),
    )
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: 'Edit “Lab”' }))
    expect(screen.getByRole('textbox', { name: 'Apps' })).toHaveValue('code.exe')
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))
    // inline and as a toast: the toast is never the only copy
    expect(await screen.findAllByText('Policy “Lab”: block needs destinations.')).toHaveLength(2)
    expect(screen.getByRole('button', { name: 'Save changes' })).toBeEnabled()
  })

  it('tests a draft before saving it', async () => {
    let sent: unknown = null
    stubFetch(
      routes({
        'POST /api/v1/policies/test': (init) => {
          sent = JSON.parse(String(init?.body))
          return json({ matched: true, via: 'home', rule: 'Lab' })
        },
      }),
    )
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: 'Edit “Lab”' }))
    await userEvent.type(screen.getByRole('textbox', { name: /Test this rule before saving it/ }), 'app:code.exe')
    await userEvent.click(screen.getAllByRole('button', { name: 'Test' })[0]!)
    expect(await screen.findByText('Match: this traffic would go through “home”.')).toBeInTheDocument()
    expect(sent).toMatchObject({ target: '', app: 'code.exe', rule: { name: 'Lab', via: 'home' } })
  })

  it('“Test a destination” shows the route, why, and what happens with the VPN off', async () => {
    stubFetch(
      routes({
        'GET /api/v1/policies/explain?target=10.2.3.4': () =>
          json({
            Target: '10.2.3.4',
            Kind: 'ip',
            Matched: true,
            Via: 'home',
            RuleName: 'Lab',
            Reason: 'matched CIDR 10.2.0.0/16 (longest-prefix wins)',
            ReasonCode: 'matched_cidr',
            ReasonArgs: ['10.2.0.0/16'],
            DefaultTo: 'default route',
          }),
      }),
    )
    render(<App />)
    const input = await screen.findByRole('textbox', { name: /Site, IP or app/ })
    await userEvent.type(input, '10.2.3.4{enter}')
    const route = await screen.findByRole('list', { name: 'Route of 10.2.3.4' })
    expect(within(route).getByText('“Lab”')).toBeInTheDocument()
    expect(within(route).getByText('matches network 10.2.0.0/16 (the most specific wins)')).toBeInTheDocument()
    expect(within(route).getByText('Normal connection')).toBeInTheDocument()
    expect(screen.getByText(/“home” is off: it uses your normal connection now/)).toBeInTheDocument()
    expect(screen.getByText('kongtrol policy show 10.2.3.4')).toBeInTheDocument()
  })

  it('asks before deleting and removes the row at once', async () => {
    let deleted = false
    stubFetch(
      routes({
        'GET /api/v1/policies': () => json(deleted ? [lab] : [work, lab]),
        'DELETE /api/v1/policies/Work': () => {
          deleted = true
          return json({ status: 'deleted' })
        },
      }),
    )
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: 'Delete “Work”' }))
    const dialog = screen.getByRole('alertdialog', { name: 'Delete rule “Work”?' })
    await userEvent.click(within(dialog).getByRole('button', { name: 'Delete' }))
    await waitFor(() => expect(screen.queryByRole('rowheader', { name: 'Work' })).toBeNull())
    expect(deleted).toBe(true)
    expect(await screen.findByText('Rule “Work” deleted.')).toBeInTheDocument()
  })

  it('reloads the file and says how many rules it read', async () => {
    stubFetch(routes({ 'POST /api/v1/policies/reload': () => json({ status: 'reloaded', policies: 2 }) }))
    render(<App />)
    fireEvent.click(await screen.findByRole('button', { name: 'Read the file again' }))
    expect(await screen.findByText('Done: 2 rules read from the file.')).toBeInTheDocument()
  })

  it('a failed load says what to do and offers Retry', async () => {
    stubFetch(routes({ 'GET /api/v1/policies': () => json({ error: 'boom' }, 500) }))
    render(<App client={noRetryClient()} />)
    expect(await screen.findByText('boom')).toBeInTheDocument()
    expect(screen.getAllByRole('button', { name: 'Retry' }).length).toBeGreaterThan(0)
  })
})
