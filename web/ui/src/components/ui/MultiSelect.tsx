// Kongtrol UI kit — MultiSelect (searchable, several values) · v1.0.0 · 2026-11-19
//
// Pick any number of values from a long list (Actividad → Registro:
// "Perfil"), built on React Aria Components (ADR-0068): a button opens a
// popover with a search field (Autocomplete: typing filters the list,
// ignoring case and accents; arrows move in the list while focus stays in
// the field) and a ListBox with selectionMode="multiple" (Enter or a click
// toggles an option; each chosen one shows a check). The chosen values
// stay visible under the button as chips (TagGroup: arrows move between
// chips, Delete / Backspace or the × removes one) with a "Quitar todos"
// button. Nothing chosen = the placeholder ("Todos los perfiles").
import { useId, useMemo } from 'react'
import {
  Autocomplete,
  Button,
  Dialog,
  DialogTrigger,
  Input,
  ListBox,
  ListBoxItem,
  Popover,
  SearchField,
  Tag,
  TagGroup,
  TagList,
  useFilter,
  type Selection,
} from 'react-aria-components'
import { usePreferences } from '../../app/preferences-context'
import { ChevronDownIcon, CloseIcon, SearchIcon } from './icons'
import { defined, type ControlA11y, type SelectOption } from './options'
import { OptionRow } from './Select'

export interface MultiSelectProps extends ControlA11y {
  value: readonly string[]
  onChange: (value: string[]) => void
  options: readonly SelectOption[]
  /** What "nothing chosen" means ("Todos los perfiles"). */
  placeholder: string
  /** Accessible name of the search field and the chips ("Perfiles elegidos"). */
  searchLabel: string
  chipsLabel: string
  emptyText?: string
  mono?: boolean | undefined
  className?: string | undefined
  disabled?: boolean | undefined
}

export function MultiSelect({
  value,
  onChange,
  options,
  placeholder,
  searchLabel,
  chipsLabel,
  emptyText,
  mono,
  className = '',
  disabled,
  id,
  'aria-labelledby': labelledBy,
  'aria-label': ariaLabel,
  'aria-describedby': describedBy,
}: MultiSelectProps) {
  const { t } = usePreferences()
  const autoId = useId()
  const valueId = `${id ?? autoId}-v`
  const { contains } = useFilter({ sensitivity: 'base' })
  const labelOf = useMemo(() => new Map(options.map((o) => [o.value, o.label])), [options])
  const chosen = value.filter(Boolean)
  const summary = chosen.length === 0 ? placeholder : chosen.length === 1 ? (labelOf.get(chosen[0]!) ?? chosen[0]!) : t('kit.multi.count', { n: chosen.length })
  const set = (sel: Selection) => {
    // "all" never happens (no select-all); keep the list's order.
    const keys = sel === 'all' ? options.map((o) => o.value) : options.map((o) => o.value).filter((v) => sel.has(v))
    onChange(keys)
  }

  return (
    <div className={`k-multi ${className}`.trim()}>
      <DialogTrigger>
        <Button
          className={`k-input k-select${mono ? ' mono' : ''}`}
          {...defined({
            id,
            isDisabled: disabled,
            'aria-labelledby': labelledBy ? `${labelledBy} ${valueId}` : undefined,
            'aria-label': ariaLabel,
            'aria-describedby': describedBy,
          })}
        >
          <span id={valueId} className={chosen.length ? 'k-select__value' : 'k-select__value k-select__placeholder'}>
            {summary}
          </span>
          <span className="k-select__chevron" aria-hidden="true">
            <ChevronDownIcon />
          </span>
        </Button>
        <Popover className="k-pop k-multi__pop" offset={4} containerPadding={8}>
          <Dialog className="k-multi__dialog" aria-label={searchLabel}>
            <Autocomplete filter={contains}>
              <SearchField className="k-multi__search" aria-label={searchLabel} autoFocus>
                <span className="k-multi__search-icon" aria-hidden="true">
                  <SearchIcon />
                </span>
                <Input className={`k-input${mono ? ' mono' : ''}`} placeholder={searchLabel} />
              </SearchField>
              <ListBox
                className={`k-listbox${mono ? ' mono' : ''}`}
                aria-label={searchLabel}
                selectionMode="multiple"
                // Esc closes the list; it must never clear the chosen values.
                escapeKeyBehavior="none"
                selectedKeys={chosen}
                onSelectionChange={set}
                items={options}
                renderEmptyState={() => <p className="k-listbox__empty">{emptyText ?? t('kit.select.empty')}</p>}
              >
                {(o) => (
                  <ListBoxItem id={o.value} textValue={o.label} className="k-listbox__option">
                    <OptionRow label={o.label} hint={o.hint} />
                  </ListBoxItem>
                )}
              </ListBox>
            </Autocomplete>
          </Dialog>
        </Popover>
      </DialogTrigger>
      {chosen.length > 0 && (
        <div className="k-multi__chips">
          <TagGroup aria-label={chipsLabel} onRemove={(keys) => onChange(chosen.filter((v) => !keys.has(v)))}>
            <TagList className="k-chips">
              {chosen.map((v) => (
                <Tag key={v} id={v} textValue={labelOf.get(v) ?? v} className={`k-chip${mono ? ' mono' : ''}`}>
                  <span>{labelOf.get(v) ?? v}</span>
                  <Button slot="remove" className="k-chip__remove" aria-label={t('kit.multi.remove', { name: labelOf.get(v) ?? v })}>
                    <CloseIcon />
                  </Button>
                </Tag>
              ))}
            </TagList>
          </TagGroup>
          <Button className="k-btn k-btn--text k-btn--sm" onPress={() => onChange([])}>
            <span className="k-btn__label">{t('kit.multi.clear')}</span>
          </Button>
        </div>
      )}
    </div>
  )
}
