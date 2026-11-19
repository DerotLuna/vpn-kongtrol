import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { App } from '../App'
import type { VPNProfile } from '../data/profiles'
import { FakeSocket, helperReady, json, stubFetch, tunnel, type Routes } from '../test/fakes'

function profile(name: string): VPNProfile {
  return {
    name,
    type: 'wireguard',
    config: `${name}.conf`,
    priority: 0,
    failover: [],
    has_username_credential: false,
    has_password_credential: false,
    kill_switch_override: 'inherit',
    auto_connect: true,
  }
}

function base(extra: Routes = {}): Routes {
  return {
    'GET /api/v1/preferences': () => json({ language: 'en', theme: '' }),
    'GET /api/v1/version': () => json({ version: 'v0.7.0' }),
    'GET /api/v1/helper/status': () => json(helperReady),
    'GET /api/v1/tunnels': () => json([tunnel('a', 'connected'), tunnel('b', 'disconnected')]),
    'GET /api/v1/setup': () => json({ setup_mode: false, needs_setup: false, restart_pending: false, config_path: '' }),
    'GET /api/v1/vpns': () => json([profile('a'), profile('b'), profile('c')]),
    ...extra,
  }
}

beforeEach(() => {
  Object.defineProperty(navigator, 'languages', { value: ['en-US'], configurable: true })
  FakeSocket.reset()
  vi.stubGlobal('WebSocket', FakeSocket)
  window.history.replaceState(null, '', '/profiles/groups')
})

describe('GroupsPage', () => {
  it('creates a group from existing profiles only and shows it at once', async () => {
    let groups = [{ name: 'office', profiles: ['a', 'b'] }]
    const fetchMock = stubFetch(
      base({
        'GET /api/v1/groups': () => json(groups),
        'POST /api/v1/groups': (init) => {
          const body = JSON.parse(String(init?.body)) as { name: string; profiles: string[] }
          groups = [...groups, body]
          return json({ status: 'created', group: body.name }, 201)
        },
      }),
    )
    render(<App />)
    const table = await screen.findByRole('table', { name: 'VPN groups' })
    expect(within(table).getByText('1 of 2 connected')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'New group' }))
    const dialog = await screen.findByRole('dialog', { name: 'New group' })
    await userEvent.click(within(dialog).getByRole('button', { name: 'Create group' }))
    expect(within(dialog).getByText('The group name is missing.')).toBeInTheDocument()
    expect(within(dialog).getByText('Tick at least one VPN.')).toBeInTheDocument()

    await userEvent.type(within(dialog).getByLabelText(/^Group name/), 'home')
    await userEvent.click(within(dialog).getByRole('checkbox', { name: /c/ }))
    await userEvent.click(within(dialog).getByRole('checkbox', { name: /^a/ }))
    await userEvent.click(within(dialog).getByRole('button', { name: 'Create group' }))

    expect(await within(table).findByText('home')).toBeInTheDocument()
    const post = fetchMock.mock.calls.find(([u, i]) => String(u) === '/api/v1/groups' && i?.method === 'POST')
    // Order follows the profile list, not the click order.
    expect(JSON.parse(String(post?.[1]?.body))).toEqual({ name: 'home', profiles: ['a', 'c'] })
    expect(await screen.findByText('Group “home” created.')).toBeInTheDocument()
  })

  it('names the profiles a group connect could not start and offers the restart', async () => {
    stubFetch(
      base({
        'GET /api/v1/groups': () => json([{ name: 'office', profiles: ['a', 'b', 'c'] }]),
        'POST /api/v1/groups/office/connect': () =>
          json({ status: 'connecting', group: 'office', started: ['b'], already_connected: ['a'] }, 202),
      }),
    )
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: 'Connect office' }))
    expect(await screen.findByText('“office”: could not start c.')).toBeInTheDocument()
    // The member it started stays in progress until the feed shows it up: the group says so.
    expect(screen.getByRole('button', { name: 'Connecting… office' })).toHaveAttribute('aria-busy', 'true')
    expect(screen.getByRole('button', { name: 'Restart now' })).toBeInTheDocument()
  })

  it('reload: a 409 restart_required is explained, not shown as a failure', async () => {
    stubFetch(
      base({
        'GET /api/v1/groups': () => json([{ name: 'office', profiles: ['a'] }]),
        'POST /api/v1/groups/office/reload': () =>
          json({ status: 'restart_required', group: 'office', missing_profiles: ['a'], message: 'english' }, 409),
      }),
    )
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: 'Reload office' }))
    expect(await screen.findByText('Restart Kongtrol to reload “office”')).toBeInTheDocument()
    expect(screen.getByText('a are not active in Kongtrol yet.')).toBeInTheDocument()
  })

  it('deletes after an app dialog, and rolls back when refused', async () => {
    stubFetch(
      base({
        'GET /api/v1/groups': () => json([{ name: 'office', profiles: ['a'] }]),
        'DELETE /api/v1/groups/office': () =>
          json({ error: 'group "office" does not exist', hint: 'Reload the page: it may have been deleted.', error_key: 'api.err.group_not_found' }, 404),
      }),
    )
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: 'Delete office' }))
    const confirm = await screen.findByRole('alertdialog', { name: 'Delete the group “office”?' })
    await userEvent.click(within(confirm).getByRole('button', { name: 'Delete' }))
    expect(await screen.findByText('group "office" does not exist')).toBeInTheDocument()
    await waitFor(() => expect(screen.getByRole('button', { name: 'Delete office' })).toBeInTheDocument())
  })

  it('without profiles, the empty state sends you to create one first', async () => {
    stubFetch(base({ 'GET /api/v1/groups': () => json([]), 'GET /api/v1/vpns': () => json([]) }))
    render(<App />)
    expect(await screen.findByRole('heading', { name: 'Create a VPN first' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Go to VPN profiles' })).toHaveAttribute('href', '/profiles')
  })
})
