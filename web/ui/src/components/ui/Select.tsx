// Kongtrol UI kit — Select and Combobox · v1.0.0 · 2026-11-19
//
// Own controls instead of the browser's <select> / <datalist>, which look
// different on every system and ignore the panel's theme (owner feedback
// 2026-11-19). Both follow the WAI-ARIA APG combobox patterns, with focus
// staying on the control and the active option named by
// aria-activedescendant:
// - Select ("select-only combobox"): a button role="combobox". Closed:
//   ArrowDown/ArrowUp/Enter/Space/Alt+ArrowDown open it, Home/End open on the
//   first/last option, a letter opens it on the next matching option.
//   Open: ArrowUp/Down, Home/End, PageUp/PageDown (10), type-ahead;
//   Enter/Space/Tab choose, Esc closes without choosing.
// - Combobox ("editable, list autocomplete"): a text input that filters the
//   options as you type; ArrowDown opens, Enter chooses the active option,
//   Esc closes (a second Esc restores the chosen value). With allowCustom,
//   what was typed is the value when nothing matches.
// The list floats with Floating UI in the top layer (floating.ts, ADR-0064),
// at least as wide as the control and never taller than the space left.
import { useId, useLayoutEffect, useMemo, useRef, useState, type CSSProperties, type KeyboardEvent, type ReactNode } from 'react'
import { useAnchored, useDismiss, useTopLayer, useTypeahead } from './floating'
import { CheckIcon, ChevronDownIcon } from './icons'

export interface SelectOption<V extends string = string> {
  value: V
  label: string
  /** A second, quieter line (e.g. what the choice does). */
  hint?: string
  disabled?: boolean | undefined
  /** Heading of a group of options (like <optgroup>); consecutive options share it. */
  group?: string | undefined
}

interface ControlA11y {
  id?: string | undefined
  'aria-labelledby'?: string | undefined
  'aria-label'?: string | undefined
  'aria-describedby'?: string | undefined
  'aria-invalid'?: boolean | undefined
}

export interface SelectProps<V extends string> extends ControlA11y {
  value: V
  onChange: (value: V) => void
  options: readonly SelectOption<V>[]
  disabled?: boolean | undefined
  /** Shown when the value is not one of the options. */
  placeholder?: string | undefined
  mono?: boolean | undefined
  className?: string | undefined
  name?: string | undefined
}

const PAGE = 10

function nextEnabled<V extends string>(options: readonly SelectOption<V>[], from: number, step: number): number {
  const n = options.length
  if (n === 0) return -1
  let i = from
  for (let k = 0; k < n; k++) {
    i = Math.min(n - 1, Math.max(0, i + step))
    if (!options[i]?.disabled) return i
    if (i === 0 || i === n - 1) break
  }
  return from
}

