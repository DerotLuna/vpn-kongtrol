import { act, render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { App } from '../App'
import { NAV_KEY, NAV_WIDE_QUERY } from './navCollapse'
import { FakeSocket, helperMissing, helperReady, json, stubFetch, tunnel } from '../test/fakes'

beforeEach(() => {
  Object.defineProperty(navigator, 'languages', { value: ['en-US'], configurable: true })
  FakeSocket.reset()
  vi.stubGlobal('WebSocket', FakeSocket)
})

const base = {
  'GET /api/v1/preferences': () => json({ language: 'en', theme: '' }),
  'GET /api/v1/version': () => json({ version: 'v0.7.0' }),
  'GET /api/v1/tunnels': () => json([tunnel('work', 'connected'), tunnel('home', 'disconnected')]),
}

describe('shell and routes', () => {
  it('has a skip link, one nav, the aggregate status and the daemon version', async () => {
    stubFetch({ ...base, 'GET /api/v1/helper/status': () => json(helperReady) })
    render(<App />)
    expect(screen.getByRole('link', { name: 'Skip to content' })).toHaveAttribute('href', '#main')
    expect(await screen.findByRole('link', { name: 'Your VPNs status: 1 of 2 connected' })).toBeInTheDocument()
    expect((await screen.findAllByText('Version v0.7.0'))[0]).toBeInTheDocument()
    const nav = screen.getAllByRole('navigation', { name: 'Sections' })[0]!
    expect(within(nav).getByRole('link', { name: 'Home' })).toHaveAttribute('aria-current', 'page')
  })

  it('serves every section at the site root, with no link to the removed classic dashboard', async () => {
    stubFetch({ ...base, 'GET /api/v1/helper/status': () => json(helperReady) })
    render(<App />)
    const nav = (await screen.findAllByRole('navigation', { name: 'Sections' }))[0]!
    const hrefs = within(nav)
      .getAllByRole('link')
      .map((a) => a.getAttribute('href'))
    expect(hrefs).toEqual(['/', '/profiles', '/rules', '/traffic', '/security', '/activity', '/settings'])
    expect(screen.queryByRole('link', { name: /classic/i })).not.toBeInTheDocument()
  })

  it('the status link keeps its visible words in its name while the live feed reconnects', async () => {
    stubFetch({ ...base, 'GET /api/v1/helper/status': () => json(helperReady) })
    render(<App />)
    await screen.findByRole('link', { name: 'Your VPNs status: 1 of 2 connected' })
    const socket = FakeSocket.last()
    act(() => {
      socket.open()
      socket.drop()
    })
    const link = await screen.findByRole('link', { name: /Reconnecting to Kongtrol…$/ })
    // WCAG 2.5.3: the accessible name contains the visible text.
    expect(link).toHaveAccessibleName('Your VPNs status: 1 of 2 connected Reconnecting to Kongtrol…')
  })

  it('navigates without reload, focuses the new heading and sets the title', async () => {
    stubFetch({ ...base, 'GET /api/v1/helper/status': () => json(helperReady) })
    render(<App />)
    const nav = screen.getAllByRole('navigation', { name: 'Sections' })[0]!
    await userEvent.click(within(nav).getByRole('link', { name: 'Security' }))
    const h1 = await screen.findByRole('heading', { level: 1, name: 'Security' })
    expect(h1).toHaveFocus()
    expect(document.title).toBe('Security · Kongtrol')
    expect(within(nav).getByRole('link', { name: 'Security' })).toHaveAttribute('aria-current', 'page')
  })

  it('routes tabs as sub-pages and unknown paths to a guided 404', async () => {
    stubFetch({
      ...base,
      'GET /api/v1/helper/status': () => json(helperReady),
      'GET /api/v1/groups': () => json([]),
      'GET /api/v1/vpns': () => json([]),
    })
    window.history.replaceState(null, '', '/profiles/groups')
    render(<App />)
    const tabs = await screen.findByRole('navigation', { name: 'Profiles' })
    expect(within(tabs).getByRole('link', { name: 'Groups' })).toHaveAttribute('aria-current', 'page')
    expect(await screen.findByRole('heading', { name: 'Your groups' })).toBeInTheDocument()

    await userEvent.click(within(tabs).getByRole('link', { name: 'VPN profiles' }))
    expect(await screen.findByRole('heading', { name: 'Create your first VPN' })).toBeInTheDocument()

    window.history.pushState(null, '', '/nope')
    window.dispatchEvent(new PopStateEvent('popstate'))
    expect(await screen.findByRole('heading', { level: 1, name: 'We could not find this page' })).toBeInTheDocument()
  })

  it('shows the helper notice once on other pages and hides it everywhere after installing', async () => {
    let status = helperMissing
    stubFetch({
      ...base,
      'GET /api/v1/helper/status': () => json(status),
      'POST /api/v1/helper/install': () => {
        status = helperReady
        return json(helperReady)
      },
    })
    window.history.replaceState(null, '', '/rules')
    render(<App />)
    expect(await screen.findByText('The system helper is missing')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Install helper' }))
    expect(await screen.findByText('Helper ready · no need to install it again')).toBeInTheDocument()
    expect(screen.queryByText('The system helper is missing')).toBeNull()
    expect(screen.queryByRole('button', { name: 'Install helper' })).toBeNull()
  })

  it('a failed install leaves the button and says what to do', async () => {
    stubFetch({
      ...base,
      'GET /api/v1/helper/status': () => json(helperMissing),
      'POST /api/v1/helper/install': () =>
        json(
          {
            error: 'You declined the permission prompt.',
            hint: 'What to do: press Install helper again and accept it.',
            error_key: 'service.elevation.cancelled',
          },
          403,
        ),
    })
    window.history.replaceState(null, '', '/rules')
    render(<App />)
    await userEvent.click(await screen.findByRole('button', { name: 'Install helper' }))
    // Inline next to the button and as a toast: the toast is never the only copy.
    expect(await screen.findAllByText('You declined the permission prompt.')).toHaveLength(2)
    expect(screen.getAllByText('What to do: press Install helper again and accept it.')).toHaveLength(2)
    expect(screen.getByRole('button', { name: 'Install helper' })).toBeEnabled()
  })

  it('toasts daemon notices on every page, not only on Home', async () => {
    sessionStorage.clear()
    stubFetch({
      ...base,
      'GET /api/v1/helper/status': () => json(helperReady),
      'GET /api/v1/events?after=-1': () =>
        json({
          seq: 3,
          events: [
            { seq: 3, time: '2026-11-18T10:00:00Z', kind: 'tunnel_watchdog', level: 'INFO', key: 'watchdog.event.reconnected', args: ['work'], profile: 'work' },
          ],
        }),
    })
    window.history.replaceState(null, '', '/rules')
    render(<App />)
    expect(await screen.findByRole('heading', { level: 1, name: 'Rules' })).toBeInTheDocument()
    const region = document.querySelector<HTMLElement>('.k-toaster [role="status"]')!
    expect(await within(region).findByText('work reconnected')).toBeInTheDocument()
  })

  it('says Kongtrol stopped answering, with what to do, over every page', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.reject(new TypeError('Failed to fetch'))))
    window.history.replaceState(null, '', '/traffic')
    render(<App />)
    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('Could not reach Kongtrol.')
    expect(alert).toHaveTextContent('This page reconnects by itself as soon as Kongtrol starts.')
    expect(within(alert).getByRole('button', { name: 'Retry' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Your VPNs status: Kongtrol is not answering/ })).toBeInTheDocument()
  })
})

