// Horarios — the plain-language sentence for a schedule · v1.0.0 · 2026-11-18
//
// "De lunes a viernes, de 9:00 a 18:00: conecta «oficina». Fuera de ese
// horario, Kongtrol la desconecta." Built from i18n keys + Intl.ListFormat,
// so ES and EN read naturally; tested in model.test.ts.
import type { FieldError } from '../ui/Field'
import { messages, type Lang, type MessageKey } from '../../i18n/messages'
import { daysShape, isOvernight, readableTime, type ExplainResult, type Overlap, type PolicyError, type ScheduleRule, type Weekday } from './model'

type T = (key: MessageKey, vars?: Record<string, string | number>) => string

export function listOf(items: readonly string[], lang: Lang): string {
  return new Intl.ListFormat(lang, { type: 'conjunction', style: 'long' }).format(items)
}

function capitalize(s: string): string {
  return s.charAt(0).toLocaleUpperCase() + s.slice(1)
}

export function dayName(d: Weekday, t: T): string {
  return t(`schedule.day.${d}` as MessageKey)
}

export function describeDays(weekdays: readonly string[] | null | undefined, t: T, lang: Lang): string {
  const shape = daysShape(weekdays)
  switch (shape.kind) {
    case 'all':
      return t('schedule.preview.days.all')
    case 'weekdays':
      return t('schedule.preview.days.weekdays')
    case 'weekend':
      return t('schedule.preview.days.weekend')
    case 'range':
      return t('schedule.preview.days.range', { from: dayName(shape.from, t), to: dayName(shape.to, t) })
    case 'list':
      return capitalize(listOf(shape.days.map((d) => dayName(d, t)), lang))
  }
}

export function describeHours(start: string | undefined, end: string | undefined, t: T): string {
  if (!start || !end) return t('schedule.preview.hours.allDay')
  const vars = { start: readableTime(start), end: readableTime(end) }
  return isOvernight(start, end) ? t('schedule.preview.hours.overnight', vars) : t('schedule.preview.hours.range', vars)
}

/** The whole sentence for a saved rule or the editor's draft. */
export function describeSchedule(r: ScheduleRule, t: T, lang: Lang): string {
  const profiles = r.profiles ?? []
  const names = listOf(profiles.map((p) => t('rules.quote', { name: p })), lang)
  const many = profiles.length > 1
  const always = daysShape(r.weekdays).kind === 'all' && (!r.start || !r.end)
  if (always) return t(many ? 'schedule.preview.always.many' : 'schedule.preview.always.one', { profiles: names })
  const head = t('schedule.preview.connect', {
    days: describeDays(r.weekdays, t, lang),
    hours: describeHours(r.start, r.end, t),
    profiles: names,
  })
  return `${head} ${t(many ? 'schedule.preview.outside.many' : 'schedule.preview.outside.one')}`
}

// ── Rules texts ───────────────────────────────────────────────────────────

/** The daemon's reason code, translated; an unknown code falls back to its English text. */
export function reasonText(ex: ExplainResult, t: T): string {
  const key = `rules.reason.${ex.ReasonCode}`
  // A reason code newer than this page: show the daemon's English text.
  if (!(key in messages.es)) return ex.Reason
  return t(key as MessageKey, { value: ex.ReasonArgs?.[0] ?? '' })
}

export function overlapText(o: Overlap, t: T): string {
  return t(`rules.overlap.${o.kind}` as MessageKey, {
    value: o.value,
    winner: o.winner.policy,
    winnerEntry: o.winner.entry,
    via: o.winner.via,
    other: o.other.policy,
    otherEntry: o.other.entry,
    otherVia: o.other.via,
  })
}

export function policyErrorText(field: string, e: PolicyError, t: T): FieldError {
  const vars = { value: e.value ?? '', line: e.line ?? 0 }
  const key = `rules.err.${field}.${e.code}` as MessageKey
  return { message: t(key, vars), fix: t(`${key}.fix` as MessageKey, vars) }
}

