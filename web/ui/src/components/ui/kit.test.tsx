import { act, fireEvent, render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { PreferencesProvider } from '../../app/PreferencesProvider'
import { toasts } from '../../data/toast'
import { json, pickOption, stubFetch } from '../../test/fakes'
import { addDays, addMonths, matchPreset, monthGrid, presetRange, weekStart, type DateRange } from './dates'
import { DateRangePicker } from './DateRangePicker'
import { Menu } from './Menu'
import { TOOLTIP_DELAY_MS } from './Tooltip'
import { Button, IconButton } from './Button'
import { DataList } from './DataList'
import { Dialog } from './Dialog'
import { ComboboxField, SelectField, TextField } from './Field'
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
  const bubble = () => document.querySelector('.k-tooltip')

  it('is named by its label; the tooltip shows on keyboard focus and hides with Esc', async () => {
    withPrefs(<IconButton label="Close panel" icon={<CloseIcon />} />)
    const b = screen.getByRole('button', { name: 'Close panel' })
    expect(bubble()).toBeNull()
    await userEvent.tab()
    expect(b).toHaveFocus()
    expect(bubble()).toHaveTextContent('Close panel')
    // Visual only: screen readers already have the name.
    expect(bubble()).toHaveAttribute('aria-hidden', 'true')
    await userEvent.keyboard('{Escape}')
    expect(bubble()).toBeNull()
    expect(b).toHaveFocus()
  })

  it('a longer tip describes the button (aria-describedby on a node that is always there)', () => {
    withPrefs(<IconButton label="Reconnect office" tip="Reconnect: re-reads its configuration" icon={<CloseIcon />} />)
    const b = screen.getByRole('button', { name: 'Reconnect office' })
    expect(b).toHaveAccessibleDescription('Reconnect: re-reads its configuration')
  })

  it('opens on hover only after the delay', async () => {
    vi.useFakeTimers()
    try {
      withPrefs(<IconButton label="Refresh" icon={<CloseIcon />} />)
      const anchor = screen.getByRole('button', { name: 'Refresh' }).parentElement!
      fireEvent.pointerEnter(anchor, { pointerType: 'mouse' })
      act(() => vi.advanceTimersByTime(TOOLTIP_DELAY_MS - 50))
      expect(bubble()).toBeNull()
      act(() => vi.advanceTimersByTime(60))
      expect(bubble()).toHaveTextContent('Refresh')
      fireEvent.pointerLeave(anchor, { pointerType: 'mouse' })
      act(() => vi.advanceTimersByTime(200))
      expect(bubble()).toBeNull()
    } finally {
      vi.useRealTimers()
    }
  })

  it('touch: a long press shows it and does not press the button', () => {
    vi.useFakeTimers()
    try {
      const onClick = vi.fn()
      withPrefs(<IconButton label="Delete" icon={<CloseIcon />} onClick={onClick} />)
      const b = screen.getByRole('button', { name: 'Delete' })
      fireEvent.pointerDown(b, { pointerType: 'touch' })
      act(() => vi.advanceTimersByTime(600))
      expect(bubble()).toHaveTextContent('Delete')
      fireEvent.pointerUp(b, { pointerType: 'touch' })
      fireEvent.click(b)
      expect(onClick).not.toHaveBeenCalled()
      // A normal tap still presses it.
      fireEvent.pointerDown(b, { pointerType: 'touch' })
      fireEvent.pointerUp(b, { pointerType: 'touch' })
      fireEvent.click(b)
      expect(onClick).toHaveBeenCalledTimes(1)
    } finally {
      vi.useRealTimers()
    }
  })
})