describe('header and side menu', () => {
  const wide = (matches: boolean) =>
    vi.stubGlobal('matchMedia', (query: string) => ({
      matches: query === NAV_WIDE_QUERY ? matches : false,
      media: query,
      onchange: null,
      addEventListener: () => {},
      removeEventListener: () => {},
      addListener: () => {},
      removeListener: () => {},
      dispatchEvent: () => false,
    }))

  it('the header has no language or theme switch (they live in Settings)', async () => {
    stubFetch({ ...base, 'GET /api/v1/helper/status': () => json(helperReady) })
    render(<App />)
    const toolbar = await screen.findByRole('group', { name: 'Add and help' })
    expect(within(toolbar).queryByRole('button', { name: /language|Spanish|ES|EN/ })).toBeNull()
    expect(within(toolbar).queryByRole('button', { name: /theme/i })).toBeNull()
  })

  it('wide window: "Collapse menu" turns the sidebar into an icon rail, keeps focus, and is remembered', async () => {
    wide(true)
    stubFetch({ ...base, 'GET /api/v1/helper/status': () => json(helperReady) })
    const user = userEvent.setup()
    const { unmount } = render(<App />)
    const app = document.querySelector('.k-app')!
    expect(app).toHaveAttribute('data-nav', 'expanded')
    const toggle = screen.getByRole('button', { name: 'Collapse menu' })
    expect(toggle).toHaveAttribute('aria-expanded', 'true')
    expect(toggle).toHaveAttribute('aria-controls', document.querySelector('.k-sidenav ul')!.id)
    await user.click(toggle)
    expect(app).toHaveAttribute('data-nav', 'collapsed')
    const expand = screen.getByRole('button', { name: 'Expand menu' })
    expect(expand).toBe(toggle)
    expect(expand).toHaveFocus()
    expect(expand).toHaveAttribute('aria-expanded', 'false')
    // Links keep their names (the label is only visually hidden).
    const nav = screen.getAllByRole('navigation', { name: 'Sections' })[0]!
    expect(within(nav).getByRole('link', { name: 'Security' })).toBeInTheDocument()
    expect(localStorage.getItem(NAV_KEY)).toBe('collapsed')
    unmount()
    render(<App />)
    expect(document.querySelector('.k-app')).toHaveAttribute('data-nav', 'collapsed')
  })

  it('collapsed: keyboard focus on an icon shows its name as a tooltip', async () => {
    wide(true)
    localStorage.setItem(NAV_KEY, 'collapsed')
    stubFetch({ ...base, 'GET /api/v1/helper/status': () => json(helperReady) })
    render(<App />)
    const nav = screen.getAllByRole('navigation', { name: 'Sections' })[0]!
    const link = within(nav).getByRole('link', { name: 'Rules' })
    act(() => link.focus())
    await userEvent.keyboard('{Shift>}{Tab}{/Shift}{Tab}')
    expect(link).toHaveFocus()
    expect(await screen.findByRole('tooltip')).toHaveTextContent('Rules')
    // The name is not announced twice.
    expect(link).toHaveAccessibleDescription('')
  })

  it('narrow window (< 1100px): starts collapsed; expanding lasts this visit only', async () => {
    wide(false)
    stubFetch({ ...base, 'GET /api/v1/helper/status': () => json(helperReady) })
    const user = userEvent.setup()
    render(<App />)
    const app = document.querySelector('.k-app')!
    expect(app).toHaveAttribute('data-nav', 'collapsed')
    await user.click(screen.getByRole('button', { name: 'Expand menu' }))
    expect(app).toHaveAttribute('data-nav', 'expanded')
    expect(localStorage.getItem(NAV_KEY)).toBeNull()
  })

  it('storage blocked: the menu still toggles', async () => {
    wide(true)
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('blocked')
    })
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('blocked')
    })
    stubFetch({ ...base, 'GET /api/v1/helper/status': () => json(helperReady) })
    render(<App />)
    await userEvent.click(screen.getByRole('button', { name: 'Collapse menu' }))
    expect(document.querySelector('.k-app')).toHaveAttribute('data-nav', 'collapsed')
  })
})
