import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { App } from '../App'
import { RESTART_TIMING, resetRestartPhase, type VPNProfile } from '../data/profiles'
import { FakeSocket, helperReady, json, stubFetch, tunnel, type Routes } from '../test/fakes'

function profile(name: string, extra: Partial<VPNProfile> = {}): VPNProfile {
  return {
    name,
    type: 'openvpn',
    config: `~/.kongtrol/configs/${name}.ovpn`,
    priority: 10,
    failover: [],
    auth_method: 'certificate',
    has_username_credential: false,
    has_password_credential: false,
    kill_switch_override: 'inherit',
    auto_connect: true,
    ...extra,
  }
}

const setupDone = { setup_mode: false, needs_setup: false, restart_pending: false, config_path: '/home/ana/.kongtrol/kongtrol.yaml' }

function base(extra: Routes = {}): Routes {
  return {
    'GET /api/v1/preferences': () => json({ language: 'en', theme: '' }),
    'GET /api/v1/version': () => json({ version: 'v0.7.0' }),
    'GET /api/v1/helper/status': () => json(helperReady),
    'GET /api/v1/tunnels': () => json([tunnel('work', 'connected')]),
    'GET /api/v1/setup': () => json(setupDone),
    'GET /api/v1/groups': () => json([]),
    'GET /api/v1/vpns/detect': () => json([]),
    ...extra,
  }
}

function bodyOf(fetchMock: ReturnType<typeof stubFetch>, key: string): unknown {
  const call = fetchMock.mock.calls.find(([url, init]) => `${init?.method ?? 'GET'} ${String(url)}` === key)
  return call ? JSON.parse(String(call[1]?.body)) : undefined
}

const timing = { ...RESTART_TIMING }

beforeEach(() => {
  Object.defineProperty(navigator, 'languages', { value: ['en-US'], configurable: true })
  FakeSocket.reset()
  vi.stubGlobal('WebSocket', FakeSocket)
  window.history.replaceState(null, '', '/profiles')
  Object.assign(RESTART_TIMING, { stepMs: 1, stopSteps: 5, startSteps: 5 })
})

afterEach(() => {
  Object.assign(RESTART_TIMING, timing)
  resetRestartPhase()
})

