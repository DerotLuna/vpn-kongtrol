// Kongtrol UI kit — DateRangePicker ("Desde–Hasta" as one control) · v1.0.0 · 2026-11-19
//
// Replaces two native <input type="date">, which look different on every
// system and ignore the theme (owner feedback 2026-11-19). One button shows
// the range ("Todas las fechas", "Últimos 7 días", "12–19 nov 2026") and
// opens a non-modal dialog with presets (Hoy, Últimos 7 días, Últimos 30
// días, Personalizado) and, for a custom range, a month calendar that
// follows the WAI-ARIA APG "Date Picker Dialog" grid: arrows move by day
// and week, Home/End to the week's first/last day, PageUp/PageDown by month
// (Shift: by year), Enter/Space pick. The first pick is the start, the
// second the end (earlier → swapped); Aplicar commits. Week starts on
// Monday in Spanish and on Sunday in English; names come from Intl in the
// page's language. Days after `max` (default today: logs have no future)
// cannot be picked. Values are local days "YYYY-MM-DD" ('' = open end).
// The dialog floats in the top layer (floating.ts, ADR-0064).
import { useId, useLayoutEffect, useMemo, useRef, useState, type CSSProperties, type KeyboardEvent } from 'react'
import { usePreferences } from '../../app/preferences-context'
import type { Lang } from '../../i18n/messages'
import { addDays, addMonths, DATE_PRESETS, fromISODay, matchPreset, monthGrid, presetRange, toISODay, weekStart, type DateRange } from './dates'
import { useAnchored, useDismiss, useTopLayer } from './floating'
import { CalendarIcon, ChevronLeftIcon, ChevronRightIcon } from './icons'

// ── Component ─────────────────────────────────────────────────────────────

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

export function DateRangePicker({ value, onChange, max, today: todayProp, ...a11y }: Props) {
  const { t, lang } = usePreferences()
  const dialogId = useId()
  const [open, setOpen] = useState(false)
  const trigger = useRef<HTMLButtonElement | null>(null)
  const panelRef = useRef<HTMLDivElement | null>(null)
  const { setReference, setFloating, floatingStyles } = useAnchored(open, { gap: 4, fitHeight: false })
  // Read once per mount (render must stay pure); the picker is short-lived.
  const [now] = useState(() => toISODay(new Date()))
  const today = todayProp ?? now
  const preset = matchPreset(value, today)

  const close = (refocus: boolean) => {
    setOpen(false)
    if (refocus) trigger.current?.focus()
  }
  useDismiss(open, [panelRef, trigger], (why) => close(why === 'escape'))

  const fmt = useMemo(() => new Intl.DateTimeFormat(lang, { day: 'numeric', month: 'short', year: 'numeric' }), [lang])
  let label: string
  if (!value.from && !value.to) label = t('kit.date.any')
  else if (preset) label = t(`kit.date.${preset}`)
  else if (value.from && value.to) label = fmt.formatRange(fromISODay(value.from), fromISODay(value.to))
  else label = value.from ? `${fmt.format(fromISODay(value.from))} –` : `– ${fmt.format(fromISODay(value.to))}`

  return (
    <>
      <button
        ref={(el) => {
          trigger.current = el
          setReference(el)
        }}
        type="button"
        className="k-input k-select k-drp__trigger"
        aria-haspopup="dialog"
        aria-expanded={open}
        aria-controls={open ? dialogId : undefined}
        onClick={() => (open ? close(false) : setOpen(true))}
        {...a11y}
      >
        <span className="k-drp__icon" aria-hidden="true">
          <CalendarIcon />
        </span>
        <span className={value.from || value.to ? 'k-select__value' : 'k-select__value k-select__placeholder'}>{label}</span>
      </button>
      {open && (
        <RangeDialog
          id={dialogId}
          panelRef={panelRef}
          setFloating={setFloating}
          style={floatingStyles}
          value={value}
          today={today}
          max={max ?? today}
          lang={lang}
          onPick={(v) => {
            onChange(v)
            close(true)
          }}
        />
      )}
    </>
  )
}

