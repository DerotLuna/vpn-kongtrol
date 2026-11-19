import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { App } from '../../App'
import { RESTART_TIMING, resetRestartPhase, type VPNProfile } from '../../data/profiles'
import { FakeSocket, helperReady, json, stubFetch, tunnel, type Routes } from '../../test/fakes'

function profile(name: string, type: VPNProfile['type'] = 'nordvpn'): VPNProfile {
  return {
    name,
    type,
    priority: 10,
    failover: [],
    has_username_credential: false,
    has_password_credential: false,
    kill_switch_override: 'inherit',
    auto_connect: true,
  }
}

const setupDone = { setup_mode: false, needs_setup: false, restart_pending: false, config_path: '' }

/** A daemon whose profile list can change and that restarts (one failed answer, then back). */
function daemon(start: VPNProfile[], extra: Routes = {}) {
  const state = { profiles: [...start], down: false }
  const routes: Routes = {
    'GET /api/v1/preferences': () => json({ language: 'en', theme: '' }),
    'GET /api/v1/version': () => json({ version: 'v0.7.0' }),
    'GET /api/v1/helper/status': () => json(helperReady),
    'GET /api/v1/tunnels': () => json([tunnel('work', 'connected'), tunnel('home', 'disconnected')]),
    'GET /api/v1/groups': () => json([]),
    'GET /api/v1/policies': () => json([]),
    'GET /api/v1/vpns': () => json(state.profiles),
    'GET /api/v1/vpns/detect': () => json([]),
    'GET /api/v1/setup': () => {
      if (state.down) {
        state.down = false
        return Promise.reject(new TypeError('down'))
      }
      return json(setupDone)
    },
    'POST /api/v1/restart': () => {
      state.down = true
      return json({ status: 'restarting' }, 202)
    },
    ...extra,
  }
  return { state, routes }
}

function bodyOf(fetchMock: ReturnType<typeof stubFetch>, key: string): unknown {
  const call = fetchMock.mock.calls.findLast(([url, init]) => `${init?.method ?? 'GET'} ${String(url)}` === key)
  return call ? JSON.parse(String(call[1]?.body)) : undefined
}

const called = (fetchMock: ReturnType<typeof stubFetch>, key: string) =>
  fetchMock.mock.calls.some(([url, init]) => `${init?.method ?? 'GET'} ${String(url)}` === key)

const timing = { ...RESTART_TIMING }

beforeEach(() => {
  Object.defineProperty(navigator, 'languages', { value: ['en-US'], configurable: true })
  FakeSocket.reset()
  vi.stubGlobal('WebSocket', FakeSocket)
  Object.assign(RESTART_TIMING, { stepMs: 1, stopSteps: 5, startSteps: 5 })
})

afterEach(() => {
  Object.assign(RESTART_TIMING, timing)
  resetRestartPhase()
})

const add = () => new URLSearchParams(window.location.search).get('add')

/** What the browser fires on a modal <dialog> when Esc is pressed (jsdom has no showModal). */
const esc = (dialog: HTMLElement) => fireEvent(dialog, new Event('cancel', { cancelable: true }))

