import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { PreferencesProvider } from '../../app/PreferencesProvider'
import { toasts } from '../../data/toast'
import { ARIA_PRESS_STYLE_ID, guardAriaStyle } from '../../lib/ariaStyleGuard'
import { json, pickOption, stubFetch } from '../../test/fakes'
import { addDays, matchPreset, presetRange, type DateRange } from './dates'
import { DateRangePicker } from './DateRangePicker'
import { TOOLTIP_DELAY_MS } from './Tooltip'
import { Button, IconButton } from './Button'
import { DataList } from './DataList'
import { Dialog } from './Dialog'
import { MultiSelectField, SelectField, TextField } from './Field'
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

describe('IconButton + Tooltip (React Aria)', () => {
  const bubble = () => document.querySelector('.k-tooltip')

  it('opens on hover only after the delay', async () => {
    const user = userEvent.setup()
    withPrefs(<IconButton label="Refresh" icon={<CloseIcon />} />)
    const b = screen.getByRole('button', { name: 'Refresh' })
    // React Aria opens hover tooltips only in pointer modality (and right away
    // for a while after another one closed, so this runs first).
    await user.click(document.body)
    const start = performance.now()
    await user.hover(b)
    expect(bubble()).toBeNull()
    expect(await screen.findByRole('tooltip')).toHaveTextContent('Refresh')
    expect(performance.now() - start).toBeGreaterThanOrEqual(TOOLTIP_DELAY_MS - 20)
    await user.unhover(b)
    await waitFor(() => expect(bubble()).toBeNull())
  })
  it('is named by its label; the tooltip shows on keyboard focus, hides with Esc, and is not read twice', async () => {
    withPrefs(<IconButton label="Close panel" icon={<CloseIcon />} />)
    const b = screen.getByRole('button', { name: 'Close panel' })
    expect(bubble()).toBeNull()
    await userEvent.tab()
    expect(b).toHaveFocus()
    expect(await screen.findByRole('tooltip')).toHaveTextContent('Close panel')
    // The tooltip repeats the name: it does not describe the button again.
    expect(b).toHaveAccessibleDescription('')
    await userEvent.keyboard('{Escape}')
    await waitFor(() => expect(bubble()).toBeNull())
    expect(b).toHaveFocus()
  })

  it('a longer tip describes the button while it shows', async () => {
    withPrefs(<IconButton label="Reconnect office" tip="Reconnect: re-reads its configuration" icon={<CloseIcon />} />)
    await userEvent.tab()
    const b = screen.getByRole('button', { name: 'Reconnect office' })
    await screen.findByRole('tooltip')
    expect(b).toHaveAccessibleDescription('Reconnect: re-reads its configuration')
  })

})

describe('Select (React Aria)', () => {
  function Demo() {
    const [v, setV] = useState('INFO')
    return (
      <SelectField
        label="Level"
        value={v}
        onChange={setV}
        options={[
          { value: '', label: 'All' },
          { value: 'INFO', label: 'Info', hint: 'Normal activity' },
          { value: 'WARN', label: 'Warning' },
          { value: 'ERROR', label: 'Error' },
        ]}
      />
    )
  }

  it('listbox pattern: named by its label, opens on the chosen option, arrows + Enter, type-ahead, Esc keeps the value', async () => {
    const user = userEvent.setup()
    withPrefs(<Demo />)
    const box = screen.getByRole('button', { name: /Level/ })
    expect(box).toHaveTextContent('Info')
    expect(box).toHaveAttribute('aria-haspopup', 'listbox')
    box.focus()
    await user.keyboard('{ArrowDown}')
    const list = await screen.findByRole('listbox')
    expect(box).toHaveAttribute('aria-expanded', 'true')
    const info = within(list).getByRole('option', { name: /Info/ })
    await waitFor(() => expect(info).toHaveFocus())
    // Chosen = a check + the selected state; the hint is the option's description.
    expect(info).toHaveAttribute('aria-selected', 'true')
    expect(info).toHaveAccessibleDescription('Normal activity')
    await user.keyboard('{ArrowDown}{Enter}')
    expect(box).toHaveTextContent('Warning')
    await waitFor(() => expect(screen.queryByRole('listbox')).toBeNull())
    expect(box).toHaveFocus()
    // Esc closes without choosing.
    await user.keyboard('{Enter}')
    await screen.findByRole('listbox')
    await user.keyboard('{ArrowDown}{Escape}')
    await waitFor(() => {
      expect(screen.queryByRole('listbox')).toBeNull()
      expect(box).toHaveFocus()
    })
    expect(box).toHaveTextContent('Warning')
    // Type-ahead inside the open list.
    await user.keyboard('{Enter}')
    await screen.findByRole('listbox')
    await user.keyboard('e')
    await user.keyboard('{Enter}')
    expect(box).toHaveTextContent('Error')
    await pickOption(user, box, 'All')
    expect(box).toHaveTextContent('All')
  })
})