export function Select<V extends string>({
  value,
  onChange,
  options,
  disabled,
  placeholder = '',
  mono,
  className = '',
  name,
  ...a11y
}: SelectProps<V>) {
  const listId = useId()
  const [open, setOpen] = useState(false)
  const [active, setActive] = useState(-1)
  const button = useRef<HTMLButtonElement | null>(null)
  const listRef = useRef<HTMLUListElement | null>(null)
  const { setReference, setFloating, floatingStyles } = useAnchored(open, { matchWidth: true, gap: 4 })
  const selected = options.findIndex((o) => o.value === value)
  const typeahead = useTypeahead(
    () => options.map((o) => o.label),
    (i) => {
      if (open) setActive(i)
      else {
        setActive(i)
        setOpen(true)
      }
    },
  )

  const show = (at: number) => {
    setActive(at < 0 ? nextEnabled(options, -1, 1) : at)
    setOpen(true)
  }
  const choose = (i: number) => {
    const o = options[i]
    if (o && !o.disabled && o.value !== value) onChange(o.value)
    setOpen(false)
  }
  useDismiss(open, [listRef, button], () => setOpen(false))

  const onKey = (e: KeyboardEvent<HTMLButtonElement>) => {
    if (disabled) return
    if (!open) {
      switch (e.key) {
        case 'ArrowDown':
        case 'ArrowUp':
        case 'Enter':
        case ' ':
          show(selected)
          break
        case 'Home':
          show(nextEnabled(options, -1, 1))
          break
        case 'End':
          show(nextEnabled(options, options.length, -1))
          break
        default:
          if (!typeahead(e.key, selected)) return
      }
      e.preventDefault()
      return
    }
    switch (e.key) {
      case 'ArrowDown':
        setActive((i) => nextEnabled(options, i, 1))
        break
      case 'ArrowUp':
        if (e.altKey) choose(active)
        else setActive((i) => nextEnabled(options, i, -1))
        break
      case 'PageDown':
        setActive((i) => nextEnabled(options, Math.min(options.length - 1, i + PAGE - 1), 1))
        break
      case 'PageUp':
        setActive((i) => nextEnabled(options, Math.max(0, i - PAGE + 1), -1))
        break
      case 'Home':
        setActive(nextEnabled(options, -1, 1))
        break
      case 'End':
        setActive(nextEnabled(options, options.length, -1))
        break
      case 'Enter':
      case ' ':
        choose(active)
        break
      case 'Tab':
        choose(active)
        return
      default:
        if (!typeahead(e.key, active)) return
    }
    e.preventDefault()
  }

  const current = options[selected]
  return (
    <>
      <button
        ref={(el) => {
          button.current = el
          setReference(el)
        }}
        type="button"
        role="combobox"
        name={name}
        className={`k-input k-select${mono ? ' mono' : ''} ${className}`.trim()}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls={open ? listId : undefined}
        aria-activedescendant={open && active >= 0 ? `${listId}-${active}` : undefined}
        disabled={disabled}
        onClick={() => (open ? setOpen(false) : show(selected))}
        onKeyDown={onKey}
        {...a11y}
      >
        <span className={current ? 'k-select__value' : 'k-select__value k-select__placeholder'}>{current?.label ?? placeholder}</span>
        <span className="k-select__chevron" aria-hidden="true">
          <ChevronDownIcon />
        </span>
      </button>
      {open && (
        <Listbox
          id={listId}
          listRef={listRef}
          setFloating={setFloating}
          style={floatingStyles}
          options={options}
          active={active}
          selectedValue={value}
          labelledBy={a11y['aria-labelledby']}
          mono={mono}
          onHover={setActive}
          onPick={(i) => {
            choose(i)
            button.current?.focus()
          }}
        />
      )}
    </>
  )
}

function Listbox<V extends string>({
  id,
  listRef,
  setFloating,
  style,
  options,
  active,
  selectedValue,
  labelledBy,
  mono,
  onHover,
  onPick,
  empty,
}: {
  id: string
  listRef: { current: HTMLUListElement | null }
  setFloating: (el: HTMLElement | null) => void
  style: CSSProperties
  options: readonly SelectOption<V>[]
  active: number
  selectedValue: string
  labelledBy: string | undefined
  mono: boolean | undefined
  onHover: (i: number) => void
  onPick: (i: number) => void
  empty?: ReactNode
}) {
  useTopLayer(listRef)
  // Keep the active option in view (keyboard moves through a long list).
  useLayoutEffect(() => {
    const el = listRef.current?.querySelector<HTMLElement>(`[data-index="${active}"]`)
    el?.scrollIntoView?.({ block: 'nearest' })
  }, [active, listRef])
  return (
    <ul
      ref={(el) => {
        listRef.current = el
        setFloating(el)
      }}
      id={id}
      role="listbox"
      aria-labelledby={labelledBy}
      popover="manual"
      className={`k-listbox k-pop${mono ? ' mono' : ''}`}
      style={style}
      // Pressing an option must not move focus off the control first.
      onPointerDown={(e) => e.preventDefault()}
    >
      {options.length === 0 && empty && <li className="k-listbox__empty">{empty}</li>}
      {options.map((o, i) => {
        const sel = o.value === selectedValue
        const head = o.group && o.group !== options[i - 1]?.group ? o.group : null
        return [
          head && (
            <li key={`g-${head}`} role="presentation" className="k-listbox__group">
              {head}
            </li>
          ),
          <li
            key={o.value}
            id={`${id}-${i}`}
            data-index={i}
            role="option"
            aria-selected={sel}
            aria-disabled={o.disabled || undefined}
            data-active={i === active || undefined}
            className="k-listbox__option"
            onPointerMove={() => onHover(i)}
            onClick={() => !o.disabled && onPick(i)}
          >
            <span className="k-listbox__check" aria-hidden="true">
              {sel && <CheckIcon />}
            </span>
            <span className="k-listbox__text">
              <span>{o.label}</span>
              {o.hint && <span className="k-listbox__hint">{o.hint}</span>}
            </span>
          </li>,
        ]
      })}
    </ul>
  )
}

