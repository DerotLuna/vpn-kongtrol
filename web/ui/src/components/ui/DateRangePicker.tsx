// Kongtrol UI kit — DateRangePicker ("Desde–Hasta" as one control) · v2.0.0 · 2026-11-19
//
// One button shows the range ("Todas las fechas", "Últimos 7 días",
// "12–19 nov 2026") and opens a non-modal dialog with presets (Hoy,
// Últimos 7 días, Últimos 30 días, Personalizado) and, for a custom range,
// React Aria Components' RangeCalendar (ADR-0068): the APG date-grid
// keyboard (arrows by day / week, Home/End, PageUp/PageDown by month,
// Shift by year, Enter picks), localized month and day names from the
// I18nProvider (Monday first in Spanish, Sunday in English), days after
// `max` (default today: logs have no future) disabled, and the range drawn
// live while choosing: after the first day, hovering or moving with the
// arrows paints every day up to the pointer (data-selected) before the
// second click. Aplicar commits. Values are local days "YYYY-MM-DD"
// ('' = open end).
import { parseDate, type CalendarDate } from '@internationalized/date'
import { useContext, useMemo, useState } from 'react'
import {
  Button,
  CalendarCell,
  CalendarGrid,
  CalendarGridBody,
  CalendarGridHeader,
  CalendarHeaderCell,
  Dialog,
  DialogTrigger,
  Heading,
  Popover,
  RangeCalendar,
  RangeCalendarStateContext,
  type RangeValue,
} from 'react-aria-components'
import { usePreferences } from '../../app/preferences-context'
import { DATE_PRESETS, fromISODay, matchPreset, presetRange, toISODay, type DateRange } from './dates'
import { CalendarIcon, ChevronLeftIcon, ChevronRightIcon } from './icons'
import { defined } from './options'

interface Props {
  value: DateRange
  onChange: (value: DateRange) => void
  /** Last day that can be picked (default: today). */
  max?: string | undefined
  id?: string | undefined
  'aria-labelledby'?: string | undefined
  'aria-label'?: string | undefined
  'aria-describedby'?: string | undefined
  'aria-invalid'?: boolean | undefined
  /** Test hook: today's date. */
  today?: string | undefined
}

export function DateRangePicker({ value, onChange, max, today: todayProp, id, 'aria-invalid': _invalid, ...a11y }: Props) {
  const { t, lang } = usePreferences()
  const [open, setOpen] = useState(false)
  // Read once per mount (render must stay pure); the picker is short-lived.
  const [now] = useState(() => toISODay(new Date()))
  const today = todayProp ?? now
  const preset = matchPreset(value, today)

  const fmt = useMemo(() => new Intl.DateTimeFormat(lang, { day: 'numeric', month: 'short', year: 'numeric' }), [lang])
  let label: string
  if (!value.from && !value.to) label = t('kit.date.any')
  else if (preset) label = t(`kit.date.${preset}`)
  else if (value.from && value.to) label = fmt.formatRange(fromISODay(value.from), fromISODay(value.to))
  else label = value.from ? `${fmt.format(fromISODay(value.from))} –` : `– ${fmt.format(fromISODay(value.to))}`

  return (
    <DialogTrigger isOpen={open} onOpenChange={setOpen}>
      <Button className="k-input k-select k-drp__trigger" {...defined({ id, ...a11y })}>
        <span className="k-drp__icon" aria-hidden="true">
          <CalendarIcon />
        </span>
        <span className={value.from || value.to ? 'k-select__value' : 'k-select__value k-select__placeholder'}>{label}</span>
      </Button>
      <Popover className="k-pop k-drp" placement="bottom start" offset={4} containerPadding={8}>
        <Dialog className="k-drp__dialog" aria-label={t('kit.date.dialog')}>
          <RangePanel
            value={value}
            today={today}
            max={max ?? today}
            onPick={(v) => {
              onChange(v)
              setOpen(false)
            }}
          />
        </Dialog>
      </Popover>
    </DialogTrigger>
  )
}