describe('MultiSelect (React Aria)', () => {
  function Demo({ onChange }: { onChange: (v: string[]) => void }) {
    const [v, setV] = useState<string[]>([])
    return (
      <MultiSelectField
        label="Profile"
        value={v}
        placeholder="All profiles"
        searchLabel="Search profile"
        chipsLabel="Chosen profiles"
        onChange={(x) => {
          setV(x)
          onChange(x)
        }}
        options={['office', 'home', 'office-2', 'café'].map((p) => ({ value: p, label: p }))}
      />
    )
  }

  it('searches (case and accents ignored), toggles several, shows chips, removes and clears', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    withPrefs(<Demo onChange={onChange} />)
    const field = screen.getByRole('button', { name: /^Profile/ })
    expect(field).toHaveTextContent('All profiles')
    await user.click(field)
    const search = screen.getByRole('searchbox', { name: 'Search profile' })
    await waitFor(() => expect(search).toHaveFocus())
    await user.keyboard('OFF')
    const list = screen.getByRole('listbox', { name: 'Search profile' })
    expect(list).toHaveAttribute('aria-multiselectable', 'true')
    expect(within(list).getAllByRole('option').map((o) => o.textContent)).toEqual(['office', 'office-2'])
    // Arrows move in the list while focus stays in the search field.
    await user.keyboard('{ArrowDown}{ArrowDown}{Enter}')
    expect(onChange).toHaveBeenLastCalledWith(['office-2'])
    expect(search).toHaveFocus()
    await user.clear(search)
    await user.keyboard('cafe')
    expect(within(screen.getByRole('listbox')).getAllByRole('option').map((o) => o.textContent)).toEqual(['café'])
    await user.keyboard('{ArrowDown}{Enter}')
    // The list's order, not the click order.
    expect(onChange).toHaveBeenLastCalledWith(['office-2', 'café'])
    await user.clear(search)
    await user.keyboard('zzz')
    expect(screen.getByText('No matches')).toBeInTheDocument()
    // Esc clears the search first, then closes the list.
    await user.keyboard('{Escape}')
    expect(search).toHaveValue('')
    await user.keyboard('{Escape}')
    await waitFor(() => expect(screen.queryByRole('listbox')).toBeNull())
    await waitFor(() => expect(field).toHaveFocus())
    expect(field).toHaveTextContent('2 selected')
    const chips = screen.getByRole('grid', { name: 'Chosen profiles' })
    expect(within(chips).getAllByRole('row').map((r) => r.textContent)).toEqual(['office-2', 'café'])
    await user.click(within(chips).getByRole('button', { name: /Remove office-2/ }))
    expect(onChange).toHaveBeenLastCalledWith(['café'])
    expect(field).toHaveTextContent('café')
    await user.click(screen.getByRole('button', { name: 'Clear all' }))
    expect(onChange).toHaveBeenLastCalledWith([])
    expect(field).toHaveTextContent('All profiles')
    expect(screen.queryByRole('grid', { name: 'Chosen profiles' })).toBeNull()
  })
})