export interface ComboboxProps extends ControlA11y {
  value: string
  onChange: (value: string) => void
  options: readonly SelectOption[]
  /** The value the "all / none" option stands for (shown as the input's placeholder). */
  placeholder?: string | undefined
  /** What was typed is the value when no option matches (e.g. a profile not seen yet). */
  allowCustom?: boolean
  /** Shown in the list when nothing matches. */
  emptyText?: string
  mono?: boolean | undefined
  className?: string | undefined
  disabled?: boolean | undefined
}

function normalize(s: string): string {
  return s.normalize('NFD').replace(/\p{Diacritic}/gu, '').toLocaleLowerCase()
}

export function Combobox({
  value,
  onChange,
  options,
  placeholder = '',
  allowCustom = false,
  emptyText,
  mono,
  className = '',
  disabled,
  ...a11y
}: ComboboxProps) {
  const listId = useId()
  const labelOf = (v: string) => options.find((o) => o.value === v)?.label ?? v
  const shown = (v: string) => (options.find((o) => o.value === v && v !== '') ? labelOf(v) : v)
  const [text, setText] = useState(() => shown(value))
  const [typed, setTyped] = useState(false)
  const [open, setOpen] = useState(false)
  const [active, setActive] = useState(-1)
  const input = useRef<HTMLInputElement | null>(null)
  const listRef = useRef<HTMLUListElement | null>(null)
  const { setReference, setFloating, floatingStyles } = useAnchored(open, { matchWidth: true, gap: 4 })

  // Follow a value changed from outside (e.g. "Borrar filtros").
  const [seen, setSeen] = useState(value)
  if (seen !== value) {
    setSeen(value)
    setText(shown(value))
    setTyped(false)
  }

  const filtered = useMemo(() => {
    if (!typed || text.trim() === '') return options
    const q = normalize(text.trim())
    return options.filter((o) => o.value !== '' && normalize(o.label).includes(q))
  }, [options, text, typed])

  const commit = (v: string) => {
    setOpen(false)
    setTyped(false)
    setText(shown(v))
    if (v !== value) onChange(v)
  }
  const commitTyped = () => {
    const q = text.trim()
    if (q === '') return commit('')
    const exact = options.find((o) => normalize(o.label) === normalize(q))
    if (exact) return commit(exact.value)
    if (allowCustom) return commit(q)
    commit(value)
  }
  useDismiss(open, [listRef, input], (why) => (why === 'escape' ? setOpen(false) : commitTyped()))

  const onKey = (e: KeyboardEvent<HTMLInputElement>) => {
    switch (e.key) {
      case 'ArrowDown':
        if (!open) {
          setOpen(true)
          setActive(Math.max(0, filtered.findIndex((o) => o.value === value)))
        } else setActive((i) => Math.min(filtered.length - 1, i + 1))
        break
      case 'ArrowUp':
        if (open) setActive((i) => Math.max(0, i - 1))
        break
      case 'Enter':
        if (open && active >= 0 && filtered[active]) commit(filtered[active]!.value)
        else commitTyped()
        break
      case 'Escape':
        if (open) setOpen(false)
        else {
          setText(shown(value))
          setTyped(false)
        }
        break
      default:
        return
    }
    e.preventDefault()
  }

  return (
    <div className={`k-combobox ${className}`.trim()} ref={setReference}>
      <input
        ref={input}
        type="text"
        role="combobox"
        autoComplete="off"
        spellCheck={false}
        className={`k-input${mono ? ' mono' : ''}`}
        aria-autocomplete="list"
        aria-expanded={open}
        aria-controls={open ? listId : undefined}
        aria-activedescendant={open && active >= 0 && filtered[active] ? `${listId}-${active}` : undefined}
        placeholder={placeholder}
        value={text}
        disabled={disabled}
        onChange={(e) => {
          setText(e.target.value)
          setTyped(true)
          setOpen(true)
          setActive(e.target.value.trim() === '' ? -1 : 0)
        }}
        onClick={() => setOpen(true)}
        onBlur={commitTyped}
        onKeyDown={onKey}
        {...a11y}
      />
      <span className="k-select__chevron k-combobox__chevron" aria-hidden="true">
        <ChevronDownIcon />
      </span>
      {open && (
        <Listbox
          id={listId}
          listRef={listRef}
          setFloating={setFloating}
          style={floatingStyles}
          options={filtered}
          active={active}
          selectedValue={value}
          labelledBy={a11y['aria-labelledby']}
          mono={mono}
          onHover={setActive}
          onPick={(i) => {
            const o = filtered[i]
            if (o) commit(o.value)
            input.current?.focus()
          }}
          empty={emptyText}
        />
      )}
    </div>
  )
}
