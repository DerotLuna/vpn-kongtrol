import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { App } from '../../App'
import { resetSessionForTests, startAuth } from '../../api/auth'
import { RESTART_TIMING, resetRestartPhase, type VPNProfile } from '../../data/profiles'
import { FakeSocket, isImportCheck, passedCheck, helperReady, json, stubFetch, tunnel, type Routes } from '../../test/fakes'

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

/** The last body sent to that route (the file step's automatic checks are not imports, ADR-0118). */
function bodyOf(fetchMock: ReturnType<typeof stubFetch>, key: string): unknown {
  const call = fetchMock.mock.calls.findLast(([url, init]) => `${init?.method ?? 'GET'} ${String(url)}` === key && !isImportCheck(init))
  return call ? JSON.parse(String(call[1]?.body)) : undefined
}

/** The automatic check passed (ADR-0118): the save buttons are on. */
const ready = (sheet: HTMLElement) => within(sheet).findByText(/^Ready: /)

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
    expect(within(sheet).getByRole('button', { name: /Add routing rule/ })).toBeInTheDocument()

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
    const sheet = await screen.findByRole('dialog', { name: 'Add routing rule' })
    const input = await within(sheet).findByRole<HTMLTextAreaElement>('textbox', { name: /Site, IP or program/ })
    expect(input).toHaveValue('https://claude.ai/chat')
    expect(within(sheet).getByRole('button', { name: /Outgoing profile/ })).toHaveTextContent('home · NordVPN · app')
    expect(within(sheet).getByRole('radio', { name: 'Use the profile setting' })).toBeChecked()
    expect(within(sheet).getByRole('checkbox', { name: /Include subdomains/ })).toBeChecked()
    expect(within(sheet).getByRole('textbox', { name: /Rule name/ })).toHaveValue('claude.ai via home')

    expect(await within(sheet).findByText('claude.ai goes over your normal connection.')).toBeInTheDocument()
    expect(
      await within(sheet).findByText('claude.ai will go through home while it is connected; while it is off, over your normal connection.'),
    ).toBeInTheDocument()
    expect(bodyOf(fetchMock, 'POST /api/v1/policies/test')).toMatchObject({
      target: 'claude.ai',
      rule: { via: 'home', on_tunnel_down: '', match: { domains: ['claude.ai', '*.claude.ai'] } },
    })

    await userEvent.clear(input)
    await userEvent.type(input, 'intranet')
    // A bare word is valid (like the CLI) but gets a hint, not an error.
    expect(within(sheet).getByText(/“intranet” has no ending/)).toBeInTheDocument()
    expect(input).not.toHaveAttribute('aria-invalid')
    await userEvent.clear(input)
    await userEvent.type(input, 'teams.exe')
    expect(within(sheet).getByText(/Programs are not sent through the VPN yet; for now Kongtrol only recognises them./)).toBeInTheDocument()
    await userEvent.clear(input)
    await userEvent.type(input, 'not a host')
    expect(within(sheet).getByText('We do not understand “not a host”.')).toBeInTheDocument()
    expect(input).toHaveAttribute('aria-invalid', 'true')
  })

  it('adds editable suggestions without replacing existing destinations', async () => {
    window.history.replaceState(null, '', '/?add=rule&via=work&target=example.com')
    stubFetch(daemon([profile('work')]).routes)
    render(<App />)
    const sheet = await screen.findByRole('dialog', { name: 'Add routing rule' })
    const input = await within(sheet).findByRole<HTMLTextAreaElement>('textbox', { name: /Site, IP or program/ })
    await userEvent.click(within(sheet).getByRole('button', { name: 'Claude' }))
    await userEvent.click(within(sheet).getByRole('button', { name: 'ChatGPT / OpenAI' }))
    await userEvent.click(within(sheet).getByRole('button', { name: 'Oracle / Java' }))
    expect(input.value).toContain('*.example.com')
    expect(input.value).toContain('*.anthropic.com')
    expect(input.value).toContain('auth.openai.com')
    expect(input.value).toContain('download.oracle.com')
    expect(input.value).toContain('javadl.oracle.com')
    expect(input.value).toContain('jdk.java.net')
    for (const name of ['X / Twitter', 'Reddit', 'Telegram Web', 'Canva', 'Punto de Corte']) {
      expect(within(sheet).queryByRole('button', { name })).not.toBeInTheDocument()
    }
    await userEvent.clear(input)
    await userEvent.type(input, 'only.example')
    expect(input).toHaveValue('only.example')
  })

  it('Inicio: the row "＋" opens the sheet on the rule with that VPN', async () => {
    stubFetch(daemon([profile('work'), profile('home')]).routes)
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: 'Add a routing rule through profile home' }))
    const sheet = await screen.findByRole('dialog', { name: 'Add routing rule' })
    expect(new URLSearchParams(window.location.search).get('via')).toBe('home')
    expect(await within(sheet).findByRole('button', { name: /Outgoing profile/ })).toHaveTextContent(/home/)
  })

  it('Profiles: "New VPN profile" opens the sheet on "Connect a VPN I already use"', async () => {
    window.history.replaceState(null, '', '/profiles')
    stubFetch(daemon([profile('work'), profile('home')]).routes)
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: 'New VPN profile' }))
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
    await userEvent.click(within(list).getByRole('button', { name: 'Add a routing rule through profile home' }))
    expect(await screen.findByRole('dialog', { name: 'Add routing rule' })).toBeInTheDocument()
    expect(new URLSearchParams(window.location.search).get('via')).toBe('home')
  })

  it('Traffic: a lookup result offers "Add routing rule for this" with the site prefilled', async () => {
    window.history.replaceState(null, '', '/traffic')
    stubFetch(
      daemon([profile('work'), profile('home')], {
        'GET /api/v1/resolve?target=claude.ai': () => json({ target: 'claude.ai', via: '', rule: '', matched: false }),
      }).routes,
    )
    render(<App />)
    await userEvent.type(await screen.findByRole('textbox', { name: 'Site, IP or program' }), 'claude.ai{Enter}')
    await userEvent.click(await screen.findByRole('button', { name: 'Add routing rule for this' }))
    const sheet = await screen.findByRole('dialog', { name: 'Add routing rule' })
    expect(await within(sheet).findByRole('textbox', { name: /Site, IP or program/ })).toHaveValue('claude.ai')
  })
})