describe('DateRangePicker (React Aria RangeCalendar)', () => {
  it('local-day helpers: presets', () => {
    expect(addDays('2026-03-01', -1)).toBe('2026-02-28')
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

  const day = (n: number) => screen.getByRole('button', { name: new RegExp(`November ${n}, 2026`) })
  const highlighted = () =>
    Array.from(document.querySelectorAll('.k-drp__day[data-selected]')).map((el) => Number(el.textContent))

  it('presets apply at once; a custom range paints live while choosing, by pointer and by keyboard', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    withPrefs(<Demo onChange={onChange} />)
    const trigger = screen.getByRole('button', { name: /Dates/ })
    expect(trigger).toHaveTextContent('Any date')
    await user.click(trigger)
    const dialog = await screen.findByRole('dialog', { name: 'Choose dates' })
    await user.click(within(dialog).getByRole('button', { name: 'Last 7 days' }))
    expect(onChange).toHaveBeenLastCalledWith({ from: '2026-11-13', to: '2026-11-19' })
    expect(trigger).toHaveTextContent('Last 7 days')
    await waitFor(() => expect(trigger).toHaveFocus())

    await user.click(trigger)
    await user.click(await screen.findByRole('button', { name: 'Custom' }))
    const grid = screen.getByRole('grid')
    // Week starts on Sunday in English.
    expect(grid.querySelector('thead th')).toHaveTextContent('S')
    // Future days cannot be picked.
    expect(day(20)).toHaveAttribute('aria-disabled', 'true')
    // First click = start; hovering paints the range up to the pointer
    // before the second click.
    await user.click(day(10))
    expect(screen.getByText('Now pick the last day (it can be the same one).')).toBeInTheDocument()
    await user.hover(day(14))
    expect(highlighted()).toEqual([10, 11, 12, 13, 14])
    await user.click(day(17))
    expect(highlighted()).toEqual([10, 11, 12, 13, 14, 15, 16, 17])
    expect(screen.getByText('Nov 10 to Nov 17')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Apply' }))
    expect(onChange).toHaveBeenLastCalledWith({ from: '2026-11-10', to: '2026-11-17' })
    expect(trigger).toHaveTextContent(/Nov 10.*17, 2026/)

    await user.click(trigger)
    await user.click(await screen.findByRole('button', { name: 'Clear dates' }))
    expect(onChange).toHaveBeenLastCalledWith({ from: '', to: '' })
    expect(trigger).toHaveTextContent('Any date')
  })

  it('keyboard: Enter starts, arrows paint the range as they move, Enter ends, PageUp changes month', async () => {
    const user = userEvent.setup()
    withPrefs(<Demo onChange={() => {}} />)
    await user.click(screen.getByRole('button', { name: /Dates/ }))
    await user.click(await screen.findByRole('button', { name: 'Custom' }))
    // The calendar opens on today, focused.
    await waitFor(() => expect(document.activeElement).toHaveAccessibleName(/November 19, 2026/))
    await user.keyboard('{Enter}')
    await user.keyboard('{ArrowLeft}')
    await user.keyboard('{ArrowLeft}')
    // The painted range follows the focused day (end = today).
    await waitFor(() => expect(highlighted().length).toBeGreaterThan(2))
    const twoBack = highlighted()
    expect(twoBack.at(-1)).toBe(19)
    await user.keyboard('{ArrowUp}')
    await waitFor(() => expect(highlighted().length).toBe(twoBack.length + 7))
    await user.keyboard('{Enter}')
    expect(screen.getByText(`Nov ${twoBack[0]! - 7} to Nov 19`)).toBeInTheDocument()
    await user.keyboard('{PageUp}')
    await waitFor(() => expect(document.activeElement).toHaveAccessibleName(/October/))
  })
})

describe('React Aria under the CSP', () => {
  it('the <style> guard id still matches the installed react-aria', async () => {
    // Node built-ins, typed by hand: the app's tsconfig has no Node types.
    const fsName: string = 'node:fs'
    const moduleName: string = 'node:module'
    const { readFileSync } = (await import(/* @vite-ignore */ fsName)) as { readFileSync: (p: string, enc: string) => string }
    const { createRequire } = (await import(/* @vite-ignore */ moduleName)) as {
      createRequire: (url: string) => { resolve: (id: string) => string }
    }
    const pkg = createRequire(import.meta.url).resolve('react-aria/package.json')
    const src = readFileSync(pkg.replace(/package\.json$/, 'dist/private/interactions/usePress.mjs'), 'utf8')
    expect(src).toContain(`'${ARIA_PRESS_STYLE_ID}'`)
    guardAriaStyle()
    guardAriaStyle()
    expect(document.querySelectorAll(`#${ARIA_PRESS_STYLE_ID}`)).toHaveLength(1)
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

  it('sheet: a Select inside lists into the <dialog> (top layer); only a real backdrop click closes it', async () => {
    const onClose = vi.fn()
    function Sheet() {
      const [v, setV] = useState('a')
      return (
        <Dialog open onClose={onClose} title="Add" variant="sheet">
          <SelectField label="Through" value={v} onChange={setV} options={[{ value: 'a', label: 'office' }, { value: 'b', label: 'home' }]} />
        </Dialog>
      )
    }
    const user = userEvent.setup()
    withPrefs(<Sheet />)
    const dialog = document.querySelector('dialog')!
    await user.click(screen.getByRole('button', { name: /Through/ }))
    const list = await screen.findByRole('listbox')
    expect(dialog.contains(list)).toBe(true)
    // A press that started on a control and ended elsewhere is not a backdrop click.
    fireEvent.pointerDown(screen.getByRole('option', { name: 'home' }))
    fireEvent.click(dialog)
    expect(onClose).not.toHaveBeenCalled()
    await user.click(screen.getByRole('option', { name: 'home' }))
    expect(screen.getByRole('button', { name: /Through/ })).toHaveTextContent('home')
    fireEvent.pointerDown(dialog)
    fireEvent.click(dialog)
    expect(onClose).toHaveBeenCalledTimes(1)
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