describe('"＋ Add": one sheet, opened from the URL', () => {
  it('topbar button: menu with focus on the first choice; Esc closes, clears the URL and gives focus back', async () => {
    stubFetch(daemon([profile('work'), profile('home')]).routes)
    render(<App />)
    const opener = await screen.findByRole('button', { name: 'Add' })
    expect(opener).toHaveAttribute('aria-haspopup', 'dialog')
    await userEvent.click(opener)
    const sheet = await screen.findByRole('dialog', { name: 'Add' })
    expect(add()).toBe('menu')
    const first = within(sheet).getByRole('button', { name: /Connect a VPN I already use/ })
    await waitFor(() => expect(first).toHaveFocus())
    expect(within(sheet).getByRole('button', { name: /Import a VPN file/ })).toBeInTheDocument()
    expect(within(sheet).getByRole('button', { name: /Send a site through a VPN/ })).toBeInTheDocument()

    esc(sheet)
    await waitFor(() => expect(screen.queryByRole('dialog', { name: 'Add' })).not.toBeInTheDocument())
    expect(add()).toBeNull()
    await waitFor(() => expect(opener).toHaveFocus())
  })

  it('deep link ?add=rule&via=&target=: prefilled sentence, "Today / After" from the daemon, inline errors and hints while typing', async () => {
    window.history.replaceState(null, '', '/?add=rule&via=home&target=https%3A%2F%2Fclaude.ai%2Fchat')
    const fetchMock = stubFetch(
      daemon([profile('work'), profile('home')], {
        'GET /api/v1/policies/explain?target=claude.ai': () =>
          json({ Target: 'claude.ai', Kind: 'domain', Matched: false, Via: '', RuleName: '', Reason: '', ReasonCode: 'no_rule', ReasonArgs: null, DefaultTo: '' }),
        'POST /api/v1/policies/test': () => json({ matched: true, via: 'home', rule: 'claude.ai via home' }),
      }).routes,
    )
    render(<App />)
    const sheet = await screen.findByRole('dialog', { name: 'Send a site through a VPN' })
    const input = await within(sheet).findByRole('textbox', { name: /Site, IP or program/ })
    expect(input).toHaveValue('https://claude.ai/chat')
    expect(within(sheet).getByRole('combobox', { name: /Goes through/ })).toHaveValue('home')
    expect(within(sheet).getByRole('radio', { name: 'Use my normal connection' })).toBeChecked()
    expect(within(sheet).getByRole('checkbox', { name: /Include subdomains/ })).toBeChecked()
    expect(within(sheet).getByRole('textbox', { name: /Rule name/ })).toHaveValue('claude.ai via home')

    expect(await within(sheet).findByText('claude.ai goes over your normal connection.')).toBeInTheDocument()
    expect(
      await within(sheet).findByText('claude.ai will go through home while it is connected; while it is off, over your normal connection.'),
    ).toBeInTheDocument()
    expect(bodyOf(fetchMock, 'POST /api/v1/policies/test')).toMatchObject({
      target: 'claude.ai',
      rule: { via: 'home', match: { domains: ['claude.ai', '*.claude.ai'] } },
    })

    await userEvent.clear(input)
    await userEvent.type(input, 'intranet')
    // A bare word is valid (like the CLI) but gets a hint, not an error.
    expect(within(sheet).getByText(/“intranet” has no ending/)).toBeInTheDocument()
    expect(input).not.toHaveAttribute('aria-invalid')
    await userEvent.clear(input)
    await userEvent.type(input, 'teams.exe')
    expect(within(sheet).getByText(/If it is, type it as app:teams.exe./)).toBeInTheDocument()
    await userEvent.clear(input)
    await userEvent.type(input, 'not a host')
    expect(within(sheet).getByText('We do not understand “not a host”.')).toBeInTheDocument()
    expect(input).toHaveAttribute('aria-invalid', 'true')
  })

  it('Inicio: the row "＋" opens the sheet on the rule with that VPN', async () => {
    stubFetch(daemon([profile('work'), profile('home')]).routes)
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: 'Send a site through home' }))
    const sheet = await screen.findByRole('dialog', { name: 'Send a site through a VPN' })
    expect(new URLSearchParams(window.location.search).get('via')).toBe('home')
    expect(await within(sheet).findByRole('combobox', { name: /Goes through/ })).toHaveValue('home')
  })

  it('Profiles: "New VPN" opens the sheet on "Connect a VPN I already use"', async () => {
    window.history.replaceState(null, '', '/profiles')
    stubFetch(daemon([profile('work'), profile('home')]).routes)
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: 'New VPN' }))
    expect(await screen.findByRole('dialog', { name: 'Connect a VPN I already use' })).toBeInTheDocument()
    expect(add()).toBe('vpn')
  })

  it('Activity: a notice about a profile offers "＋" for that VPN', async () => {
    window.history.replaceState(null, '', '/activity')
    stubFetch(
      daemon([profile('work'), profile('home')], {
        'GET /api/v1/events?after=0': () =>
          json({
            seq: 1,
            events: [
              {
                seq: 1,
                time: new Date().toISOString(),
                kind: 'policy',
                level: 'WARN',
                key: 'policy.event.allow',
                args: ['intranet.example', 'home'],
                profile: 'home',
                can_connect: true,
              },
            ],
          }),
      }).routes,
    )
    render(<App />)
    const list = await screen.findByRole('region', { name: /Notices/ })
    await userEvent.click(within(list).getByRole('button', { name: 'Send a site through home' }))
    expect(await screen.findByRole('dialog', { name: 'Send a site through a VPN' })).toBeInTheDocument()
    expect(new URLSearchParams(window.location.search).get('via')).toBe('home')
  })

  it('Traffic: a lookup result offers "Send this through…" with the site prefilled', async () => {
    window.history.replaceState(null, '', '/traffic')
    stubFetch(
      daemon([profile('work'), profile('home')], {
        'GET /api/v1/resolve?target=claude.ai': () => json({ target: 'claude.ai', via: '', rule: '', matched: false }),
      }).routes,
    )
    render(<App />)
    await userEvent.type(await screen.findByRole('textbox', { name: 'Site, IP or program' }), 'claude.ai{Enter}')
    await userEvent.click(await screen.findByRole('button', { name: 'Send this through…' }))
    const sheet = await screen.findByRole('dialog', { name: 'Send a site through a VPN' })
    expect(await within(sheet).findByRole('textbox', { name: /Site, IP or program/ })).toHaveValue('claude.ai')
  })
})

