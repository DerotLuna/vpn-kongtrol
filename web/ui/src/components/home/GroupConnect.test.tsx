import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { PreferencesProvider } from '../../app/PreferencesProvider'
import { json, stubFetch } from '../../test/fakes'
import { GroupConnect } from './GroupConnect'

beforeEach(() => {
  Object.defineProperty(navigator, 'languages', { value: ['en-US'], configurable: true })
  stubFetch({ 'GET /api/v1/preferences': () => json({ language: 'en', theme: '' }) })
})

const draw = (up: number, busy?: 'connect' | 'disconnect', onRun = vi.fn()) =>
  render(
    <PreferencesProvider>
      <GroupConnect name="office" total={3} up={up} busy={busy} onRun={onRun} />
    </PreferencesProvider>,
  )

describe('GroupConnect squares', () => {
  it('none up: only Conectar (filled); all up: only Desconectar (ink outline)', () => {
    const { unmount } = draw(0)
    expect(screen.getAllByRole('button')).toHaveLength(1)
    expect(screen.getByRole('button', { name: 'Connect office' })).toHaveClass('k-icon-btn--sm', 'k-icon-btn--filled')
    unmount()
    draw(3)
    expect(screen.getAllByRole('button')).toHaveLength(1)
    expect(screen.getByRole('button', { name: 'Disconnect office' })).toHaveClass('k-icon-btn--sm', 'k-icon-btn--strong')
  })

  it('some up: both, each saying why (the rest / all), and each runs its own action', async () => {
    const onRun = vi.fn()
    draw(1, undefined, onRun)
    const rest = screen.getByRole('button', { name: 'Connect the rest of office (2 of 3)' })
    const all = screen.getByRole('button', { name: 'Disconnect all of office (1 of 3 connected)' })
    expect(rest).toHaveClass('k-icon-btn--filled')
    expect(all).toHaveClass('k-icon-btn--outlined')
    await userEvent.click(rest)
    await userEvent.click(all)
    expect(onRun.mock.calls).toEqual([['connect'], ['disconnect']])
  })

  it('in progress: a spinner inside the same square, the other one inert', async () => {
    const onRun = vi.fn()
    draw(1, 'connect', onRun)
    const busy = screen.getByRole('button', { name: 'Connecting office…' })
    expect(busy).toHaveAttribute('aria-busy', 'true')
    expect(busy.querySelector('.k-spinner')).not.toBeNull()
    const other = screen.getByRole('button', { name: /Disconnect all of office/ })
    expect(other).toHaveAttribute('aria-disabled', 'true')
    await userEvent.click(other)
    expect(onRun).not.toHaveBeenCalled()
  })
})