describe('Menu (⋯)', () => {
  it('APG menu button: opens on the first item, arrows wrap, Esc returns focus, choosing runs the item', async () => {
    const user = userEvent.setup()
    const reload = vi.fn()
    const add = vi.fn()
    withPrefs(
      <Menu
        label="More actions for office"
        items={[
          { id: 'r', label: 'Reconnect', onSelect: reload },
          { id: 'a', label: 'Add a site', onSelect: add },
        ]}
      />,
    )
    const button = screen.getByRole('button', { name: 'More actions for office' })
    expect(button).toHaveAttribute('aria-haspopup', 'menu')
    await user.click(button)
    expect(button).toHaveAttribute('aria-expanded', 'true')
    const items = within(screen.getByRole('menu')).getAllByRole('menuitem')
    expect(items[0]).toHaveFocus()
    await user.keyboard('{ArrowUp}')
    expect(items[1]).toHaveFocus()
    await user.keyboard('{ArrowDown}')
    expect(items[0]).toHaveFocus()
    await user.keyboard('a')
    expect(items[1]).toHaveFocus()
    await user.keyboard('{Escape}')
    expect(screen.queryByRole('menu')).toBeNull()
    expect(button).toHaveFocus()
    await user.keyboard('{ArrowUp}')
    expect(within(screen.getByRole('menu')).getAllByRole('menuitem')[1]).toHaveFocus()
    await user.keyboard('{Enter}')
    expect(add).toHaveBeenCalledTimes(1)
    expect(reload).not.toHaveBeenCalled()
    expect(screen.queryByRole('menu')).toBeNull()
  })
})

describe('Select', () => {
  function Demo() {
    const [v, setV] = useState('INFO')
    return (
      <SelectField
        label="Level"
        value={v}
        onChange={setV}
        options={[
          { value: '', label: 'All' },
          { value: 'INFO', label: 'Info' },
          { value: 'WARN', label: 'Warning' },
          { value: 'ERROR', label: 'Error' },
        ]}
      />
    )
  }

  it('select-only combobox: named by its label, keyboard, type-ahead, Esc keeps the value', async () => {
    const user = userEvent.setup()
    withPrefs(<Demo />)
    const box = screen.getByRole('combobox', { name: 'Level' })
    expect(box).toHaveTextContent('Info')
    box.focus()
    await user.keyboard('{ArrowDown}')
    const list = screen.getByRole('listbox')
    expect(box).toHaveAttribute('aria-expanded', 'true')
    // Opens on the chosen option.
    expect(box.getAttribute('aria-activedescendant')).toBe(within(list).getByRole('option', { name: 'Info' }).id)
    await user.keyboard('{ArrowDown}{Enter}')
    expect(box).toHaveTextContent('Warning')
    expect(screen.queryByRole('listbox')).toBeNull()
    // Type-ahead while closed opens on the match; Esc closes without choosing.
    await user.keyboard('e')
    expect(box.getAttribute('aria-activedescendant')).toBe(within(screen.getByRole('listbox')).getByRole('option', { name: 'Error' }).id)
    await user.keyboard('{Escape}')
    expect(box).toHaveTextContent('Warning')
    await user.keyboard('{End}{Enter}')
    expect(box).toHaveTextContent('Error')
    await pickOption(user, box, 'All')
    expect(box).toHaveTextContent('All')
  })
})

describe('Combobox', () => {
  function Demo({ onChange }: { onChange: (v: string) => void }) {
    const [v, setV] = useState('')
    return (
      <ComboboxField
        label="Profile"
        value={v}
        placeholder="All profiles"
        allowCustom
        emptyText="No matches"
        onChange={(x) => {
          setV(x)
          onChange(x)
        }}
        options={[
          { value: '', label: 'All profiles' },
          { value: 'office', label: 'office' },
          { value: 'home', label: 'home' },
          { value: 'office-2', label: 'office-2' },
        ]}
      />
    )
  }

  it('filters as you type, Enter picks the active option, "All" clears, free text is kept', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    withPrefs(<Demo onChange={onChange} />)
    const input = screen.getByRole('combobox', { name: 'Profile' })
    await user.type(input, 'off')
    const list = screen.getByRole('listbox')
    expect(within(list).getAllByRole('option').map((o) => o.textContent)).toEqual(['office', 'office-2'])
    await user.keyboard('{ArrowDown}{Enter}')
    expect(onChange).toHaveBeenLastCalledWith('office-2')
    expect(input).toHaveValue('office-2')
    await user.clear(input)
    await user.type(input, 'zzz')
    expect(within(screen.getByRole('listbox')).getByText('No matches')).toBeInTheDocument()
    await user.keyboard('{Enter}')
    expect(onChange).toHaveBeenLastCalledWith('zzz')
    await user.clear(input)
    await user.keyboard('{ArrowDown}')
    await user.click(within(screen.getByRole('listbox')).getByRole('option', { name: 'All profiles' }))
    expect(onChange).toHaveBeenLastCalledWith('')
    expect(input).toHaveValue('')
  })
})

