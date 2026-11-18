// Horarios — create / edit one schedule, inline · v1.0.0 · 2026-11-18
//
// Days as toggles (not "mon,tue" text), VPNs as checkboxes of existing
// profiles (not free text), native time inputs (always 24 h HH:MM values)
// and a live sentence that says exactly what Kongtrol will do.
import { useId, useRef, useState, type FormEvent } from 'react'
import { Link } from 'react-router'
import { ApiError } from '../../api/client'
import { usePreferences } from '../../app/preferences-context'
import { useSaveSchedule } from '../../data/rules'
import type { MessageKey } from '../../i18n/messages'
import { Button } from '../ui/Button'
import { Card } from '../ui/Card'
import { TextField, type FieldError } from '../ui/Field'
import { ErrorState } from '../ui/States'
import { ClockIcon } from './icons'
import {
  daysShape,
  hasErrors,
  isOvernight,
  SCHEDULE_FIELDS,
  scheduleToPayload,
  validateSchedule,
  WEEKDAYS,
  type ScheduleDraft,
  type ScheduleErrors,
  type ScheduleRule,
  type Weekday,
} from './model'
import { dayName, describeSchedule } from './preview'

interface Props {
  original?: string | undefined
  initial: ScheduleDraft
  rules: readonly ScheduleRule[]
  profiles: readonly string[]
  onClose: () => void
}

const PRESETS: Record<'weekdays' | 'all' | 'weekend', Weekday[]> = {
  weekdays: ['mon', 'tue', 'wed', 'thu', 'fri'],
  all: [...WEEKDAYS],
  weekend: ['sat', 'sun'],
}

