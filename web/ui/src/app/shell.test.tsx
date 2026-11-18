import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { App } from '../App'
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
    stubFetch({ ...base, 'GET /api/v1/helper/status': () => json(helperReady) })
    window.history.replaceState(null, '', '/profiles/groups')
    render(<App />)
    const tabs = await screen.findByRole('navigation', { name: 'Profiles' })
    expect(within(tabs).getByRole('link', { name: 'Groups' })).toHaveAttribute('aria-current', 'page')
    expect(screen.getByText('Connect, disconnect and reload a whole group')).toBeInTheDocument()

    await userEvent.click(within(tabs).getByRole('link', { name: 'VPN profiles' }))
    expect(await screen.findByText('Create, edit and delete VPN profiles')).toBeInTheDocument()

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