function RangeDialog({
  id,
  panelRef,
  setFloating,
  style,
  value,
  today,
  max,
  lang,
  onPick,
}: {
  id: string
  panelRef: { current: HTMLDivElement | null }
  setFloating: (el: HTMLElement | null) => void
  style: CSSProperties
  value: DateRange
  today: string
  max: string
  lang: Lang
  onPick: (v: DateRange) => void
}) {
  const { t } = usePreferences()
  const preset = matchPreset(value, today)
  const [custom, setCustom] = useState(!preset && !!(value.from || value.to))
  const [draft, setDraft] = useState<DateRange>(preset ? { from: '', to: '' } : value)
  const [focusDay, setFocusDay] = useState(value.to || value.from || today)
  // Bumped by keyboard moves and "Personalizado": the focused day follows.
  const [focusTick, setFocusTick] = useState(0)
  const titleId = `${id}-title`
  const start = weekStart(lang)

  useTopLayer(panelRef)
  // On open: focus the chosen preset (or the calendar's day in custom mode).
  useLayoutEffect(() => {
    const root = panelRef.current
    const target = root?.querySelector<HTMLElement>('[data-autofocus]') ?? root?.querySelector<HTMLElement>('button')
    target?.focus()
    // Only on mount.
  }, [panelRef])
  useLayoutEffect(() => {
    if (focusTick === 0) return
    panelRef.current?.querySelector<HTMLElement>(`[data-day="${focusDay}"]`)?.focus()
  }, [focusDay, focusTick, panelRef])

  const weeks = monthGrid(focusDay, start)
  const monthName = new Intl.DateTimeFormat(lang, { month: 'long', year: 'numeric' }).format(fromISODay(focusDay))
  const dayName = new Intl.DateTimeFormat(lang, { dateStyle: 'full' })
  const short = new Intl.DateTimeFormat(lang, { day: 'numeric', month: 'short' })
  const weekdays = Array.from({ length: 7 }, (_, i) => {
    // 2024-01-07 is a Sunday.
    const d = new Date(2024, 0, 7 + ((i + start) % 7))
    return {
      short: new Intl.DateTimeFormat(lang, { weekday: 'narrow' }).format(d),
      long: new Intl.DateTimeFormat(lang, { weekday: 'long' }).format(d),
    }
  })

  const move = (to: string) => {
    setFocusDay(to > max ? max : to)
    setFocusTick((n) => n + 1)
  }
  const pick = (day: string) => {
    if (day > max) return
    if (!draft.from || draft.to) setDraft({ from: day, to: '' })
    else setDraft(day < draft.from ? { from: day, to: draft.from } : { from: draft.from, to: day })
  }
  const onDayKey = (e: KeyboardEvent<HTMLButtonElement>, day: string) => {
    const dow = (fromISODay(day).getDay() - start + 7) % 7
    switch (e.key) {
      case 'ArrowLeft':
        move(addDays(day, -1))
        break
      case 'ArrowRight':
        move(addDays(day, 1))
        break
      case 'ArrowUp':
        move(addDays(day, -7))
        break
      case 'ArrowDown':
        move(addDays(day, 7))
        break
      case 'Home':
        move(addDays(day, -dow))
        break
      case 'End':
        move(addDays(day, 6 - dow))
        break
      case 'PageUp':
        move(addMonths(day, e.shiftKey ? -12 : -1))
        break
      case 'PageDown':
        move(addMonths(day, e.shiftKey ? 12 : 1))
        break
      default:
        return
    }
    e.preventDefault()
  }

  const inRange = (d: string) => !!draft.from && !!draft.to && d >= draft.from && d <= draft.to
  const status = !draft.from
    ? t('kit.date.pickStart')
    : !draft.to
      ? t('kit.date.pickEnd')
      : t('kit.date.chosen', { from: short.format(fromISODay(draft.from)), to: short.format(fromISODay(draft.to)) })

  return (
    <div
      ref={(el) => {
        panelRef.current = el
        setFloating(el)
      }}
      id={id}
      role="dialog"
      aria-label={t('kit.date.dialog')}
      popover="manual"
      className="k-drp k-pop"
      style={style}
    >
      <div className="k-drp__presets" role="group" aria-label={t('kit.date.presets')}>
        {DATE_PRESETS.map((p) => (
          <button
            key={p}
            type="button"
            className="k-drp__preset"
            aria-pressed={!custom && preset === p}
            data-autofocus={(!custom && preset === p) || undefined}
            onClick={() => onPick(presetRange(p, today))}
          >
            {t(`kit.date.${p}`)}
          </button>
        ))}
        <button
          type="button"
          className="k-drp__preset"
          aria-pressed={custom}
          aria-expanded={custom}
          onClick={() => {
            setCustom(true)
            setFocusTick((n) => n + 1)
          }}
        >
          {t('kit.date.custom')}
        </button>
      </div>

      {custom && (
        <div className="k-drp__cal">
          <div className="k-drp__month">
            <button type="button" className="k-icon-btn k-icon-btn--sm" aria-label={t('kit.date.prevMonth')} onClick={() => move(addMonths(focusDay, -1))}>
              <ChevronLeftIcon />
            </button>
            <h3 id={titleId} className="k-drp__title" aria-live="polite">
              {monthName}
            </h3>
            <button
              type="button"
              className="k-icon-btn k-icon-btn--sm"
              aria-label={t('kit.date.nextMonth')}
              disabled={addMonths(focusDay, 1).slice(0, 7) > max.slice(0, 7)}
              onClick={() => move(addMonths(focusDay, 1))}
            >
              <ChevronRightIcon />
            </button>
          </div>
          <table role="grid" aria-labelledby={titleId} className="k-drp__grid">
            <thead>
              <tr>
                {weekdays.map((w) => (
                  <th key={w.long} scope="col" abbr={w.long}>
                    {w.short}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {weeks.map((week, wi) => (
                <tr key={wi}>
                  {week.map((day, di) =>
                    day === '' ? (
                      <td key={`e${di}`} />
                    ) : (
                      <td key={day} aria-selected={day === draft.from || day === draft.to || inRange(day)}>
                        <button
                          type="button"
                          className="k-drp__day"
                          data-day={day}
                          data-start={day === draft.from || undefined}
                          data-end={day === draft.to || undefined}
                          data-range={inRange(day) || undefined}
                          data-autofocus={(custom && day === focusDay) || undefined}
                          tabIndex={day === focusDay ? 0 : -1}
                          aria-label={dayName.format(fromISODay(day))}
                          aria-current={day === today ? 'date' : undefined}
                          aria-disabled={day > max || undefined}
                          onClick={() => {
                            setFocusDay(day)
                            pick(day)
                          }}
                          onKeyDown={(e) => {
                            if (e.key === 'Enter' || e.key === ' ') {
                              e.preventDefault()
                              pick(day)
                              return
                            }
                            onDayKey(e, day)
                          }}
                        >
                          {fromISODay(day).getDate()}
                        </button>
                      </td>
                    ),
                  )}
                </tr>
              ))}
            </tbody>
          </table>
          <p className="k-drp__status" aria-live="polite">
            {status}
          </p>
          <p className="k-drp__keys">{t('kit.date.keys')}</p>
        </div>
      )}

      <div className="k-drp__foot">
        {(value.from || value.to || draft.from) && (
          <button type="button" className="k-btn k-btn--text k-btn--sm" onClick={() => onPick({ from: '', to: '' })}>
            <span className="k-btn__label">{t('kit.date.clear')}</span>
          </button>
        )}
        {custom && (
          <button type="button" className="k-btn k-btn--filled k-btn--sm" disabled={!draft.from || !draft.to} onClick={() => onPick(draft)}>
            <span className="k-btn__label">{t('kit.date.apply')}</span>
          </button>
        )}
      </div>
    </div>
  )
}