describe('A: connect a VPN I already use', () => {
  it('found app first → name deduped ("NordVPN 2") → Save = create + restart, no connect → Done offers Connect', async () => {
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
    expect(within(sheet).getByText('“Save and activate” saves and connects it. “Save” only saves it: connect it whenever you want.')).toBeInTheDocument()
    expect(within(sheet).getByRole('button', { name: 'Save and activate' })).toBeInTheDocument()
    await userEvent.click(within(sheet).getByRole('button', { name: 'Save' }))

    expect(await within(sheet).findByText('“NordVPN 2” is ready.')).toBeInTheDocument()
    expect(called(fetchMock, 'POST /api/v1/tunnels/NordVPN%202/connect')).toBe(false)
    expect(bodyOf(fetchMock, 'POST /api/v1/vpns')).toEqual({ name: 'NordVPN 2', type: 'nordvpn', priority: 10, auto_connect: true })
    const order = fetchMock.mock.calls.map(([u, i]) => `${i?.method ?? 'GET'} ${String(u)}`)
    expect(order.indexOf('POST /api/v1/restart')).toBeGreaterThan(order.indexOf('POST /api/v1/vpns'))
    await userEvent.click(within(sheet).getByRole('button', { name: 'Connect NordVPN 2' }))
    await waitFor(() => expect(called(fetchMock, 'POST /api/v1/tunnels/NordVPN%202/connect')).toBe(true))
    await waitFor(() => expect(window.location.pathname).toBe('/'))
  })

  it('"Save and activate" creates the profile and connects it at once (ADR-0115)', async () => {
    window.history.replaceState(null, '', '/?add=vpn')
    const { routes, state } = daemon([], {
      'GET /api/v1/vpns/detect': () => json([{ label: 'NordVPN', adapter_type: 'nordvpn', path: 'C:/NordVPN/nordvpn.exe' }]),
      'POST /api/v1/vpns': () => {
        state.profiles.push(profile('NordVPN'))
        return json({ status: 'created', profile: 'NordVPN', restart_required: false }, 201)
      },
      'POST /api/v1/tunnels/NordVPN/connect': () => json({ status: 'connecting', tunnel: 'NordVPN' }, 202),
    })
    const fetchMock = stubFetch(routes)
    render(<App />)
    const sheet = await screen.findByRole('dialog', { name: 'Connect a VPN I already use' })
    await userEvent.click(await within(sheet).findByRole('button', { name: 'Use NordVPN' }))
    await userEvent.click(within(sheet).getByRole('button', { name: /When this profile’s connection is disconnected/ }))
    await userEvent.click(await screen.findByRole('option', { name: 'Block and try to connect the profile' }))
    await userEvent.click(within(sheet).getByRole('button', { name: 'Save and activate' }))

    expect(await within(sheet).findByText('“NordVPN” is saved and connecting.')).toBeInTheDocument()
    expect(bodyOf(fetchMock, 'POST /api/v1/vpns')).toMatchObject({ on_tunnel_down: 'connect' })
    await waitFor(() => expect(called(fetchMock, 'POST /api/v1/tunnels/NordVPN/connect')).toBe(true))
    const order = fetchMock.mock.calls.map(([u, i]) => `${i?.method ?? 'GET'} ${String(u)}`)
    expect(order.indexOf('POST /api/v1/tunnels/NordVPN/connect')).toBeGreaterThan(order.indexOf('POST /api/v1/vpns'))
    // Already connecting: no "Connect" choice on the Done step.
    expect(within(sheet).queryByRole('button', { name: 'Connect NordVPN' })).toBeNull()
    await userEvent.click(within(sheet).getByRole('button', { name: 'Add routing rule' }))
    const destinations = await within(sheet).findByRole<HTMLTextAreaElement>('textbox', { name: /Site, IP or program/ })
    expect(destinations.value).toBe('')
    await userEvent.click(within(sheet).getByRole('button', { name: 'Claude' }))
    expect(destinations.value).toContain('claude.ai')
    expect(within(sheet).getByRole('button', { name: /Outgoing profile/ })).toHaveTextContent('NordVPN')
    await userEvent.click(within(sheet).getByRole('button', { name: 'ChatGPT / OpenAI' }))
    expect(destinations.value).toContain('chatgpt.com')
    expect(called(fetchMock, 'POST /api/v1/policies')).toBe(false)
    await userEvent.click(within(sheet).getByRole('button', { name: 'Back' }))
    expect(within(sheet).getByText('“NordVPN” is saved and connecting.')).toBeInTheDocument()
    await userEvent.click(within(sheet).getByRole('button', { name: 'Add routing rule' }))
    expect(within(sheet).getByRole<HTMLTextAreaElement>('textbox', { name: /Site, IP or program/ }).value).toContain('chatgpt.com')
    await userEvent.click(within(sheet).getByRole('button', { name: 'Back' }))
    esc(sheet)
    expect(await within(sheet).findByText('You have unsaved changes')).toBeInTheDocument()
  })

  it('corporate client asks for the gateway; a name the daemon would refuse is caught first', async () => {
    window.history.replaceState(null, '', '/?add=vpn&type=forticlient')
    const fetchMock = stubFetch(daemon([profile('work')]).routes)
    render(<App />)
    const sheet = await screen.findByRole('dialog', { name: 'Connect a VPN I already use' })
    const name = await within(sheet).findByRole('textbox', { name: /^Name/ })
    expect(name).toHaveValue('FortiClient')
    await userEvent.type(name, '<x>')
    await userEvent.click(within(sheet).getByRole('button', { name: 'Save' }))
    expect(await within(sheet).findByText('That name cannot be used.')).toBeInTheDocument()
    expect(within(sheet).getByText('The server is missing.')).toBeInTheDocument()
    expect(called(fetchMock, 'POST /api/v1/vpns')).toBe(false)
  })
})

