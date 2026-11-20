// Kongtrol UI kit — Select · v2.0.0 · 2026-11-19
//
// The panel's own select instead of the browser's <select> (which looks
// different on every system and ignores the theme), built on React Aria
// Components' Select + ListBox (ADR-0068): the WAI-ARIA listbox pattern
// with keyboard (arrows, Home/End, PageUp/PageDown, type-ahead, Enter/
// Space, Esc), screen reader names and focus handling from a mature,
// tested library instead of our own. Groups render like <optgroup>.
// Styling (kit-controls.css): the input's box with a chevron; the list has
// no open/close animation; the chosen option is a check plus a subtle
// background, the focused one the hover background (no side bar).
import { useMemo, type ReactNode } from 'react'
import { Button, Header, ListBox, ListBoxItem, ListBoxSection, Popover, Select as AriaSelect, SelectValue, Text, type Key } from 'react-aria-components'
import { CheckIcon, ChevronDownIcon } from './icons'
import { defined, groupOptions, type ControlA11y, type SelectOption } from './options'

export type { ControlA11y, SelectOption } from './options'

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

/** One option row: the check (chosen), the label and an optional hint. */
export function OptionRow({ label, hint }: { label: string; hint?: string | undefined }) {
  return (
    <>
      <span className="k-listbox__check" aria-hidden="true">
        <CheckIcon />
      </span>
      <span className="k-listbox__text">
        <Text slot="label">{label}</Text>
        {hint && (
          <Text slot="description" className="k-listbox__hint">
            {hint}
          </Text>
        )}
      </span>
    </>
  )
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
  id,
  'aria-invalid': invalid,
  ...a11y
}: SelectProps<V>) {
  const sections = useMemo(() => groupOptions(options), [options])
  const known = options.some((o) => o.value === value)
  const disabledKeys = useMemo(() => options.filter((o) => o.disabled).map((o) => o.value), [options])
  const item = (o: SelectOption<V>): ReactNode => (
    <ListBoxItem key={o.value} id={o.value} textValue={o.label} className="k-listbox__option">
      <OptionRow label={o.label} hint={o.hint} />
    </ListBoxItem>
  )
  return (
    <AriaSelect
      className={`k-select-root ${className}`.trim()}
      selectedKey={known ? value : null}
      onSelectionChange={(k: Key | null) => {
        if (k != null && k !== value) onChange(String(k) as V)
      }}
      disabledKeys={disabledKeys}
      placeholder={placeholder}
      {...defined({ isDisabled: disabled, isInvalid: invalid, name, ...a11y })}
    >
      <Button {...defined({ id })} className={`k-input k-select${mono ? ' mono' : ''}`}>
        <SelectValue className="k-select__value">
          {({ isPlaceholder, selectedText }) => <span className={isPlaceholder ? 'k-select__placeholder' : undefined}>{isPlaceholder ? placeholder : selectedText}</span>}
        </SelectValue>
        <span className="k-select__chevron" aria-hidden="true">
          <ChevronDownIcon />
        </span>
      </Button>
      <Popover className={`k-pop k-listbox-pop${mono ? ' mono' : ''}`} offset={4} containerPadding={8}>
        <ListBox className="k-listbox">
          {sections.map((s, i) =>
            s.group ? (
              <ListBoxSection key={`g${i}`} id={`g${i}`} className="k-listbox__section">
                <Header className="k-listbox__group">{s.group}</Header>
                {s.items.map(item)}
              </ListBoxSection>
            ) : (
              s.items.map(item)
            ),
          )}
        </ListBox>
      </Popover>
    </AriaSelect>
  )
}