describe('A: connect a VPN I already use', () => {
  it('found app first → name deduped ("NordVPN 2") → Save and activate = create + restart → Done', async () => {
    window.history.replaceState(null, '', '/?add=vpn')
    const { routes, state } = daemon([profile('NordVPN')], {
      'GET /api/v1/vpns/detect': () =>
        json([
          { label: 'NordVPN', adapter_type: 'nordvpn', path: 'C:/NordVPN/nordvpn.exe' },
          { label: 'Mystery VPN', adapter_type: 'mystery', path: 'C:/m.exe' },
        ]),
      'POST /api/v1/vpns': () => {
        state.profiles.push(profile('NordVPN 2'))
        return json({ status: 'created', profile: 'NordVPN 2', restart_required: true }, 201)
      },
      'POST /api/v1/tunnels/NordVPN%202/connect': () => json({ status: 'connecting', tunnel: 'NordVPN 2' }, 202),
    })
    const fetchMock = stubFetch(routes)
    render(<App />)
    const sheet = await screen.findByRole('dialog', { name: 'Connect a VPN I already use' })
    expect(await within(sheet).findByText('You already have it as “NordVPN”')).toBeInTheDocument()
    expect(within(sheet).getByText('We also saw Mystery VPN, which Kongtrol does not handle yet.')).toBeInTheDocument()
    const use = within(sheet).getByRole('button', { name: 'Use NordVPN' })
    await waitFor(() => expect(use).toHaveFocus())
    await userEvent.click(use)

    const name = within(sheet).getByRole('textbox', { name: /^Name/ })
    expect(name).toHaveValue('NordVPN 2')
    await waitFor(() => expect(name).toHaveFocus())
    expect(within(sheet).getByText(/connected VPNs reconnect after a brief cut/)).toBeInTheDocument()
    await userEvent.click(within(sheet).getByRole('button', { name: 'Save and activate' }))

    expect(await within(sheet).findByText('“NordVPN 2” is ready.')).toBeInTheDocument()
    expect(bodyOf(fetchMock, 'POST /api/v1/vpns')).toEqual({ name: 'NordVPN 2', type: 'nordvpn', priority: 10, auto_connect: true })
    const order = fetchMock.mock.calls.map(([u, i]) => `${i?.method ?? 'GET'} ${String(u)}`)
    expect(order.indexOf('POST /api/v1/restart')).toBeGreaterThan(order.indexOf('POST /api/v1/vpns'))
    await userEvent.click(within(sheet).getByRole('button', { name: 'Connect NordVPN 2' }))
    await waitFor(() => expect(called(fetchMock, 'POST /api/v1/tunnels/NordVPN%202/connect')).toBe(true))
    await waitFor(() => expect(window.location.pathname).toBe('/'))
  })

  it('corporate client asks for the gateway; a name the daemon would refuse is caught first', async () => {
    window.history.replaceState(null, '', '/?add=vpn&type=forticlient')
    const fetchMock = stubFetch(daemon([profile('work')]).routes)
    render(<App />)
    const sheet = await screen.findByRole('dialog', { name: 'Connect a VPN I already use' })
    const name = await within(sheet).findByRole('textbox', { name: /^Name/ })
    expect(name).toHaveValue('FortiClient')
    await userEvent.type(name, '<x>')
    await userEvent.click(within(sheet).getByRole('button', { name: 'Save and activate' }))
    expect(await within(sheet).findByText('That name cannot be used.')).toBeInTheDocument()
    expect(within(sheet).getByText('The server is missing.')).toBeInTheDocument()
    expect(called(fetchMock, 'POST /api/v1/vpns')).toBe(false)
  })
})