describe('C with no profile: add the VPN first, then back to the rule with the draft kept', () => {
  it('recovers the first rule after the replacement daemon rejects the old session', async () => {
    window.history.replaceState(null, '', '/rules?add=rule&target=example.net#k=first')
    let replaced = false
    let down = false
    const { routes, state } = daemon([], {
      'POST /api/v1/session': (init) => {
        const { nonce } = JSON.parse(String(init?.body))
        if (nonce !== (replaced ? 'reopened' : 'first')) return json({}, 401)
        return json({ session_token: replaced ? 'new-page' : 'old-page' })
      },
      'GET /api/v1/vpns/detect': () => json([{ label: 'Surfshark', adapter_type: 'surfshark' }]),
      'POST /api/v1/vpns': () => {
        state.profiles.push(profile('Surfshark', 'surfshark'))
        return json({ status: 'created', profile: 'Surfshark', restart_required: true }, 201)
      },
      'POST /api/v1/restart': () => {
        down = true
        return json({ status: 'restarting' }, 202)
      },
      'GET /api/v1/setup': () => {
        if (down) {
          down = false
          replaced = true
          return Promise.reject(new TypeError('down'))
        }
        return json(setupDone)
      },
      'POST /api/v1/ws-ticket': () => json({ ticket: 'one-shot' }),
      'POST /api/v1/policies': () => json({ status: 'created' }, 201),
    })
    for (const [key, handler] of Object.entries(routes)) {
      if (key === 'POST /api/v1/session') continue
      routes[key] = (init) => {
        const token = new Headers(init?.headers).get('X-Kongtrol-Session')
        return token === (replaced ? 'new-page' : 'old-page') ? handler(init) : json({}, 401)
      }
    }
    const fetchMock = stubFetch(routes)
    await startAuth()
    const first = render(<App />)
    const sheet = await screen.findByRole('dialog', { name: 'Add routing rule' })
    await userEvent.click(await within(sheet).findByRole('button', { name: 'Add VPN profile' }))
    await userEvent.click(await within(sheet).findByRole('button', { name: 'Use Surfshark' }))
    await userEvent.click(within(sheet).getByRole('button', { name: 'Save' }))
    expect(await within(sheet).findByText('Your profile was saved. Reopen the dashboard to continue.')).toBeInTheDocument()
    expect(within(sheet).getByText('example.net')).toBeInTheDocument()
    expect(screen.getAllByRole('alert')).toHaveLength(1)
    expect(screen.queryByText(/Reconnecting to Kongtrol/)).not.toBeInTheDocument()
    expect(called(fetchMock, 'POST /api/v1/policies')).toBe(false)

    first.unmount()
    resetSessionForTests()
    window.history.replaceState(null, '', '/rules#k=reopened')
    await startAuth()
    render(<App />)
    const resumed = await screen.findByRole('dialog', { name: 'Add routing rule' })
    expect(await within(resumed).findByRole('textbox', { name: /Site, IP or program/ })).toHaveValue('example.net')
    expect(within(resumed).getByRole('button', { name: /Outgoing profile/ })).toHaveTextContent('Surfshark')
    await userEvent.click(within(resumed).getByRole('button', { name: 'Save paused' }))
    expect(await within(resumed).findByText(/example.net via Surfshark.*saved/)).toBeInTheDocument()
    expect(bodyOf(fetchMock, 'POST /api/v1/policies')).toMatchObject({ via: 'Surfshark', match: { domains: ['example.net', '*.example.net'] } })
  })

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
    const sheet = await screen.findByRole('dialog', { name: 'Add routing rule' })
    expect(await within(sheet).findByText('First add the VPN profile it will go through.')).toBeInTheDocument()
    await userEvent.click(within(sheet).getByRole('button', { name: 'Add VPN profile' }))
    await userEvent.click(await within(sheet).findByRole('button', { name: 'Use Surfshark' }))
    await userEvent.click(within(sheet).getByRole('button', { name: 'Save' }))

    expect(await within(sheet).findByText('Done: “Surfshark” is added. Go on with your rule.')).toBeInTheDocument()
    expect(within(sheet).getByRole('textbox', { name: /Site, IP or program/ })).toHaveValue('claude.ai')
    expect(within(sheet).getByRole('button', { name: /Outgoing profile/ })).toHaveTextContent(/Surfshark/)
    await userEvent.click(within(sheet).getByRole('button', { name: 'Save paused' }))
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
    expect(within(sheet).getByRole('button', { name: /How it signs in/ })).toHaveTextContent('Username and password')
    await ready(sheet)
    await userEvent.type(within(sheet).getByRole('textbox', { name: 'Username' }), 'ana')
    await userEvent.type(within(sheet).getByLabelText('Password'), 's3cret')
    await userEvent.click(within(sheet).getByRole('button', { name: /When this profile’s connection is disconnected/ }))
    await userEvent.click(await screen.findByRole('option', { name: 'Block and try to connect the profile' }))
    await userEvent.click(within(sheet).getByRole('button', { name: 'Save' }))

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
      on_tunnel_down: 'connect',
      config: '/home/ana/.kongtrol/configs/Oficina.ovpn',
      auth_method: 'credentials',
      username: 'ana',
      password: 's3cret',
    })
    expect(within(sheet).getByText('/home/ana/.kongtrol/configs/Oficina.ovpn')).toBeInTheDocument()
    expect(within(sheet).queryByText(/from your Proton account/)).not.toBeInTheDocument()
  })

  it('a file from a Proton account says, once imported, not to connect it next to the Proton app (ADR-0103)', async () => {
    window.history.replaceState(null, '', '/?add=file')
    const { routes, state } = daemon([profile('work')], {
      'POST /api/v1/vpns/import': () =>
        json(
          {
            status: 'imported',
            path: '/home/ana/.kongtrol/configs/proton-ch.conf',
            type: 'wireguard',
            overwritten: false,
            provider: 'protonvpn',
            hint_key: 'profile.import.proton_hint',
            hint: 'This file is from your Proton account.',
          },
          201,
        ),
      'POST /api/v1/vpns': () => {
        state.profiles.push(profile('proton-ch', 'wireguard'))
        return json({ status: 'created', profile: 'proton-ch', restart_required: false }, 201)
      },
    })
    stubFetch(routes)
    render(<App />)
    const sheet = await screen.findByRole('dialog', { name: 'Import a VPN file' })
    await within(sheet).findByRole('button', { name: 'Choose file' })
    const input = sheet.querySelector<HTMLInputElement>('input[type="file"]')!
    await userEvent.upload(input, new File(['[Interface]\n# NetShield = 1\nAddress = 10.2.0.2/32\n'], 'proton-ch.conf'))
    await ready(sheet)
    await userEvent.click(within(sheet).getByRole('button', { name: 'Save' }))
    expect(
      await within(sheet).findByText('This file is from your Proton account. Do not connect it while the Proton app is connected: they clash.'),
    ).toBeInTheDocument()
  })
})

