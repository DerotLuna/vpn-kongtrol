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

  it('a program chip says it is not applied yet, with a tooltip saying why (ADR-0113)', async () => {
    stubFetch(routes())
    render(<App />)
    const table = await screen.findByRole('table', { name: 'Rules by destination, in the order they are checked' })
    const [workRow, labRow] = within(table).getAllByRole('row').slice(1)
    const tag = within(labRow!).getByRole('button', { name: 'not applied yet' })
    expect(tag).toHaveAccessibleDescription(/this VPN does not carry programs/)
    expect(within(workRow!).queryByRole('button', { name: 'not applied yet' })).not.toBeInTheDocument()
  })

  it('the Done step of a program rule never says it goes through the VPN, and offers no test (ADR-0113)', async () => {
    const explain = vi.fn(() => json({ Target: 'claude.exe', Kind: 'app', Matched: true, Via: 'home', RuleName: 'x' }))
    stubFetch(
      routes(
        {
          'GET /api/v1/vpns': () => json([{ name: 'home', type: 'nordvpn' }]),
          'POST /api/v1/policies': () => json({ status: 'created' }, 201),
          'PUT /api/v1/policies/claude.exe%20via%20home': () => json({ status: 'updated' }),
          'GET /api/v1/policies/explain?target=app%3Aclaude.exe': explain,
        },
        [],
      ),
    )
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: 'Create the first rule' }))
    const sheet = await screen.findByRole('dialog', { name: 'Send a site through a VPN' })
    fireEvent.change(await within(sheet).findByRole('textbox', { name: /Site, IP or program/ }), { target: { value: 'app:claude.exe' } })
    const notApplied = 'Programs: they only go through the VPN on Linux, with a WireGuard profile that has Executable egress on. Otherwise Kongtrol only recognises them: use sites or IPs.'
    expect(within(sheet).getByText(notApplied)).toBeInTheDocument()
    await userEvent.click(within(sheet).getByRole('button', { name: 'Save paused' }))
    expect(await within(sheet).findByText('The rule “claude.exe via home” is saved.')).toBeInTheDocument()
    expect(within(sheet).getByText(notApplied)).toBeInTheDocument()

    await userEvent.click(within(sheet).getByRole('button', { name: 'Activate rule' }))
    await waitFor(() => expect(within(sheet).queryByRole('button', { name: 'Activate rule' })).not.toBeInTheDocument())
    expect(within(sheet).queryByRole('button', { name: 'Test' })).not.toBeInTheDocument()
    expect(within(sheet).queryByRole('button', { name: 'Connect home now' })).not.toBeInTheDocument()
    expect(within(sheet).queryByText(/Tested:/)).not.toBeInTheDocument()
    expect(explain).not.toHaveBeenCalled()
  })

  it('validates the entire list, saves it paused, and connects only after explicit activation and connection', async () => {
    let body: unknown = null
    const activationPatches: unknown[] = []
    const connect = vi.fn(() => json({ status: 'connecting' }, 202))
    stubFetch(
      routes(
        {
          'GET /api/v1/vpns': () => json([{ name: 'home', type: 'nordvpn' }]),
          'POST /api/v1/policies': (init) => {
            body = JSON.parse(String(init?.body))
            return json({ status: 'created' }, 201)
          },
          'PUT /api/v1/policies/git.acme.test%20via%20home': (init) => {
            activationPatches.push(JSON.parse(String(init?.body)))
            return activationPatches.length === 1
              ? json({ error: 'Activation refused.', hint: 'Try again.' }, 400)
              : json({ status: 'updated' })
          },
          'POST /api/v1/tunnels/home/connect': connect,
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
    await userEvent.click(await within(sheet).findByRole('button', { name: 'Save paused' }))
    expect(within(sheet).getByText('Type a site or an IP.')).toBeInTheDocument()

    const destinations = within(sheet).getByRole('textbox', { name: /Site, IP or program/ })
    fireEvent.change(destinations, { target: { value: 'https://git.acme.test/x,y, git.acme.test\n192.0.2.4, bad host' } })
    await userEvent.click(within(sheet).getByRole('button', { name: 'Save paused' }))
    expect(body).toBeNull()
    expect(destinations).toHaveAttribute('aria-invalid', 'true')
    fireEvent.change(destinations, { target: { value: 'https://git.acme.test/x,y, git.acme.test\n192.0.2.4\napp:Code, Inc.exe' } })
    expect(within(sheet).getByRole('button', { name: /Goes through/ })).toHaveTextContent(/home/)
    await userEvent.click(within(sheet).getByRole('radio', { name: 'Block it' }))
    await userEvent.click(within(sheet).getByRole('button', { name: 'Save paused' }))

    expect(await within(sheet).findByText('The rule “git.acme.test via home” is saved.')).toBeInTheDocument()
    expect(body).toEqual({
      name: 'git.acme.test via home',
      via: 'home',
      disabled: true,
      on_tunnel_down: 'block',
      match: { domains: ['git.acme.test', '*.git.acme.test'], ip_ranges: ['192.0.2.4'], apps: ['Code, Inc.exe'] },
    })
    expect(within(sheet).queryByRole('button', { name: 'Connect home now' })).not.toBeInTheDocument()
    expect(activationPatches).toEqual([])
    expect(connect).not.toHaveBeenCalled()

    await userEvent.click(within(sheet).getByRole('button', { name: 'Activate rule' }))
    expect(await within(sheet).findByText('Activation refused.')).toBeInTheDocument()
    expect(within(sheet).queryByRole('button', { name: 'Connect home now' })).not.toBeInTheDocument()
    await userEvent.click(within(sheet).getByRole('button', { name: 'Activate rule' }))
    const connectNow = await within(sheet).findByRole('button', { name: 'Connect home now' })
    expect(activationPatches).toEqual([{ disabled: false }, { disabled: false }])
    expect(connect).not.toHaveBeenCalled()
    await userEvent.click(connectNow)
    await waitFor(() => expect(connect).toHaveBeenCalledOnce())
  })

  it('pauses and resumes from the list and leaves the saved state alone when editing', async () => {
    const patches: Record<string, unknown>[] = []
    const saved = { ...lab, disabled: true }
    stubFetch(routes({
      'PUT /api/v1/policies/Lab': (init) => {
        const patch = JSON.parse(String(init?.body)) as Record<string, unknown>
        patches.push(patch)
        if (typeof patch.disabled === 'boolean') saved.disabled = patch.disabled
        return json({ status: 'updated' })
      },
    }, [work, saved]))
    render(<App />)
    const legacy = await screen.findByRole('switch', { name: 'Rule “Work”' })
    expect(legacy).toBeChecked()
    const toggle = screen.getByRole('switch', { name: 'Rule “Lab”' })
    expect(toggle).not.toBeChecked()
    expect(toggle).toHaveTextContent('Paused')
    await userEvent.click(toggle)
    await waitFor(() => expect(legacy).toBeEnabled())
    expect(toggle).toBeChecked()
    expect(patches).toEqual([{ disabled: false }])
    await userEvent.click(toggle)
    await waitFor(() => expect(legacy).toBeEnabled())
    expect(patches).toEqual([{ disabled: false }, { disabled: true }])
    expect(toggle).not.toBeChecked()

    await userEvent.click(screen.getByRole('button', { name: 'Edit “Lab”' }))
    expect(screen.getByRole('textbox', { name: 'Site, IP or program' })).toHaveValue('10.2.0.0/16\napp:code.exe')
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(patches).toHaveLength(3))
    expect(patches[2]).not.toHaveProperty('disabled')
    expect(toggle).not.toBeChecked()
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
    expect(screen.getByRole('textbox', { name: 'Site, IP or program' })).toHaveValue('10.2.0.0/16\napp:code.exe')
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

describe('Rules page: order between overlapping rules (ADR-0108)', () => {
  const git = { name: 'Git', via: 'home', domains: ['git.corp.example'], ip_ranges: null, apps: null, on_tunnel_down: 'allow' }

  it('marks the rows the order settles, explains who decided and links to the priority field', async () => {
    stubFetch(
      routes(
        {
          'GET /api/v1/vpns': () =>
            json([
              { name: 'office', type: 'openvpn', priority: 10 },
              { name: 'home', type: 'nordvpn', priority: 10 },
            ]),
        },
        [work, git, lab],
      ),
    )
    render(<App />)
    const table = await screen.findByRole('table', { name: 'Rules by destination, in the order they are checked' })
    const rows = within(table).getAllByRole('row').slice(1)
    const hint = await within(rows[0]!).findByRole('link', { name: 'Priority 10' })
    expect(hint).toHaveAttribute('href', '/profiles?edit=office&focus=priority')
    expect(hint).toHaveAccessibleDescription(/only decides between your rules when two cover the same site or the same IP range/)
    expect(within(rows[1]!).getByRole('link', { name: 'Priority 10' })).toHaveAttribute('href', '/profiles?edit=home&focus=priority')
    // Lab only overlaps by a nested range: the most specific one wins, the order changes nothing.
    expect(within(rows[2]!).queryByRole('link', { name: /Priority|by position/ })).not.toBeInTheDocument()
    expect(screen.getByText('Both VPNs have the same priority: the position in the list decides.')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Change the priority of home' })).toHaveAttribute('href', '/profiles?edit=home&focus=priority')
  })

  it('a decided priority needs no link; with no priorities the rows say "by position"', async () => {
    stubFetch(
      routes(
        {
          'GET /api/v1/vpns': () =>
            json([
              { name: 'office', type: 'openvpn', priority: 30 },
              { name: 'home', type: 'nordvpn', priority: 10 },
            ]),
        },
        [work, git],
      ),
    )
    const { unmount } = render(<App />)
    expect(await screen.findByText('Priority decides: office (30) goes before home (10).')).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /Change the priority of/ })).not.toBeInTheDocument()
    unmount()

    stubFetch(routes({ 'GET /api/v1/vpns': () => json([]) }, [work, git]))
    render(<App />)
    const table = await screen.findByRole('table', { name: 'Rules by destination, in the order they are checked' })
    expect(within(table).getAllByRole('link', { name: 'by position' })).toHaveLength(2)
    expect(screen.getByText('Neither VPN has a priority: the position in the list decides.')).toBeInTheDocument()
  })
})