describe('C with no profile: add the VPN first, then back to the rule with the draft kept', () => {
  it('rule → "Add a VPN" → A → restart → back on the rule with that VPN', async () => {
    window.history.replaceState(null, '', '/?add=rule&target=claude.ai')
    const { routes, state } = daemon([], {
      'GET /api/v1/vpns/detect': () => json([{ label: 'Surfshark', adapter_type: 'surfshark', path: 'C:/s.exe' }]),
      'POST /api/v1/vpns': () => {
        state.profiles.push(profile('Surfshark', 'surfshark'))
        return json({ status: 'created', profile: 'Surfshark', restart_required: true }, 201)
      },
      'POST /api/v1/policies': () => json({ status: 'created' }, 201),
    })
    const fetchMock = stubFetch(routes)
    render(<App />)
    const sheet = await screen.findByRole('dialog', { name: 'Send a site through a VPN' })
    expect(await within(sheet).findByText('First add the VPN it will go through.')).toBeInTheDocument()
    await userEvent.click(within(sheet).getByRole('button', { name: 'Add a VPN' }))
    await userEvent.click(await within(sheet).findByRole('button', { name: 'Use Surfshark' }))
    await userEvent.click(within(sheet).getByRole('button', { name: 'Save and activate' }))

    expect(await within(sheet).findByText('Done: “Surfshark” is added. Go on with your rule.')).toBeInTheDocument()
    expect(within(sheet).getByRole('textbox', { name: /Site, IP or program/ })).toHaveValue('claude.ai')
    expect(within(sheet).getByRole('combobox', { name: /Goes through/ })).toHaveValue('Surfshark')
    await userEvent.click(within(sheet).getByRole('button', { name: 'Save rule' }))
    expect(await within(sheet).findByText('The rule “claude.ai via Surfshark” is saved.')).toBeInTheDocument()
    expect(bodyOf(fetchMock, 'POST /api/v1/policies')).toMatchObject({ via: 'Surfshark', match: { domains: ['claude.ai', '*.claude.ai'] } })
  })
})

