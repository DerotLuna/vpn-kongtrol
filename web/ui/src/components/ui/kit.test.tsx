import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { PreferencesProvider } from '../../app/PreferencesProvider'
import { toasts } from '../../data/toast'
import { json, stubFetch } from '../../test/fakes'
import { Button, IconButton } from './Button'
import { DataList } from './DataList'
import { Dialog } from './Dialog'
import { TextField } from './Field'
import { CloseIcon } from './icons'
import { Banner } from './States'
import { Switch } from './Switch'
import { Tabs } from './Tabs'
import { Toaster } from './Toaster'

function withPrefs(ui: React.ReactNode) {
  return render(<PreferencesProvider>{ui}</PreferencesProvider>)
}

beforeEach(() => {
  Object.defineProperty(navigator, 'languages', { value: ['en-US'], configurable: true })
  stubFetch({ 'GET /api/v1/preferences': () => json({ language: 'en', theme: '' }) })
})

describe('Button', () => {
  it('loading keeps the name, disables and sets aria-busy', () => {
    render(<Button loading>Save</Button>)
    const b = screen.getByRole('button', { name: 'Save' })
    expect(b).toBeDisabled()
    expect(b).toHaveAttribute('aria-busy', 'true')
  })
})

describe('IconButton + Tooltip', () => {
  it('is named by its label and the tooltip shows on focus and hides with Esc', async () => {
    withPrefs(<IconButton label="Close panel" icon={<CloseIcon />} />)
    const b = screen.getByRole('button', { name: 'Close panel' })
    const tip = screen.getByRole('tooltip', { hidden: true })
    await userEvent.tab()
    expect(b).toHaveFocus()
    expect(tip).toHaveAttribute('data-open')
    await userEvent.keyboard('{Escape}')
    expect(tip).not.toHaveAttribute('data-open')
    expect(b).toHaveFocus()
  })
})

describe('Switch', () => {
  function Demo() {
    const [on, setOn] = useState(false)
    return <Switch checked={on} onChange={setOn} label="Kill switch" description="Blocks traffic" onText="On" offText="Off" />
  }

  it('is a switch named by its label, toggles with Space, and says its state in words', async () => {
    render(<Demo />)
    const sw = screen.getByRole('switch', { name: 'Kill switch' })
    expect(sw).toHaveAttribute('aria-checked', 'false')
    expect(sw).toHaveAccessibleDescription('Blocks traffic')
    expect(sw).toHaveTextContent('Off')
    sw.focus()
    await userEvent.keyboard(' ')
    expect(sw).toHaveAttribute('aria-checked', 'true')
    expect(sw).toHaveTextContent('On')
  })
})

describe('Tabs', () => {
  function Demo() {
    const [v, setV] = useState('a')
    return (
      <Tabs
        label="Views"
        value={v}
        onChange={setV}
        items={[
          { id: 'a', label: 'One', panel: <p>Panel one</p> },
          { id: 'b', label: 'Two', panel: <p>Panel two</p> },
          { id: 'c', label: 'Three', panel: <p>Panel three</p> },
        ]}
      />
    )
  }

  it('follows the APG pattern: arrows, Home/End, roving tabindex', async () => {
    render(<Demo />)
    const [one, two, three] = screen.getAllByRole('tab')
    expect(one).toHaveAttribute('aria-selected', 'true')
    expect(two).toHaveAttribute('tabindex', '-1')
    one!.focus()
    await userEvent.keyboard('{ArrowRight}')
    expect(two).toHaveFocus()
    expect(screen.getByRole('tabpanel', { name: 'Two' })).toHaveTextContent('Panel two')
    await userEvent.keyboard('{End}')
    expect(three).toHaveAttribute('aria-selected', 'true')
    await userEvent.keyboard('{ArrowRight}')
    expect(one).toHaveFocus()
  })
})

describe('Dialog', () => {
  function Demo() {
    const [open, setOpen] = useState(false)
    return (
      <>
        <button type="button" onClick={() => setOpen(true)}>
          Delete
        </button>
        <Dialog open={open} onClose={() => setOpen(false)} title="Delete profile?" description="Cannot be undone." alert />
      </>
    )
  }

  it('opens as a labelled alertdialog and closes with Esc', async () => {
    withPrefs(<Demo />)
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))
    const dialog = screen.getByRole('alertdialog', { name: 'Delete profile?' })
    expect(dialog).toHaveAccessibleDescription('Cannot be undone.')
    expect(dialog).toHaveAttribute('open')
    act(() => {
      dialog.dispatchEvent(new Event('cancel', { cancelable: true }))
    })
    expect(dialog).not.toHaveAttribute('open')
  })
})

describe('TextField', () => {
  it('links the error (what happened + what to do) and marks the input invalid', () => {
    render(<TextField label="Destination" error={{ message: 'Not a network.', fix: 'Type 10.0.0.0/8.' }} hint="IP or CIDR" />)
    const input = screen.getByRole('textbox', { name: 'Destination' })
    expect(input).toHaveAttribute('aria-invalid', 'true')
    expect(input).toHaveAccessibleDescription('Not a network. Type 10.0.0.0/8. IP or CIDR')
  })
})

describe('Toaster', () => {
  it('shows one toast at a time in a live region; errors stay until dismissed', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    withPrefs(<Toaster />)
    act(() => {
      toasts.show({ tone: 'ok', message: 'Saved' })
      toasts.show({ tone: 'bad', message: 'Could not save', detail: 'What to do: retry.' })
    })
    expect(screen.getByRole('status')).toHaveTextContent('Saved')
    expect(screen.queryByText('Could not save')).toBeNull()
    act(() => {
      vi.advanceTimersByTime(4_100)
    })
    expect(await screen.findByText('Could not save')).toBeInTheDocument()
    act(() => {
      vi.advanceTimersByTime(60_000)
    })
    expect(screen.getByText('Could not save')).toBeInTheDocument()
    vi.useRealTimers()
    await userEvent.click(screen.getByRole('button', { name: 'Dismiss notice' }))
    expect(screen.queryByText('Could not save')).toBeNull()
  })
})

describe('Banner', () => {
  it('is a status by default and an alert when urgent', () => {
    const { rerender } = render(<Banner tone="caution" title="Helper missing" />)
    expect(screen.getByRole('status')).toHaveTextContent('Helper missing')
    rerender(<Banner tone="bad" title="Kongtrol stopped" urgent />)
    expect(screen.getByRole('alert')).toHaveTextContent('Kongtrol stopped')
  })
})

describe('DataList', () => {
  const columns = [{ id: 'name', header: 'Name', cell: (r: { id: string }) => r.id }]
  it('marks only rows added after the first render as new (enter animation)', () => {
    const { rerender, container } = render(<DataList caption="x" columns={columns} rows={[{ id: 'a' }]} rowKey={(r) => r.id} />)
    expect(container.querySelector('tr[data-new]')).toBeNull()
    rerender(<DataList caption="x" columns={columns} rows={[{ id: 'a' }, { id: 'b' }]} rowKey={(r) => r.id} />)
    const marked = Array.from(container.querySelectorAll('tbody tr[data-new]')).map((tr) => tr.getAttribute('data-key'))
    expect(marked).toEqual(['b'])
  })
})
