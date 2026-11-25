import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { App } from '../App'
import { RESTART_TIMING, resetRestartPhase, type VPNProfile } from '../data/profiles'
import { FakeSocket, helperReady, json, pickOption, stubFetch, tunnel, type Routes } from '../test/fakes'

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

  it('suggests an installed VPN and opens the first profile in the same session', async () => {
    let saved = false
    const fetchMock = stubFetch(
      base({
        'GET /api/v1/vpns': () => json(saved ? [profile('nord', { type: 'nordvpn' })] : []),
        'GET /api/v1/setup': () => json({ ...setupDone, setup_mode: !saved, needs_setup: !saved }),
        'GET /api/v1/vpns/detect': () => json([{ label: 'NordVPN', adapter_type: 'nordvpn', path: 'C:/NordVPN/nordvpn.exe' }]),
        'POST /api/v1/vpns': () => {
          saved = true
          return json({ status: 'created', profile: 'nord', restart_required: false }, 201)
        },
      }),
    )
    render(<App />)
    expect(await screen.findByRole('heading', { name: 'Welcome to Kongtrol: create your first VPN' })).toBeInTheDocument()
    await userEvent.click(await screen.findByRole('button', { name: /Use NordVPN/ }))

    // The tile opens the shell's add sheet on its short form (ADR-0055).
    const dialog = await screen.findByRole('dialog', { name: 'Connect a VPN I already use' })
    expect(new URLSearchParams(window.location.search).get('add')).toBe('vpn')
    // NordVPN asks for no password: its app holds the session.
    expect(within(dialog).queryByLabelText('Password')).not.toBeInTheDocument()
    const name = within(dialog).getByLabelText(/^Name/)
    expect(name).toHaveValue('NordVPN')
    await userEvent.clear(name)
    await userEvent.type(name, 'nord')
    await userEvent.click(within(dialog).getByText('More options'))
    await userEvent.type(within(dialog).getByLabelText('Country (optional)'), 'CH')
    expect(within(dialog).getByText('Once saved, you can connect the profile or assign it a rule.')).toBeInTheDocument()
    await userEvent.click(within(dialog).getByRole('button', { name: 'Save and activate' }))

    await waitFor(() => expect(bodyOf(fetchMock, 'POST /api/v1/vpns')).toBeDefined())
    expect(bodyOf(fetchMock, 'POST /api/v1/vpns')).toEqual({ name: 'nord', type: 'nordvpn', priority: 10, auto_connect: true, country: 'CH' })
    expect(await screen.findByText('“nord” is ready.')).toBeInTheDocument()
    expect(fetchMock.mock.calls.some(([u, i]) => String(u) === '/api/v1/restart' && i?.method === 'POST')).toBe(false)
    await userEvent.click(screen.getByRole('button', { name: 'See it on Home' }))
    await waitFor(() => expect(window.location.pathname).toBe('/'))
  })

  it('checks the form inline before sending and focuses the first problem', async () => {
    const fetchMock = stubFetch(base({ 'GET /api/v1/vpns': () => json([profile('work'), profile('home')]) }))
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: 'Edit home' }))
    const dialog = await screen.findByRole('dialog', { name: 'Edit home' })
    await userEvent.clear(within(dialog).getByLabelText(/^Name/))
    await userEvent.type(within(dialog).getByLabelText(/^Name/), 'work')
    await userEvent.clear(within(dialog).getByLabelText(/OpenVPN file/))
    await userEvent.click(within(dialog).getByRole('button', { name: 'Save' }))

    expect(await within(dialog).findByText('You already have a VPN called “work”.')).toBeInTheDocument()
    expect(within(dialog).getByText('The OpenVPN file is missing.')).toBeInTheDocument()
    await waitFor(() => expect(within(dialog).getByLabelText(/^Name/)).toHaveFocus())
    expect(fetchMock.mock.calls.some(([, i]) => i?.method === 'PUT')).toBe(false)

    // Fixing a field clears its error on the next check.
    await userEvent.clear(within(dialog).getByLabelText(/^Name/))
    await userEvent.type(within(dialog).getByLabelText(/^Name/), 'office')
    expect(within(dialog).queryByText('You already have a VPN called “work”.')).not.toBeInTheDocument()
  })

  it.each([
    { supported: true, mode: '' },
    { supported: false, mode: '' },
    { supported: false, mode: 'all' },
  ] as const)('gates executable routing (supported=$supported, mode=$mode)', async ({ supported, mode }) => {
    const fetchMock = stubFetch(base({
      'GET /api/v1/vpns': () => json([profile('home', {
        type: 'wireguard',
        config: '/home/ana/wg0.conf',
        app_route_supported: supported,
        app_route_mode: mode,
      })]),
      'GET /api/v1/vpns/home/default-egress': () => json({ enabled: false, compatible: false }),
      'PUT /api/v1/vpns/home': () => json({ status: 'updated', profile: 'home', restart_required: false }),
    }))
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: 'Edit home' }))
    const dialog = within(await screen.findByRole('dialog', { name: 'Edit home' }))
    const selector = () => dialog.queryByLabelText('Route priority')
    if (supported) {
      expect(selector()).toBeEnabled()
      const file = dialog.getByDisplayValue('/home/ana/wg0.conf')
      await userEvent.type(file, '2')
      expect(selector()).not.toBeInTheDocument()
    } else {
      expect(selector()).not.toBeInTheDocument()
    }
    const turnOff = dialog.queryByRole('button', { name: 'Turn off executable egress' })
    if (mode) {
      expect(turnOff).toBeEnabled()
      await userEvent.click(turnOff!)
      await userEvent.click(dialog.getByRole('button', { name: 'Save' }))
      await waitFor(() => expect(bodyOf(fetchMock, 'PUT /api/v1/vpns/home')).toMatchObject({ app_route_mode: '' }))
    } else {
      expect(turnOff).not.toBeInTheDocument()
    }
  })

  it('selects native default egress from the saved WireGuard profile', async () => {
    let enabled = false
    const fetchMock = stubFetch(base({
      'GET /api/v1/vpns': () => json([profile('home', { type: 'wireguard', config: '/home/ana/wg0.conf' })]),
      'GET /api/v1/vpns/home/default-egress': () => json({ enabled, compatible: true }),
      'PUT /api/v1/vpns/home/default-egress': () => {
        enabled = true
        return json({ enabled, compatible: true })
      },
    }))
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: 'Edit home' }))
    const toggle = await screen.findByRole('switch', { name: 'WireGuard default egress' })
    expect(toggle).toHaveAttribute('aria-checked', 'false')
    await userEvent.click(toggle)
    await waitFor(() => expect(bodyOf(fetchMock, 'PUT /api/v1/vpns/home/default-egress')).toEqual({ enabled: true }))
    await waitFor(() => expect(toggle).toHaveAttribute('aria-checked', 'true'))
  })

  it('editing: a connected VPN saves with "Save and reconnect" (PUT ?reconnect=true); a disconnected one with "Save"', async () => {
    const fetchMock = stubFetch(
      base({
        'GET /api/v1/vpns': () => json([profile('work'), profile('home')]),
        'PUT /api/v1/vpns/work?reconnect=true': () => json({ status: 'updated', profile: 'work', restart_required: false, reconnected: true }),
        'PUT /api/v1/vpns/home': () => json({ status: 'updated', profile: 'home', restart_required: true }),
      }),
    )
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: 'Edit work' }))
    let dialog = await screen.findByRole('dialog', { name: 'Edit work' })
    expect(within(dialog).getByText('Only this VPN cuts for a few seconds.')).toBeInTheDocument()
    expect(within(dialog).queryByRole('button', { name: 'Save' })).not.toBeInTheDocument()
    await userEvent.click(within(dialog).getByRole('button', { name: 'Save and reconnect' }))
    expect(await screen.findByText('VPN “work” saved and reconnected.')).toBeInTheDocument()
    expect(fetchMock.mock.calls.some(([u, i]) => String(u) === '/api/v1/vpns/work?reconnect=true' && i?.method === 'PUT')).toBe(true)

    await userEvent.click(await screen.findByRole('button', { name: 'Edit home' }))
    dialog = await screen.findByRole('dialog', { name: 'Edit home' })
    expect(within(dialog).queryByText('Only this VPN cuts for a few seconds.')).not.toBeInTheDocument()
    expect(within(dialog).queryByRole('button', { name: 'Save and reconnect' })).not.toBeInTheDocument()
    await userEvent.click(within(dialog).getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(fetchMock.mock.calls.some(([u, i]) => String(u) === '/api/v1/vpns/home' && i?.method === 'PUT')).toBe(true))
  })

  it('editing a connected VPN on an older daemon: "Reconnecting…" while saving, then the toast offers Reconnect', async () => {
    let release = () => {}
    const held = new Promise<void>((r) => (release = r))
    const fetchMock = stubFetch(
      base({
        'GET /api/v1/vpns': () => json([profile('work')]),
        'PUT /api/v1/vpns/work?reconnect=true': async () => {
          await held
          return json({ status: 'updated', profile: 'work', restart_required: true, reconnect_required: true })
        },
        'POST /api/v1/tunnels/work/reload': () => json({ status: 'restarting', tunnel: 'work' }, 202),
      }),
    )
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: 'Edit work' }))
    const dialog = await screen.findByRole('dialog', { name: 'Edit work' })
    await userEvent.click(within(dialog).getByRole('button', { name: 'Save and reconnect' }))
    expect(await within(dialog).findByRole('button', { name: 'Reconnecting…' })).toHaveAttribute('aria-busy', 'true')
    release()
    expect(await screen.findByText('Reconnect it to use the changes: only this VPN cuts for a few seconds.')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Reconnect' }))
    await waitFor(() =>
      expect(fetchMock.mock.calls.some(([u, i]) => String(u) === '/api/v1/tunnels/work/reload' && i?.method === 'POST')).toBe(true),
    )
  })

  it('puts a daemon validation error on its field, with what to do', async () => {
    stubFetch(
      base({
        'GET /api/v1/vpns': () => json([profile('work'), profile('home', { failover: ['work'] })]),
        'PUT /api/v1/vpns/work?reconnect=true': () =>
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
    await userEvent.click(within(dialog).getByRole('button', { name: 'Save and reconnect' }))
    expect(await within(dialog).findByText('The backups form a loop: work → home → work.')).toBeInTheDocument()
    expect(within(dialog).getByText('Remove one of those failover entries.')).toBeInTheDocument()
  })

  it('adds a backup in order and refuses a loop before saving', async () => {
    const fetchMock = stubFetch(
      base({
        'GET /api/v1/vpns': () => json([profile('work'), profile('home', { failover: ['work'] }), profile('spare')]),
        'PUT /api/v1/vpns/work?reconnect=true': () => json({ status: 'updated', profile: 'work', restart_required: true }),
      }),
    )
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: 'Edit work' }))
    const dialog = await screen.findByRole('dialog', { name: 'Edit work' })
    const add = within(dialog).getByRole('button', { name: /Backup VPN to add/ })
    await pickOption(userEvent.setup(), add, 'home')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Add backup' }))
    await userEvent.click(within(dialog).getByRole('button', { name: 'Save and reconnect' }))
    expect(await within(dialog).findByText('The backups form a loop: work → home → work.')).toBeInTheDocument()

    await userEvent.click(within(dialog).getByRole('button', { name: 'Remove home' }))
    await pickOption(userEvent.setup(), within(dialog).getByRole('button', { name: /Backup VPN to add/ }), 'spare')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Add backup' }))
    await userEvent.click(within(dialog).getByRole('button', { name: 'Save and reconnect' }))
    await waitFor(() => expect(bodyOf(fetchMock, 'PUT /api/v1/vpns/work?reconnect=true')).toMatchObject({ failover: ['spare'] }))
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
    // The refusal arrives only after the row left the list (as with a real
    // daemon): the dialog must not depend on the list still having it.
    let answer = () => {}
    const answered = new Promise<void>((r) => (answer = r))
    stubFetch(
      base({
        'GET /api/v1/vpns': () => json([profile('work'), profile('home')]),
        'DELETE /api/v1/vpns/home': async () => {
          await answered
          return json(
            {
              error: 'profile "home" is in use · policies: intranet, git · groups: — · failover backup of: — · schedules: office',
              hint: 'Remove it from those places (or switch them to another profile) and delete it again.',
              error_key: 'api.err.vpn_in_use',
            },
            409,
          )
        },
      }),
    )
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: 'Delete home' }))
    const confirm = await screen.findByRole('alertdialog', { name: 'Delete the VPN “home”?' })
    await userEvent.click(within(confirm).getByRole('button', { name: 'Delete' }))
    await waitFor(() => expect(screen.queryByRole('button', { name: 'Delete home' })).not.toBeInTheDocument())
    answer()
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
    await userEvent.click(within(dialog).getByRole('button', { name: 'Save' }))
    expect(await screen.findByText('VPN “home” is now called “casa”.')).toBeInTheDocument()
    expect(bodyOf(fetchMock, 'PUT /api/v1/vpns/home')).toMatchObject({ name: 'casa' })
    expect(await screen.findByRole('button', { name: 'Restart now' })).toBeInTheDocument()
  })

  it('rename: a connected VPN says to disconnect it first, and the daemon refusal lands on the name', async () => {
    stubFetch(
      base({
        'GET /api/v1/vpns': () => json([profile('work'), profile('home')]),
        'PUT /api/v1/vpns/work?reconnect=true': () =>
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
    await userEvent.click(within(dialog).getByRole('button', { name: 'Save and reconnect' }))
    expect(await within(dialog).findByText('Disconnect it and save the new name again.')).toBeInTheDocument()
    await waitFor(() => expect(name).toHaveAttribute('aria-invalid', 'true'))
  })

  it('imports a .conf: fills the path; a taken name offers Replace; a refused file says why and what to do', async () => {
    let calls = 0
    const fetchMock = stubFetch(
      base({
        'GET /api/v1/vpns': () => json([profile('work'), profile('casa', { type: 'wireguard', config: '~/.kongtrol/configs/old.conf' })]),
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
    // Editing a WireGuard profile keeps the in-form import (new profiles use the add sheet).
    await userEvent.click(await screen.findByRole('button', { name: 'Edit casa' }))
    const dialog = await screen.findByRole('dialog')
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

describe('ProfilesPage: profiles applied live (ADR-0053)', () => {
  it('deleting a connected profile: the refusal says to disconnect it first, in the same dialog', async () => {
    stubFetch(
      base({
        'GET /api/v1/vpns': () => json([profile('work'), profile('home')]),
        'DELETE /api/v1/vpns/work': () =>
          json(
            { error: '"work" is connected or connecting: it cannot be deleted now', hint: 'Disconnect it and delete it again.', error_key: 'api.err.vpn_delete_busy' },
            409,
          ),
      }),
    )
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: 'Delete work' }))
    await userEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: 'Delete' }))
    const refused = await screen.findByRole('alertdialog', { name: 'Delete the VPN “work”?' })
    expect(await within(refused).findByText('"work" is connected or connecting: it cannot be deleted now')).toBeInTheDocument()
    expect(within(refused).getByText('Disconnect it and delete it again.')).toBeInTheDocument()
  })

  it('a reconnect that fails says so, keeps the settings and offers Reconnect', async () => {
    stubFetch(
      base({
        'GET /api/v1/vpns': () => json([profile('work')]),
        'PUT /api/v1/vpns/work?reconnect=true': () =>
          json({ status: 'updated', profile: 'work', restart_required: false, reconnect_required: true, reconnect_error: 'work: tunnel did not come up' }),
      }),
    )
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: 'Edit work' }))
    await userEvent.click(within(await screen.findByRole('dialog', { name: 'Edit work' })).getByRole('button', { name: 'Save and reconnect' }))
    expect(await screen.findByText('VPN “work” saved, but it could not reconnect.')).toBeInTheDocument()
    expect(screen.getByText(/work: tunnel did not come up · Press “Reconnect”/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Reconnect' })).toBeInTheDocument()
  })
})

describe('ProfilesPage: priority (ADR-0108)', () => {
  it('shows a set priority in the list; none when the position decides', async () => {
    stubFetch(base({ 'GET /api/v1/vpns': () => json([profile('work', { priority: 20 }), profile('home', { priority: 0 })]) }))
    render(<App />)
    const table = await screen.findByRole('table', { name: 'Configured VPN profiles' })
    const rows = within(table).getAllByRole('row')
    expect(within(rows.find((r) => within(r).queryByText('work'))!).getByText(/Priority 20/)).toBeInTheDocument()
    expect(within(rows.find((r) => within(r).queryByText('home'))!).queryByText(/Priority/)).not.toBeInTheDocument()
  })

  it('the deep link opens the editor on a visible priority field; a bad number says what to do', async () => {
    window.history.replaceState(null, '', '/profiles?edit=work&focus=priority')
    stubFetch(base({ 'GET /api/v1/vpns': () => json([profile('work', { priority: 0 }), profile('home')]) }))
    render(<App />)
    const dialog = await screen.findByRole('dialog', { name: 'Edit work' })
    const field = within(dialog).getByLabelText('Priority')
    await waitFor(() => expect(field).toHaveFocus())
    expect(field).toHaveValue('')
    expect(field).toHaveAttribute('placeholder', 'by position')
    expect(field.closest('details')).toBeNull()
    expect(within(dialog).getByText(/same site or the same IP range/)).toBeInTheDocument()
    expect(window.location.search).toBe('')
    await userEvent.type(field, '-3')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Save and reconnect' }))
    expect(await within(dialog).findByText('Priority must be a whole number from 1 to 1000.')).toBeInTheDocument()
    expect(within(dialog).getByText('Type a number, e.g. 10, or leave it empty to let the position decide.')).toBeInTheDocument()
  })
})