describe('B: import a VPN file', () => {
  it('.ovpn with auth-user-pass shows user + password; a taken file offers Replace; then create + restart', async () => {
    window.history.replaceState(null, '', '/?add=file')
    let imports = 0
    const { routes, state } = daemon([profile('work')], {
      'POST /api/v1/vpns/import': (init) => {
        imports++
        const body = JSON.parse(String(init?.body)) as { overwrite?: boolean }
        if (!body.overwrite) {
          return json({ error: 'a file "Oficina.ovpn" already exists', hint: 'Pick another name or confirm.', error_key: 'api.err.import_exists' }, 409)
        }
        return json({ status: 'imported', path: '/home/ana/.kongtrol/configs/Oficina.ovpn', type: 'openvpn', overwritten: true }, 201)
      },
      'POST /api/v1/vpns': () => {
        state.profiles.push(profile('Oficina', 'openvpn'))
        return json({ status: 'created', profile: 'Oficina', restart_required: true }, 201)
      },
    })
    const fetchMock = stubFetch(routes)
    render(<App />)
    const sheet = await screen.findByRole('dialog', { name: 'Import a VPN file' })
    const chooser = await within(sheet).findByRole('button', { name: 'Choose file' })
    const input = sheet.querySelector<HTMLInputElement>('input[type="file"]')!

    await userEvent.upload(input, new File(['x'], 'notes.txt'), { applyAccept: false })
    expect(await within(sheet).findByText('“notes.txt” is not a VPN file Kongtrol reads.')).toBeInTheDocument()

    await userEvent.upload(input, new File(['client\nremote vpn.acme.test 1194\nauth-user-pass\n'], 'Oficina.ovpn'))
    expect(await within(sheet).findByText('Oficina.ovpn · OpenVPN')).toBeInTheDocument()
    expect(chooser).toHaveTextContent('Choose another file')
    expect(within(sheet).getByText('This file asks for a username and password.')).toBeInTheDocument()
    expect(within(sheet).getByRole('textbox', { name: /^Name/ })).toHaveValue('Oficina')
    expect(within(sheet).getByRole('combobox', { name: 'How it signs in' })).toHaveValue('credentials')
    await userEvent.type(within(sheet).getByRole('textbox', { name: 'Username' }), 'ana')
    await userEvent.type(within(sheet).getByLabelText('Password'), 's3cret')
    await userEvent.click(within(sheet).getByRole('button', { name: 'Save and activate' }))

    expect(await within(sheet).findByText('a file "Oficina.ovpn" already exists')).toBeInTheDocument()
    expect(within(sheet).getByText('Pick another name or confirm.')).toBeInTheDocument()
    await userEvent.click(within(sheet).getByRole('button', { name: 'Replace' }))

    expect(await within(sheet).findByText('“Oficina” is ready.')).toBeInTheDocument()
    expect(imports).toBe(2)
    expect(bodyOf(fetchMock, 'POST /api/v1/vpns')).toEqual({
      name: 'Oficina',
      type: 'openvpn',
      priority: 10,
      auto_connect: true,
      config: '/home/ana/.kongtrol/configs/Oficina.ovpn',
      auth_method: 'credentials',
      username: 'ana',
      password: 's3cret',
    })
    expect(within(sheet).getByText('/home/ana/.kongtrol/configs/Oficina.ovpn')).toBeInTheDocument()
  })
})

describe('drafts are protected', () => {
  it('Esc with something typed asks first; "Keep editing" keeps it, "Discard" closes', async () => {
    window.history.replaceState(null, '', '/?add=rule')
    stubFetch(daemon([profile('work')]).routes)
    render(<App />)
    const sheet = await screen.findByRole('dialog', { name: 'Send a site through a VPN' })
    await userEvent.type(await within(sheet).findByRole('textbox', { name: /Site, IP or program/ }), 'claude.ai')
    esc(sheet)
    expect(await within(sheet).findByText('You have unsaved changes')).toBeInTheDocument()
    await userEvent.click(within(sheet).getByRole('button', { name: 'Keep editing' }))
    expect(within(sheet).getByRole('textbox', { name: /Site, IP or program/ })).toHaveValue('claude.ai')
    esc(sheet)
    await userEvent.click(await within(sheet).findByRole('button', { name: 'Discard' }))
    await waitFor(() => expect(screen.queryByRole('dialog', { name: 'Send a site through a VPN' })).not.toBeInTheDocument())
    expect(add()).toBeNull()
  })
})

describe('applied live (restart_required: false) and apps connected now', () => {
  it('a profile the daemon applied live needs no restart; the app connected now comes first, marked', async () => {
    window.history.replaceState(null, '', '/?add=vpn')
    const { routes, state } = daemon([profile('work')], {
      'GET /api/v1/vpns/detect': () =>
        json([
          { label: 'NordVPN', adapter_type: 'nordvpn', path: 'C:/n.exe', active: false },
          { label: 'Surfshark', adapter_type: 'surfshark', path: 'C:/s.exe', active: true, interface: 'Surfshark DCO adapter' },
          { label: 'FortiClient', adapter_type: 'forticlient', path: 'C:/f.exe', active: null },
        ]),
      'POST /api/v1/vpns': () => {
        state.profiles.push(profile('Surfshark', 'surfshark'))
        return json({ status: 'created', profile: 'Surfshark', restart_required: false }, 201)
      },
    })
    const fetchMock = stubFetch(routes)
    render(<App />)
    const sheet = await screen.findByRole('dialog', { name: 'Connect a VPN I already use' })
    const rows = await within(sheet).findAllByRole('listitem')
    expect(rows[0]).toHaveTextContent('Surfshark')
    expect(within(rows[0]!).getByText('Connected now')).toBeInTheDocument()
    expect(within(sheet).getAllByText('Connected now')).toHaveLength(1)
    await userEvent.click(within(rows[0]!).getByRole('button', { name: 'Use Surfshark' }))
    await userEvent.click(within(sheet).getByRole('button', { name: 'Save and activate' }))
    expect(await within(sheet).findByText('“Surfshark” is ready.')).toBeInTheDocument()
    expect(called(fetchMock, 'POST /api/v1/restart')).toBe(false)
  })
})