export function ScheduleEditor({ original, initial, rules, profiles, onClose }: Props) {
  const { t, lang } = usePreferences()
  const save = useSaveSchedule()
  const formRef = useRef<HTMLFormElement>(null)
  const [draft, setDraft] = useState<ScheduleDraft>(initial)
  const [submitted, setSubmitted] = useState(false)
  const [serverError, setServerError] = useState<unknown>(null)
  const id = useId()

  const ctx = { names: rules.map((r) => r.name), editing: original, profiles }
  const errors: ScheduleErrors = submitted ? validateSchedule(draft, ctx) : {}
  const err = (f: keyof ScheduleErrors): FieldError | null => {
    const e = errors[f]
    if (!e) return null
    const key = `schedule.err.${f}.${e.code}`
    return { message: t(key as MessageKey, { value: e.value ?? '' }), fix: t(`${key}.fix` as MessageKey) }
  }
  const set = <K extends keyof ScheduleDraft>(k: K, v: ScheduleDraft[K]) => setDraft((d) => ({ ...d, [k]: v }))
  const toggle = <V,>(list: readonly V[], v: V): V[] => (list.includes(v) ? list.filter((x) => x !== v) : [...list, v])

  const payload = scheduleToPayload(draft)
  const preview = draft.profiles.length > 0 ? describeSchedule(payload, t, lang) : t('schedule.preview.pickVpn')
  const overnight = !draft.allDay && isOvernight(draft.start, draft.end) && daysShape(draft.days).kind !== 'all'

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setSubmitted(true)
    setServerError(null)
    const found = validateSchedule(draft, ctx)
    if (hasErrors(found)) {
      const first = SCHEDULE_FIELDS.find((f) => found[f])
      formRef.current?.querySelector<HTMLElement>(`[name="${first}"]`)?.focus()
      return
    }
    try {
      await save.mutateAsync({ original, payload })
      onClose()
    } catch (error) {
      setServerError(error)
    }
  }
  const conflict = serverError instanceof ApiError && serverError.status === 409
  const profilesErr = err('profiles')

  return (
    <Card
      className="r-editor"
      title={
        <span id={`${id}-t`} tabIndex={-1} data-autofocus>
          {original ? t('schedule.editor.editTitle', { name: original }) : t('schedule.editor.newTitle')}
        </span>
      }
    >
      <form ref={formRef} className="r-form" noValidate onSubmit={onSubmit} aria-labelledby={`${id}-t`}>
        <TextField
          name="name"
          label={t('schedule.field.name')}
          placeholder={t('schedule.field.name.placeholder')}
          value={draft.name}
          onChange={(e) => set('name', e.target.value)}
          error={err('name') ?? (conflict ? { message: t('schedule.err.name.duplicate', { value: draft.name }), fix: t('schedule.err.name.duplicate.fix') } : null)}
          required
          autoComplete="off"
        />

        <fieldset className="r-choice" aria-describedby={profilesErr ? `${id}-p-err` : undefined}>
          <legend className="k-field__label">{t('schedule.field.profiles')}</legend>
          {profiles.length === 0 ? (
            <p className="k-field__error" id={`${id}-p-err`}>
              <strong>{t('schedule.err.profiles.noProfiles')}</strong>{' '}
              <Link to="/profiles">{t('schedule.err.profiles.noProfiles.fix')}</Link>
            </p>
          ) : (
            <>
              <div className="r-checks">
                {profiles.map((p, i) => (
                  <label key={p} className="r-check">
                    <input
                      type="checkbox"
                      name={i === 0 ? 'profiles' : undefined}
                      checked={draft.profiles.includes(p)}
                      onChange={() => set('profiles', toggle(draft.profiles, p))}
                    />
                    <span className="mono">{p}</span>
                  </label>
                ))}
              </div>
              {profilesErr && (
                <p className="k-field__error" id={`${id}-p-err`}>
                  <strong>{profilesErr.message}</strong> <span>{profilesErr.fix}</span>
                </p>
              )}
            </>
          )}
        </fieldset>

        <fieldset className="r-choice">
          <legend className="k-field__label">{t('schedule.field.days')}</legend>
          <div className="r-days">
            {WEEKDAYS.map((d) => (
              <label key={d} className="r-day">
                <input
                  type="checkbox"
                  checked={draft.days.includes(d)}
                  onChange={() => set('days', WEEKDAYS.filter((x) => toggle(draft.days, d).includes(x)))}
                />
                <span aria-hidden="true">{t(`schedule.day.${d}.short` as MessageKey)}</span>
                <span className="visually-hidden">{dayName(d, t)}</span>
              </label>
            ))}
          </div>
          <div className="r-presets">
            {(Object.keys(PRESETS) as (keyof typeof PRESETS)[]).map((k) => (
              <Button key={k} variant="text" size="sm" onClick={() => set('days', PRESETS[k])}>
                {t(`schedule.preset.${k}` as MessageKey)}
              </Button>
            ))}
          </div>
          <p className="k-field__hint">{t('schedule.field.days.hint')}</p>
        </fieldset>

        <fieldset className="r-choice">
          <legend className="k-field__label">{t('schedule.field.hours')}</legend>
          <label className="r-check">
            <input type="checkbox" checked={draft.allDay} onChange={(e) => set('allDay', e.target.checked)} />
            <span>{t('schedule.field.allDay')}</span>
          </label>
          {!draft.allDay && (
            <div className="r-times">
              <TextField
                name="start"
                type="time"
                step={60}
                label={t('schedule.field.start')}
                value={draft.start}
                onChange={(e) => set('start', e.target.value)}
                error={err('start')}
                mono
              />
              <TextField
                name="end"
                type="time"
                step={60}
                label={t('schedule.field.end')}
                value={draft.end}
                onChange={(e) => set('end', e.target.value)}
                error={err('end')}
                mono
              />
            </div>
          )}
          {overnight && <p className="k-field__hint">{t('schedule.field.overnight')}</p>}
        </fieldset>

        <div className="r-preview" aria-live="polite">
          <ClockIcon />
          <p>
            <span className="r-preview__label">{t('schedule.preview.label')}</span>
            {preview}
          </p>
        </div>

        {serverError !== null && !conflict && <ErrorState error={serverError} />}

        <div className="r-form__actions">
          <Button type="submit" loading={save.isPending}>
            {original ? t('rules.action.saveChanges') : t('schedule.action.create')}
          </Button>
          <Button variant="text" onClick={onClose}>
            {t('rules.action.cancel')}
          </Button>
        </div>
      </form>
    </Card>
  )
}
