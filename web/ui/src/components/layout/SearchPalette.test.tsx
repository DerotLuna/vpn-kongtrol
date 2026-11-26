import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { App } from '../../App'
import { helperReady, json, stubFetch } from '../../test/fakes'

beforeEach(() => {
  Object.defineProperty(navigator, 'languages', { value: ['en-US'], configurable: true })
  window.history.replaceState(null, '', '/')
  sessionStorage.clear()
  vi.stubGlobal('WebSocket', class { close() {} })
})

describe('global search', () => {
  it('opens with Ctrl+K, finds a rule by destination and navigates to it', async () => {
    stubFetch({
      'GET /api/v1/preferences': () => json({ language: 'en', theme: '' }),
      'GET /api/v1/helper/status': () => json(helperReady),
      'GET /api/v1/tunnels': () => json([]),
      'GET /api/v1/groups': () => json([]),
      'GET /api/v1/vpns': () => json([]),
      'GET /api/v1/policies': () => json([{ name: 'work-rule', via: 'work', domains: ['office.example'], ip_ranges: [], apps: [], on_tunnel_down: 'allow' }]),
      'GET /api/v1/policies/meta': () => json({ profiles: [], config_path: '' }),
      'GET /api/v1/events?after=-1': () => json({ events: [], seq: 0 }),
      'GET /api/v1/tunnels/routes': () => json({ tunnels: [] }),
    })
    render(<App />)
    const user = userEvent.setup()
    await user.keyboard('{Control>}k{/Control}')
    const dialog = await screen.findByRole('dialog', { name: 'Search Kongtrol' })
    const search = within(dialog).getByRole('searchbox')
    await user.type(search, 'office.example')
    const item = await within(dialog).findByRole('option', { name: /work-rule/ })
    await user.click(item)
    expect(await screen.findByRole('heading', { level: 1, name: 'Rules' })).toBeInTheDocument()
    const row = await screen.findByRole('row', { name: /work-rule/ })
    await waitFor(() => expect(row).toHaveFocus())
  })

  it('opens from the header and closes on Escape', async () => {
    stubFetch({
      'GET /api/v1/preferences': () => json({ language: 'en', theme: '' }),
      'GET /api/v1/helper/status': () => json(helperReady),
      'GET /api/v1/tunnels': () => json([]),
      'GET /api/v1/groups': () => json([]),
      'GET /api/v1/vpns': () => json([]),
      'GET /api/v1/policies': () => json([]),
      'GET /api/v1/events?after=-1': () => json({ events: [], seq: 0 }),
      'GET /api/v1/tunnels/routes': () => json({ tunnels: [] }),
    })
    render(<App />)
    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: 'Search (Ctrl or Cmd + K)' }))
    const dialog = await screen.findByRole('dialog', { name: 'Search Kongtrol' })
    await waitFor(() => expect(within(dialog).getByRole('searchbox')).toHaveFocus())
    await user.type(within(dialog).getByRole('searchbox'), 'profile')
    fireEvent.keyDown(within(dialog).getByRole('searchbox'), { key: 'Escape', isComposing: true })
    expect(screen.getByRole('dialog', { name: 'Search Kongtrol' })).toBeInTheDocument()
    await user.keyboard('{Escape}')
    expect(screen.queryByRole('dialog', { name: 'Search Kongtrol' })).not.toBeInTheDocument()
  })

  it('navigates to a section using the keyboard', async () => {
    stubFetch({
      'GET /api/v1/preferences': () => json({ language: 'en', theme: '' }),
      'GET /api/v1/helper/status': () => json(helperReady),
      'GET /api/v1/tunnels': () => json([]),
      'GET /api/v1/groups': () => json([]),
      'GET /api/v1/vpns': () => json([]),
      'GET /api/v1/policies': () => json([]),
      'GET /api/v1/events?after=-1': () => json({ events: [], seq: 0 }),
      'GET /api/v1/tunnels/routes': () => json({ tunnels: [] }),
    })
    render(<App />)
    const user = userEvent.setup()
    await user.keyboard('{Control>}k{/Control}')
    const dialog = await screen.findByRole('dialog', { name: 'Search Kongtrol' })
    const search = within(dialog).getByRole('searchbox')
    await user.type(search, 'Traffic')
    await within(dialog).findByRole('option', { name: /Traffic/ })
    await user.keyboard('{ArrowDown}{Enter}')
    expect(await screen.findByRole('heading', { level: 1, name: 'Traffic' })).toBeInTheDocument()
  })

  it('opens the add sheet from a search action', async () => {
    stubFetch({
      'GET /api/v1/preferences': () => json({ language: 'en', theme: '' }),
      'GET /api/v1/helper/status': () => json(helperReady),
      'GET /api/v1/tunnels': () => json([]),
      'GET /api/v1/groups': () => json([]),
      'GET /api/v1/vpns': () => json([]),
      'GET /api/v1/policies': () => json([]),
      'GET /api/v1/events?after=-1': () => json({ events: [], seq: 0 }),
      'GET /api/v1/tunnels/routes': () => json({ tunnels: [] }),
    })
    render(<App />)
    const user = userEvent.setup()
    await user.keyboard('{Control>}k{/Control}')
    const dialog = await screen.findByRole('dialog', { name: 'Search Kongtrol' })
    await user.type(within(dialog).getByRole('searchbox'), 'Import VPN configuration file')
    await user.click(await within(dialog).findByRole('option', { name: /Import VPN configuration file/ }))
    await waitFor(() => expect(window.location.search).toContain('add=file'))
    expect(screen.queryByRole('dialog', { name: 'Search Kongtrol' })).not.toBeInTheDocument()
    expect(await screen.findByRole('dialog')).toBeInTheDocument()
  })
})