describe('DateRangePicker', () => {
  it('local-day helpers: months clamp, grids start on Monday (es) or Sunday (en), presets', () => {
    expect(addMonths('2026-01-31', 1)).toBe('2026-02-28')
    expect(addDays('2026-03-01', -1)).toBe('2026-02-28')
    expect(weekStart('es')).toBe(1)
    expect(weekStart('en')).toBe(0)
    // 1 Nov 2026 is a Sunday: last cell of the first week in ES, first in EN.
    expect(monthGrid('2026-11-19', 1)[0]).toEqual(['', '', '', '', '', '', '2026-11-01'])
    expect(monthGrid('2026-11-19', 0)[0]?.[0]).toBe('2026-11-01')
    expect(presetRange('last7', '2026-11-19')).toEqual({ from: '2026-11-13', to: '2026-11-19' })
    expect(matchPreset({ from: '2026-10-21', to: '2026-11-19' }, '2026-11-19')).toBe('last30')
    expect(matchPreset({ from: '2026-11-01', to: '2026-11-19' }, '2026-11-19')).toBeNull()
  })

  function Demo({ onChange }: { onChange: (r: DateRange) => void }) {
    const [r, setR] = useState<DateRange>({ from: '', to: '' })
    return (
      <DateRangePicker
        aria-label="Dates"
        today="2026-11-19"
        value={r}
        onChange={(x) => {
          setR(x)
          onChange(x)
        }}
      />
    )
  }

  it('one control: presets apply at once; a custom range by keyboard (arrows, Enter twice, Apply)', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    withPrefs(<Demo onChange={onChange} />)
    const trigger = screen.getByRole('button', { name: /Dates/ })
    expect(trigger).toHaveTextContent('Any date')
    await user.click(trigger)
    const dialog = screen.getByRole('dialog', { name: 'Choose dates' })
    await user.click(within(dialog).getByRole('button', { name: 'Last 7 days' }))
    expect(onChange).toHaveBeenLastCalledWith({ from: '2026-11-13', to: '2026-11-19' })
    expect(trigger).toHaveTextContent('Last 7 days')
    expect(trigger).toHaveFocus()

    await user.click(trigger)
    await user.click(screen.getByRole('button', { name: 'Custom' }))
    const grid = screen.getByRole('grid')
    // The calendar opens on the range's end day, focused.
    expect(document.activeElement).toHaveAttribute('data-day', '2026-11-19')
    // Future days cannot be picked.
    expect(within(grid).getByRole('button', { name: /November 20, 2026/ })).toHaveAttribute('aria-disabled', 'true')
    await user.keyboard('{ArrowLeft}{ArrowLeft}{Enter}')
    expect(screen.getByText('Now pick the last day (it can be the same one).')).toBeInTheDocument()
    await user.keyboard('{ArrowUp}{Enter}')
    // The second pick was earlier: the range is put in order.
    expect(screen.getByText('Nov 10 to Nov 17')).toBeInTheDocument()
    await user.keyboard('{PageUp}')
    expect(document.activeElement).toHaveAttribute('data-day', '2026-10-10')
    await user.click(screen.getByRole('button', { name: 'Apply' }))
    expect(onChange).toHaveBeenLastCalledWith({ from: '2026-11-10', to: '2026-11-17' })
    expect(trigger).toHaveTextContent(/Nov 10.*17, 2026/)

    await user.click(trigger)
    await user.click(screen.getByRole('button', { name: 'Clear dates' }))
    expect(onChange).toHaveBeenLastCalledWith({ from: '', to: '' })
    expect(trigger).toHaveTextContent('Any date')
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