function toRange(v: DateRange): RangeValue<CalendarDate> | null {
  if (!v.from || !v.to) return null
  return { start: parseDate(v.from), end: parseDate(v.to) }
}

/** "Elige el primer día" → "Ahora elige el último día" while a range is half chosen → "Del … al …". */
function Status({ done }: { done: string }) {
  const { t } = usePreferences()
  const state = useContext(RangeCalendarStateContext)
  return (
    <p className="k-drp__status" aria-live="polite">
      {state?.anchorDate ? t('kit.date.pickEnd') : done}
    </p>
  )
}

function RangePanel({ value, today, max, onPick }: { value: DateRange; today: string; max: string; onPick: (v: DateRange) => void }) {
  const { t, lang } = usePreferences()
  const preset = matchPreset(value, today)
  const [custom, setCustom] = useState(!preset && !!(value.from || value.to))
  const [draft, setDraft] = useState<RangeValue<CalendarDate> | null>(preset ? null : toRange(value))
  const short = new Intl.DateTimeFormat(lang, { day: 'numeric', month: 'short' })
  const status = draft
    ? t('kit.date.chosen', { from: short.format(fromISODay(draft.start.toString())), to: short.format(fromISODay(draft.end.toString())) })
    : t('kit.date.pickStart')

  return (
    <div className="k-drp__body">
      <div className="k-drp__presets" role="group" aria-label={t('kit.date.presets')}>
        {DATE_PRESETS.map((p) => (
          <button
            key={p}
            type="button"
            className="k-drp__preset"
            aria-pressed={!custom && preset === p}
            autoFocus={!custom && preset === p}
            onClick={() => onPick(presetRange(p, today))}
          >
            {t(`kit.date.${p}`)}
          </button>
        ))}
        <button type="button" className="k-drp__preset" aria-pressed={custom} aria-expanded={custom} onClick={() => setCustom(true)}>
          {t('kit.date.custom')}
        </button>
      </div>

      {custom && (
        <RangeCalendar
          className="k-drp__cal"
          aria-label={t('kit.date.dialog')}
          value={draft}
          onChange={setDraft}
          maxValue={parseDate(max)}
          defaultFocusedValue={parseDate(value.to || value.from || today)}
          autoFocus
        >
          <header className="k-drp__month">
            <Button slot="previous" className="k-icon-btn k-icon-btn--sm" aria-label={t('kit.date.prevMonth')}>
              <ChevronLeftIcon />
            </Button>
            <Heading className="k-drp__title" />
            <Button slot="next" className="k-icon-btn k-icon-btn--sm" aria-label={t('kit.date.nextMonth')}>
              <ChevronRightIcon />
            </Button>
          </header>
          <CalendarGrid className="k-drp__grid" weekdayStyle="narrow">
            <CalendarGridHeader>{(day) => <CalendarHeaderCell>{day}</CalendarHeaderCell>}</CalendarGridHeader>
            <CalendarGridBody>{(date) => <CalendarCell date={date} className="k-drp__day" />}</CalendarGridBody>
          </CalendarGrid>
          <Status done={status} />
          <p className="k-drp__keys">{t('kit.date.keys')}</p>
        </RangeCalendar>
      )}

      <div className="k-drp__foot">
        {(value.from || value.to || draft) && (
          <button type="button" className="k-btn k-btn--text k-btn--sm" onClick={() => onPick({ from: '', to: '' })}>
            <span className="k-btn__label">{t('kit.date.clear')}</span>
          </button>
        )}
        {custom && (
          <button
            type="button"
            className="k-btn k-btn--filled k-btn--sm"
            disabled={!draft}
            onClick={() => draft && onPick({ from: draft.start.toString(), to: draft.end.toString() })}
          >
            <span className="k-btn__label">{t('kit.date.apply')}</span>
          </button>
        )}
      </div>
    </div>
  )
}