describe('B: files an .ovpn names (ADR-0105)', () => {
  const ovpn = 'client\nremote vpn.acme.test 1194\nca ca.crt\ntls-auth ta.key 1\n'
  const imported = (name: string) => json({ status: 'imported', path: `/home/ana/.kongtrol/configs/${name}.ovpn`, type: 'openvpn', overwritten: false }, 201)
  const zoneOf = (sheet: HTMLElement) => sheet.querySelector<HTMLElement>('.k-add-drop')!
  const drop = (el: HTMLElement, files: File[]) => fireEvent.drop(el, { dataTransfer: { files, types: ['Files'] } })

  async function openFileStep(extra: Routes) {
    window.history.replaceState(null, '', '/?add=file')
    const { routes, state } = daemon([profile('work')], {
      'POST /api/v1/vpns': () => {
        state.profiles.push(profile('Oficina', 'openvpn'))
        return json({ status: 'created', profile: 'Oficina', restart_required: false }, 201)
      },
      ...extra,
    })
    const fetchMock = stubFetch(routes)
    render(<App />)
    const sheet = await screen.findByRole('dialog', { name: 'Import a VPN file' })
    await within(sheet).findByRole('button', { name: 'Choose file' })
    return { sheet, fetchMock }
  }

  it('the .ovpn dropped together with its certificates sends them as files', async () => {
    const { sheet, fetchMock } = await openFileStep({ 'POST /api/v1/vpns/import': () => imported('Oficina') })
    drop(zoneOf(sheet), [new File(['CA'], 'ca.crt'), new File([ovpn], 'Oficina.ovpn'), new File(['TA'], 'ta.key')])
    expect(await within(sheet).findByText('Oficina.ovpn · OpenVPN')).toBeInTheDocument()
    expect(within(sheet).getByText('Together with: ca.crt, ta.key')).toBeInTheDocument()
    await ready(sheet)
    await userEvent.click(within(sheet).getByRole('button', { name: 'Save' }))
    expect(await within(sheet).findByText('“Oficina” is ready.')).toBeInTheDocument()
    const body = bodyOf(fetchMock, 'POST /api/v1/vpns/import') as { files: { name: string; content_base64: string }[] }
    expect(body.files).toEqual([
      { name: 'ca.crt', content_base64: btoa('CA') },
      { name: 'ta.key', content_base64: btoa('TA') },
    ])
  })

  it('missing files come back by name as soon as the file is chosen; "Add <name>" checks again with them (ADR-0118)', async () => {
    let calls = 0
    const { sheet, fetchMock } = await openFileStep({
      'POST /api/v1/vpns/import?check': (init) => {
        const body = JSON.parse(String(init?.body)) as { files?: { name: string }[] }
        if (!body.files?.some((f) => f.name === 'ta.key')) {
          return json(
            { error: 'files this .ovpn needs are missing: ta.key', hint: 'Add them with “Add”.', error_key: 'api.err.import_missing_files', missing: ['ta.key'] },
            400,
          )
        }
        return json(passedCheck(init, { included: ['ca.crt', 'ta.key'] }))
      },
      'POST /api/v1/vpns/import': () => {
        calls++
        return imported('Oficina')
      },
    })
    drop(zoneOf(sheet), [new File([ovpn], 'Oficina.ovpn'), new File(['CA'], 'ca.crt')])
    await within(sheet).findByText('Oficina.ovpn · OpenVPN')
    // No "Save" pressed: the check found what is missing, and saving waits for it.
    expect(await within(sheet).findByText('This .ovpn needs 1 more file')).toBeInTheDocument()
    expect(within(sheet).getByRole('button', { name: 'Save' })).toBeDisabled()
    expect(within(sheet).getByRole('button', { name: 'Save and activate' })).toBeDisabled()
    expect(within(sheet).getByText('Solve what is shown above to save.')).toBeInTheDocument()
    const addKey = within(sheet).getByRole('button', { name: 'Add ta.key' })
    const click = vi.spyOn(HTMLInputElement.prototype, 'click').mockImplementation(() => {})
    await userEvent.click(addKey)
    expect(click).toHaveBeenCalled()
    click.mockRestore()
    const extraInput = sheet.querySelectorAll<HTMLInputElement>('input[type="file"]')[1]!
    await userEvent.upload(extraInput, new File(['TA'], 'ta.key'))
    await ready(sheet)
    expect(within(sheet).getByText('They will go inside the profile: ca.crt, ta.key')).toBeInTheDocument()
    expect(calls).toBe(0)
    await userEvent.click(within(sheet).getByRole('button', { name: 'Save' }))
    expect(await within(sheet).findByText('“Oficina” is ready.')).toBeInTheDocument()
    expect(calls).toBe(1)
    const body = bodyOf(fetchMock, 'POST /api/v1/vpns/import') as { files: { name: string }[] }
    expect(body.files.map((f) => f.name)).toEqual(['ca.crt', 'ta.key'])
  })

  it('a file dropped while one is chosen replaces it and never reuses the old stored copy', async () => {
    const bodies: { content_base64?: string }[] = []
    let creates = 0
    const { sheet } = await openFileStep({
      'POST /api/v1/vpns/import': (init) => {
        bodies.push(JSON.parse(String(init?.body)) as { content_base64?: string })
        return imported('Oficina')
      },
      'POST /api/v1/vpns': () => {
        creates++
        return creates === 1
          ? json({ error: 'the daemon could not save it', hint: 'Try again.', error_key: 'api.err.save' }, 500)
          : json({ status: 'created', profile: 'Oficina', restart_required: false }, 201)
      },
    })
    const zone = zoneOf(sheet)
    drop(zone, [new File(['client\nremote a 1\n'], 'Oficina.ovpn')])
    await within(sheet).findByText('Oficina.ovpn · OpenVPN')
    await ready(sheet)
    await userEvent.click(within(sheet).getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(creates).toBe(1))

    // Dragging over the button inside the zone must not flicker it off.
    // The browser fires enter on the child, then leave on the zone.
    fireEvent.dragEnter(zone, { dataTransfer: { types: ['Files'] } })
    expect(zone).toHaveAttribute('data-drag')
    fireEvent.dragEnter(within(sheet).getByRole('button', { name: 'Choose another file' }), { dataTransfer: { types: ['Files'] } })
    fireEvent.dragLeave(zone, { dataTransfer: { types: ['Files'] } })
    expect(zone).toHaveAttribute('data-drag')

    drop(zone, [new File(['client\nremote b 1\n'], 'Oficina.ovpn')])
    expect(zone).not.toHaveAttribute('data-drag')
    await waitFor(() => expect(within(sheet).queryByText('the daemon could not save it')).not.toBeInTheDocument())
    await ready(sheet)
    await userEvent.click(within(sheet).getByRole('button', { name: 'Save' }))
    expect(await within(sheet).findByText('“Oficina” is ready.')).toBeInTheDocument()
    expect(bodies.map((b) => atob(b.content_base64 ?? ''))).toEqual(['client\nremote a 1\n', 'client\nremote b 1\n'])
  })

  it('"Choose file" opens the OS dialog through the daemon and imports by path; unavailable falls back to the browser', async () => {
    let picks = 0
    const { sheet, fetchMock } = await openFileStep({
      'GET /api/v1/vpns/import/pick': () => json({ available: true }),
      'POST /api/v1/vpns/import/pick': () => {
        picks++
        if (picks === 1) return json({ error: 'this computer cannot open the choose-a-file window', hint: 'Use “Choose file” again.', error_key: 'api.err.picker_unavailable' }, 503)
        return json({ path: 'C:\\VPN\\Oficina.ovpn', name: 'Oficina.ovpn', type: 'openvpn', content_base64: btoa(ovpn) })
      },
      'POST /api/v1/vpns/import': () => imported('Oficina'),
    })
    const chooser = within(sheet).getByRole('button', { name: 'Choose file' })
    // Wait for the availability answer, then the first try says the dialog cannot open here.
    await waitFor(() => expect(called(fetchMock, 'GET /api/v1/vpns/import/pick')).toBe(true))
    await userEvent.click(chooser)
    expect(await within(sheet).findByText('this computer cannot open the choose-a-file window')).toBeInTheDocument()
    const click = vi.spyOn(HTMLInputElement.prototype, 'click').mockImplementation(() => {})
    await userEvent.click(chooser)
    expect(click).toHaveBeenCalled()
    click.mockRestore()
    expect(picks).toBe(1)
  })

  it('a path from the OS dialog is imported by path (the daemon reads its folder)', async () => {
    const { sheet, fetchMock } = await openFileStep({
      'GET /api/v1/vpns/import/pick': () => json({ available: true }),
      'POST /api/v1/vpns/import/pick': () => json({ path: 'C:\\VPN\\Oficina.ovpn', name: 'Oficina.ovpn', type: 'openvpn', content_base64: btoa(ovpn) }),
      'POST /api/v1/vpns/import': () => json({ status: 'imported', path: '/k/Oficina.ovpn', type: 'openvpn', overwritten: false, included: ['ca.crt', 'ta.key'] }, 201),
    })
    await waitFor(() => expect(called(fetchMock, 'GET /api/v1/vpns/import/pick')).toBe(true))
    await userEvent.click(within(sheet).getByRole('button', { name: 'Choose file' }))
    expect(await within(sheet).findByText('Oficina.ovpn · OpenVPN')).toBeInTheDocument()
    await ready(sheet)
    await userEvent.click(within(sheet).getByRole('button', { name: 'Save' }))
    expect(await within(sheet).findByText('“Oficina” is ready.')).toBeInTheDocument()
    expect(bodyOf(fetchMock, 'POST /api/v1/vpns/import')).toEqual({ name: 'Oficina', type: 'openvpn', path: 'C:\\VPN\\Oficina.ovpn', overwrite: false })
  })
})