describe('Traffic › Recent', () => {
  const recent = (body: unknown) => ({ 'GET /api/v1/connections/recent?limit=50': () => json(body) })

  it('lists recent destinations; "Send this through…" opens the rule sheet with that destination', async () => {
    window.history.replaceState(null, '', '/traffic')
    stubFetch(
      daemon([profile('work'), profile('home')], {
        ...recent({
          supported: true,
          enabled: true,
          max: 500,
          ttl_seconds: 3600,
          connections: [
            { host: 'claude.ai', remote_ip: '160.79.104.10', port: 443, process: 'chrome.exe', via: 'work', first_seen: new Date().toISOString(), last_seen: new Date().toISOString(), count: 3 },
            { remote_ip: '10.2.0.5', port: 22, first_seen: new Date().toISOString(), last_seen: new Date().toISOString(), count: 1 },
          ],
        }),
      }).routes,
    )
    render(<App />)
    const table = await screen.findByRole('table', { name: 'Recent connections, newest first' })
    const rows = within(table).getAllByRole('row').slice(1)
    expect(rows[0]).toHaveTextContent('claude.ai')
    expect(rows[0]).toHaveTextContent('chrome.exe')
    expect(rows[0]).toHaveTextContent('work')
    expect(rows[1]).toHaveTextContent('10.2.0.5')
    expect(rows[1]).toHaveTextContent('Normal connection')
    await userEvent.click(within(rows[1]!).getByRole('button', { name: 'Send this through…' }))
    const sheet = await screen.findByRole('dialog', { name: 'Send a site through a VPN' })
    expect(await within(sheet).findByRole('textbox', { name: /Site, IP or program/ })).toHaveValue('10.2.0.5')
  })

  it('off: says where to turn it on; unsupported OS: says why; older daemon: no card', async () => {
    window.history.replaceState(null, '', '/traffic')
    stubFetch(daemon([profile('work')], recent({ supported: true, enabled: false, max: 500, ttl_seconds: 3600, connections: [] })).routes)
    const { unmount } = render(<App />)
    expect(await screen.findByText('The recent connections list is off.')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Go to Privacy' })).toHaveAttribute('href', '/settings#privacy')
    unmount()

    stubFetch(daemon([profile('work')], recent({ supported: false, enabled: true, max: 500, ttl_seconds: 3600, connections: null })).routes)
    const second = render(<App />)
    expect(await screen.findByText(/cannot see connections yet without administrator rights/)).toBeInTheDocument()
    second.unmount()

    stubFetch(daemon([profile('work')]).routes)
    render(<App />)
    expect(await screen.findByRole('heading', { name: 'Traffic' })).toBeInTheDocument()
    await waitFor(() => expect(screen.queryByText('Recent')).not.toBeInTheDocument())
  })

  it('Settings › Privacy: the switch turns the list on (memory only)', async () => {
    window.history.replaceState(null, '', '/settings')
    const fetchMock = stubFetch(
      daemon([profile('work')], {
        'GET /api/v1/settings': () =>
          json({ dashboard_bind: '127.0.0.1', dashboard_port: 9741, health_check_interval: '30s', health_check_timeout: '5s', audit_log_path: '', audit_log_max_size_mb: 10, audit_log_sign: false, recent_connections_enabled: false }),
        'PUT /api/v1/settings': (init) => json({ ...JSON.parse(String(init?.body)) }),
      }).routes,
    )
    render(<App />)
    const sw = await screen.findByRole('switch', { name: /Recent connections \(Traffic\)/ })
    expect(screen.getByText('Kept in memory only, cleared when Kongtrol closes.')).toBeInTheDocument()
    await userEvent.click(sw)
    await waitFor(() => expect(bodyOf(fetchMock, 'PUT /api/v1/settings')).toEqual({ recent_connections_enabled: true }))
  })
})