describe('ProfilesPage', () => {
  it('lists profiles with live state, sign-in and a read-only kill switch link', async () => {
    stubFetch(
      base({
        'GET /api/v1/vpns': () =>
          json([
            profile('work', { has_username_credential: true, has_password_credential: true, kill_switch_override: 'on', failover: ['home'] }),
            profile('home', { type: 'nordvpn', config: '', server: 'us3710' }),
          ]),
      }),
    )
    render(<App />)
    const table = await screen.findByRole('table', { name: 'Configured VPN profiles' })
    const rows = within(table).getAllByRole('row')
    const work = rows.find((r) => within(r).queryByText('work'))!
    expect(within(work).getByText('Connected')).toBeInTheDocument()
    expect(within(work).getByText('Username and password stored')).toBeInTheDocument()
    expect(within(work).getByText('home')).toBeInTheDocument()
    expect(within(work).getByRole('link', { name: "Change work's kill switch in Security" })).toHaveAttribute('href', '/security#profile-overrides')
    const home = rows.find((r) => within(r).queryByText('home') && within(r).queryByText(/NordVPN/))!
    expect(within(home).getByText("In the provider's app")).toBeInTheDocument()
    expect(screen.getByText('/home/ana/.kongtrol/kongtrol.yaml')).toBeInTheDocument()
  })

  it('first run: suggests the installed VPN app, saves the first profile, restarts and goes home', async () => {
    let restarted = false
    let setupCalls = 0
    const fetchMock = stubFetch(
      base({
        'GET /api/v1/vpns': () => json([]),
        'GET /api/v1/setup': () => {
          setupCalls++
          // While restarting: answers until POST /restart, then down once, then up again.
          if (restarted && setupCalls === 1000) return Promise.reject(new TypeError('down'))
          return json({ setup_mode: true, needs_setup: !restarted, restart_pending: false, config_path: '' })
        },
        'GET /api/v1/vpns/detect': () => json([{ label: 'NordVPN', adapter_type: 'nordvpn', path: 'C:/NordVPN/nordvpn.exe' }]),
        'POST /api/v1/vpns': () => json({ status: 'created', profile: 'nord', restart_required: true }, 201),
        'POST /api/v1/restart': () => {
          restarted = true
          setupCalls = 999
          return json({ status: 'restarting' }, 202)
        },
      }),
    )
    render(<App />)
    expect(await screen.findByRole('heading', { name: 'Welcome to Kongtrol: create your first VPN' })).toBeInTheDocument()
    await userEvent.click(await screen.findByRole('button', { name: /Use NordVPN/ }))

    const dialog = await screen.findByRole('dialog', { name: 'Your first VPN' })
    // NordVPN asks for no password: its app holds the session.
    expect(within(dialog).queryByLabelText('Password')).not.toBeInTheDocument()
    await userEvent.type(within(dialog).getByLabelText(/^Name/), 'nord')
    await userEvent.type(within(dialog).getByLabelText('Country (optional)'), 'CH')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Create and turn on' }))

    await waitFor(() => expect(bodyOf(fetchMock, 'POST /api/v1/vpns')).toBeDefined())
    expect(bodyOf(fetchMock, 'POST /api/v1/vpns')).toEqual({ name: 'nord', type: 'nordvpn', priority: 10, auto_connect: true, country: 'CH' })
    await waitFor(() => expect(fetchMock.mock.calls.some(([u, i]) => String(u) === '/api/v1/restart' && i?.method === 'POST')).toBe(true))
    expect(await screen.findByText('Kongtrol restarted · you can connect your VPNs now')).toBeInTheDocument()
    await waitFor(() => expect(window.location.pathname).toBe('/'))
  })

  it('checks the form inline before sending and focuses the first problem', async () => {
    const fetchMock = stubFetch(base({ 'GET /api/v1/vpns': () => json([profile('work')]) }))
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: 'New VPN' }))
    const dialog = await screen.findByRole('dialog', { name: 'New VPN' })
    expect(within(dialog).getByText('Step 1 of 2', { exact: false })).toBeInTheDocument()
    await userEvent.click(within(dialog).getByRole('button', { name: /OpenVPN/ }))
    await userEvent.type(within(dialog).getByLabelText(/^Name/), 'work')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Create VPN' }))

    expect(await within(dialog).findByText('You already have a VPN called “work”.')).toBeInTheDocument()
    expect(within(dialog).getByText('The .ovpn file is missing.')).toBeInTheDocument()
    await waitFor(() => expect(within(dialog).getByLabelText(/^Name/)).toHaveFocus())
    expect(fetchMock.mock.calls.some(([, i]) => i?.method === 'POST')).toBe(false)

    // Fixing a field clears its error on the next check.
    await userEvent.clear(within(dialog).getByLabelText(/^Name/))
    await userEvent.type(within(dialog).getByLabelText(/^Name/), 'office')
    expect(within(dialog).queryByText('You already have a VPN called “work”.')).not.toBeInTheDocument()
  })

  it('puts a daemon validation error on its field, with what to do', async () => {
    stubFetch(
      base({
        'GET /api/v1/vpns': () => json([profile('work'), profile('home', { failover: ['work'] })]),
        'PUT /api/v1/vpns/work': () =>
          json(
            { error: 'The backups form a loop: work → home → work.', hint: 'Remove one of those failover entries.', error_key: 'config.err.failover_cycle' },
            400,
          ),
      }),
    )
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: 'Edit work' }))
    const dialog = await screen.findByRole('dialog', { name: 'Edit work' })
    expect(within(dialog).getByLabelText(/^Name/)).not.toHaveAttribute('readonly')
    // Pass the local check (daemon has the final say): no backup added, just save.
    await userEvent.click(within(dialog).getByRole('button', { name: 'Save changes' }))
    expect(await within(dialog).findByText('The backups form a loop: work → home → work.')).toBeInTheDocument()
    expect(within(dialog).getByText('Remove one of those failover entries.')).toBeInTheDocument()
  })

  it('adds a backup in order and refuses a loop before saving', async () => {
    const fetchMock = stubFetch(
      base({
        'GET /api/v1/vpns': () => json([profile('work'), profile('home', { failover: ['work'] }), profile('spare')]),
        'PUT /api/v1/vpns/work': () => json({ status: 'updated', profile: 'work', restart_required: true }),
      }),
    )
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: 'Edit work' }))
    const dialog = await screen.findByRole('dialog', { name: 'Edit work' })
    const add = within(dialog).getByRole('combobox', { name: 'Backup VPN to add' })
    await userEvent.selectOptions(add, 'home')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Add backup' }))
    await userEvent.click(within(dialog).getByRole('button', { name: 'Save changes' }))
    expect(await within(dialog).findByText('The backups form a loop: work → home → work.')).toBeInTheDocument()

    await userEvent.click(within(dialog).getByRole('button', { name: 'Remove home' }))
    await userEvent.selectOptions(within(dialog).getByRole('combobox', { name: 'Backup VPN to add' }), 'spare')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Add backup' }))
    await userEvent.click(within(dialog).getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(bodyOf(fetchMock, 'PUT /api/v1/vpns/work')).toMatchObject({ failover: ['spare'] }))
    // Saved: the restart banner appears with its action.
    expect(await screen.findByText('Restart Kongtrol to use the changes')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Restart now' })).toBeEnabled()
  })

  it('restart: a daemon that cannot restart says so with what to do; one that never comes back times out', async () => {
    let mode: 'refuse' | 'gone' = 'refuse'
    stubFetch(
      base({
        'GET /api/v1/vpns': () => json([profile('work')]),
        'GET /api/v1/setup': () =>
          mode === 'gone' ? Promise.reject(new TypeError('down')) : json({ ...setupDone, setup_mode: true, restart_pending: true }),
        'POST /api/v1/restart': () => {
          if (mode === 'refuse') return json({ error: 'restart not supported by this daemon instance', code: 'not_supported' }, 501)
          return json({ status: 'restarting' }, 202)
        },
      }),
    )
    render(<App />)
    // The daemon's own restart_pending shows the banner too.
    await userEvent.click(await screen.findByRole('button', { name: 'Restart now' }))
    expect(await screen.findByText('Could not restart Kongtrol')).toBeInTheDocument()
    expect(screen.getByText(/This Kongtrol can't restart itself\. · Close it with/)).toBeInTheDocument()

    mode = 'gone'
    await userEvent.click(screen.getByRole('button', { name: 'Dismiss notice' }))
    await userEvent.click(screen.getByRole('button', { name: 'Restart now' }))
    expect(await screen.findByText('Kongtrol did not answer again')).toBeInTheDocument()
  })

  it('deletes optimistically; a refusal puts the row back and lists every place that uses it', async () => {
    stubFetch(
      base({
        'GET /api/v1/vpns': () => json([profile('work'), profile('home')]),
        'DELETE /api/v1/vpns/home': () =>
          json(
            {
              error: 'profile "home" is in use · policies: intranet, git · groups: — · failover backup of: — · schedules: office',
              hint: 'Remove it from those places (or switch them to another profile) and delete it again.',
              error_key: 'api.err.vpn_in_use',
            },
            409,
          ),
      }),
    )
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: 'Delete home' }))
    const confirm = await screen.findByRole('alertdialog', { name: 'Delete the VPN “home”?' })
    await userEvent.click(within(confirm).getByRole('button', { name: 'Delete' }))
    const refused = await screen.findByRole('alertdialog', { name: 'Delete the VPN “home”?' })
    expect(await within(refused).findByText('profile "home" is in use')).toBeInTheDocument()
    const where = within(refused).getByLabelText('Where it is used')
    expect(within(where).getByText('policies')).toBeInTheDocument()
    expect(within(where).getByText('intranet')).toBeInTheDocument()
    expect(within(where).getByText('git')).toBeInTheDocument()
    expect(within(where).getByText('office')).toBeInTheDocument()
    // Empty lists ("—") are left out.
    expect(within(where).queryByText('groups')).not.toBeInTheDocument()
    expect(within(refused).getByText(/Remove it from those places/)).toBeInTheDocument()
    expect(within(refused).queryByRole('button', { name: 'Delete' })).not.toBeInTheDocument()
    await userEvent.click(within(refused).getByRole('button', { name: 'Got it' }))
    expect(await screen.findByRole('button', { name: 'Delete home' })).toBeInTheDocument()
  })

  it('renames a profile, says what followed it and offers the restart', async () => {
    const fetchMock = stubFetch(
      base({
        'GET /api/v1/vpns': () => json([profile('work'), profile('home')]),
        'PUT /api/v1/vpns/home': () => json({ status: 'renamed', profile: 'casa', previous: 'home', restart_required: true }),
      }),
    )
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: 'Edit home' }))
    const dialog = await screen.findByRole('dialog', { name: 'Edit home' })
    const name = within(dialog).getByLabelText(/^Name/)
    await userEvent.clear(name)
    await userEvent.type(name, 'casa')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Save changes' }))
    expect(await screen.findByText('VPN “home” is now called “casa”.')).toBeInTheDocument()
    expect(bodyOf(fetchMock, 'PUT /api/v1/vpns/home')).toMatchObject({ name: 'casa' })
    expect(await screen.findByRole('button', { name: 'Restart now' })).toBeInTheDocument()
  })

  it('rename: a connected VPN says to disconnect it first, and the daemon refusal lands on the name', async () => {
    stubFetch(
      base({
        'GET /api/v1/vpns': () => json([profile('work'), profile('home')]),
        'PUT /api/v1/vpns/work': () =>
          json(
            {
              error: '"work" is connected or connecting: it cannot be renamed now',
              hint: 'Disconnect it and save the new name again.',
              error_key: 'api.err.vpn_rename_busy',
            },
            409,
          ),
      }),
    )
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: 'Edit work' }))
    const dialog = await screen.findByRole('dialog', { name: 'Edit work' })
    const name = within(dialog).getByLabelText(/^Name/)
    await userEvent.type(name, '2')
    expect(within(dialog).getByText('“work” is connected: disconnect it to rename it.')).toBeInTheDocument()
    await userEvent.click(within(dialog).getByRole('button', { name: 'Save changes' }))
    expect(await within(dialog).findByText('Disconnect it and save the new name again.')).toBeInTheDocument()
    await waitFor(() => expect(name).toHaveAttribute('aria-invalid', 'true'))
  })

  it('imports a .conf: fills the path; a taken name offers Replace; a refused file says why and what to do', async () => {
    let calls = 0
    const fetchMock = stubFetch(
      base({
        'GET /api/v1/vpns': () => json([profile('work')]),
        'POST /api/v1/vpns/import': (init) => {
          calls++
          const body = JSON.parse(String(init?.body)) as { name: string; overwrite?: boolean; content_base64: string }
          if (atob(body.content_base64).includes('PostUp')) {
            return json(
              {
                error: 'the file runs commands (PostUp, line 3): Kongtrol does not allow that, for safety',
                hint: 'Delete that line from the file (Kongtrol already handles routes and DNS) and import it again.',
                error_key: 'api.err.import_hook',
              },
              400,
            )
          }
          if (calls === 1 && !body.overwrite) {
            return json({ error: 'a file "casa.conf" already exists', hint: 'Pick another name or confirm.', error_key: 'api.err.import_exists' }, 409)
          }
          return json({ status: 'imported', path: '/home/ana/.kongtrol/configs/casa.conf', type: 'wireguard', overwritten: !!body.overwrite }, 201)
        },
      }),
    )
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: /New VPN/ }))
    await userEvent.click(await screen.findByRole('button', { name: /WireGuard/ }))
    const dialog = await screen.findByRole('dialog')
    await userEvent.type(within(dialog).getByLabelText(/^Name/), 'casa')
    const input = dialog.querySelector<HTMLInputElement>('input[type="file"]')!
    expect(input).toHaveAttribute('accept', '.conf')

    // Wrong extension: refused here, nothing sent.
    await userEvent.upload(input, new File(['x'], 'notes.txt', { type: 'text/plain' }), { applyAccept: false })
    expect(await within(dialog).findByText('“notes.txt” is not a .conf file.')).toBeInTheDocument()
    expect(calls).toBe(0)

    const good = new File(['[Interface]\nPrivateKey = a\n'], 'office.conf')
    await userEvent.upload(input, good)
    expect(await within(dialog).findByText('a file "casa.conf" already exists')).toBeInTheDocument()
    await userEvent.click(within(dialog).getByRole('button', { name: 'Replace' }))
    expect(await within(dialog).findByText(/Replaced: the path is in the field/)).toBeInTheDocument()
    expect(within(dialog).getByLabelText(/\.conf file/)).toHaveValue('/home/ana/.kongtrol/configs/casa.conf')
    const sent = fetchMock.mock.calls.filter(([u]) => String(u) === '/api/v1/vpns/import').map(([, i]) => JSON.parse(String(i?.body)))
    expect(sent.map((b) => [b.name, b.type, b.overwrite])).toEqual([
      ['casa', 'wireguard', false],
      ['casa', 'wireguard', true],
    ])

    await userEvent.upload(input, new File(['[Interface]\nPrivateKey = a\nPostUp = x\n'], 'bad.conf'))
    expect(await within(dialog).findByText(/runs commands \(PostUp, line 3\)/)).toBeInTheDocument()
    expect(within(dialog).getByText(/Delete that line from the file/)).toBeInTheDocument()
  })

  it('does not offer delete while a group or another profile still needs the profile', async () => {
    const fetchMock = stubFetch(
      base({
        'GET /api/v1/vpns': () => json([profile('work', { failover: ['home'] }), profile('home'), profile('x')]),
        'GET /api/v1/groups': () => json([]),
      }),
    )
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: 'Delete home' }))
    const dialog = await screen.findByRole('alertdialog')
    expect(within(dialog).getByText('“home” is the backup of work.')).toBeInTheDocument()
    expect(within(dialog).queryByRole('button', { name: 'Delete' })).not.toBeInTheDocument()
    await userEvent.click(within(dialog).getByRole('button', { name: 'Got it' }))
    expect(fetchMock.mock.calls.some(([, i]) => i?.method === 'DELETE')).toBe(false)
  })

  it('filters the list and explains an empty result', async () => {
    stubFetch(base({ 'GET /api/v1/vpns': () => json(['a', 'b', 'c', 'work'].map((n) => profile(n))) }))
    render(<App />)
    const search = await screen.findByRole('searchbox', { name: 'Search VPNs' })
    await userEvent.type(search, 'zzz')
    expect(screen.getByRole('heading', { name: 'No VPN matches “zzz”' })).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Clear search' }))
    expect(screen.getByRole('table', { name: 'Configured VPN profiles' })).toBeInTheDocument()
  })

  it('shows the error with what to do when the list cannot load', async () => {
    stubFetch(base({ 'GET /api/v1/vpns': () => json({ error: 'config file is broken', hint: 'Fix kongtrol.yaml line 3.' }, 400) }))
    render(<App />)
    expect(await screen.findByText('config file is broken')).toBeInTheDocument()
    expect(screen.getByText('Fix kongtrol.yaml line 3.')).toBeInTheDocument()
  })
})