describe('drafts are protected', () => {
  it('Esc with something typed asks first; "Keep editing" keeps it, "Discard" closes', async () => {
    window.history.replaceState(null, '', '/?add=rule')
    stubFetch(daemon([profile('work')]).routes)
    render(<App />)
    const sheet = await screen.findByRole('dialog', { name: 'Add routing rule' })
    await userEvent.type(await within(sheet).findByRole('textbox', { name: /Site, IP or program/ }), 'claude.ai')
    esc(sheet)
    expect(await within(sheet).findByText('You have unsaved changes')).toBeInTheDocument()
    await userEvent.click(within(sheet).getByRole('button', { name: 'Keep editing' }))
    expect(within(sheet).getByRole('textbox', { name: /Site, IP or program/ })).toHaveValue('claude.ai')
    esc(sheet)
    await userEvent.click(await within(sheet).findByRole('button', { name: 'Discard' }))
    await waitFor(() => expect(screen.queryByRole('dialog', { name: 'Add routing rule' })).not.toBeInTheDocument())
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
    await userEvent.click(within(sheet).getByRole('button', { name: 'Save' }))
    expect(await within(sheet).findByText('“Surfshark” is ready.')).toBeInTheDocument()
    expect(called(fetchMock, 'POST /api/v1/restart')).toBe(false)
  })
})

describe('Traffic › Recent', () => {
  const recent = (body: unknown) => ({ 'GET /api/v1/connections/recent?limit=500': () => json(body) })

  it('lists recent destinations; "Add as a rule" (with a tooltip that names it) opens the rule sheet with that destination', async () => {
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
    const addRule = within(rows[1]!).getByRole('button', { name: 'Add as a rule' })
    expect(addRule).toHaveAccessibleDescription('Creates a rule so 10.2.0.5 goes through the VPN profile you choose.')
    await userEvent.click(addRule)
    const sheet = await screen.findByRole('dialog', { name: 'Add routing rule' })
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
    expect(screen.getByText("Only in this computer's memory: never saved to disk or sent off it; cleared when Kongtrol closes.")).toBeInTheDocument()
    await userEvent.click(sw)
    await waitFor(() => expect(bodyOf(fetchMock, 'PUT /api/v1/settings')).toEqual({ recent_connections_enabled: true }))
  })
})

describe('provider app added: move the rules of its account’s files (ADR-0104)', () => {
  const direct = (name: string, provider?: string): VPNProfile => ({
    ...profile(name, 'wireguard'),
    config: `/home/ana/.kongtrol/configs/${name}.conf`,
    ...(provider ? { provider } : {}),
  })
  const rule = (name: string, via: string) => ({ name, via, domains: [`${name}.example`], ip_ranges: null, apps: null, on_tunnel_down: 'allow' })

  function protonDaemon(start: VPNProfile[], policies: unknown[]) {
    const state = { policies }
    const d = daemon(start, {
      'GET /api/v1/vpns/detect': () => json([{ label: 'Proton VPN', adapter_type: 'protonvpn', path: 'C:/p.exe', active: false }]),
      'GET /api/v1/policies': () => json(state.policies),
      'POST /api/v1/vpns': () => {
        d.state.profiles.push(profile('Proton VPN', 'protonvpn'))
        return json({ status: 'created', profile: 'Proton VPN', restart_required: false }, 201)
      },
      'POST /api/v1/policies/move': () => {
        state.policies = state.policies.map((p) => ({ ...(p as object), via: 'Proton VPN' }))
        return json({ status: 'moved', from: 'proton-ch', to: 'Proton VPN', moved: ['news', 'shop'] })
      },
    })
    return d
  }

  async function addProtonApp() {
    window.history.replaceState(null, '', '/?add=vpn')
    const sheet = await screen.findByRole('dialog', { name: 'Connect a VPN I already use' })
    await userEvent.click(await within(sheet).findByRole('button', { name: 'Use Proton VPN' }))
    await userEvent.click(within(sheet).getByRole('button', { name: 'Save' }))
    expect(await within(sheet).findByText('“Proton VPN” is ready.')).toBeInTheDocument()
    return sheet
  }

  it('one card per file with rules (plural / singular); "Move rules" moves, "Keep them" only hides', async () => {
    const { routes } = protonDaemon(
      [direct('proton-ch', 'protonvpn'), direct('proton-us', 'protonvpn'), direct('proton-jp', 'protonvpn'), direct('home')],
      [rule('news', 'proton-ch'), rule('shop', 'proton-ch'), rule('mail', 'proton-us'), rule('bank', 'home')],
    )
    window.history.replaceState(null, '', '/?add=vpn')
    const fetchMock = stubFetch(routes)
    render(<App />)
    const sheet = await addProtonApp()

    expect(await within(sheet).findByText('Use Proton VPN for your proton-ch rules?')).toBeInTheDocument()
    expect(
      within(sheet).getByText(
        '2 rules use proton-ch, your Proton VPN file. If you move them to Proton VPN, those sites will go through the Proton VPN app. proton-ch stays saved in case you use it again.',
      ),
    ).toBeInTheDocument()
    expect(within(sheet).getByText('Use Proton VPN for your proton-us rules?')).toBeInTheDocument()
    expect(within(sheet).getByText(/^1 rule uses proton-us, your Proton VPN file\. If you move it to Proton VPN/)).toBeInTheDocument()
    expect(within(sheet).queryByText(/proton-jp rules/)).not.toBeInTheDocument()
    expect(within(sheet).queryByText(/your home rules/)).not.toBeInTheDocument()

    await userEvent.click(within(sheet).getByRole('button', { name: 'Keep them on proton-us' }))
    await waitFor(() => expect(within(sheet).queryByText('Use Proton VPN for your proton-us rules?')).not.toBeInTheDocument())
    expect(called(fetchMock, 'POST /api/v1/policies/move')).toBe(false)

    await userEvent.click(within(sheet).getByRole('button', { name: 'Move rules' }))
    expect(await screen.findByText('Done: 2 rules now use Proton VPN.')).toBeInTheDocument()
    expect(bodyOf(fetchMock, 'POST /api/v1/policies/move')).toEqual({ from: 'proton-ch', to: 'Proton VPN' })
    await waitFor(() => expect(within(sheet).queryByText('Use Proton VPN for your proton-ch rules?')).not.toBeInTheDocument())
  })

  it('no card when no file of that provider has rules', async () => {
    const { routes } = protonDaemon([direct('proton-ch', 'protonvpn'), direct('home')], [rule('bank', 'home')])
    window.history.replaceState(null, '', '/?add=vpn')
    stubFetch(routes)
    render(<App />)
    const sheet = await addProtonApp()
    await waitFor(() => expect(within(sheet).getByRole('button', { name: /Connect Proton VPN/ })).toBeInTheDocument())
    expect(within(sheet).queryByRole('button', { name: 'Move rules' })).not.toBeInTheDocument()
  })
})
