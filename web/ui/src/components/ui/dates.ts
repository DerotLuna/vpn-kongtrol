// Kongtrol UI kit — local-day helpers for the DateRangePicker · v1.0.0 · 2026-11-19
//
// Days are local calendar days as "YYYY-MM-DD" (what the audit API takes as
// from / to). Pure functions, tested in kit.test.tsx.
import type { Lang } from '../../i18n/messages'

export interface DateRange {
  from: string
  to: string
}

export type DatePreset = 'today' | 'last7' | 'last30'
export const DATE_PRESETS: readonly DatePreset[] = ['today', 'last7', 'last30']

export function toISODay(d: Date): string {
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`
}

export function fromISODay(s: string): Date {
  const [y, m, d] = s.split('-').map(Number)
  return new Date(y!, (m ?? 1) - 1, d ?? 1)
}

export function addDays(s: string, n: number): string {
  const d = fromISODay(s)
  d.setDate(d.getDate() + n)
  return toISODay(d)
}

/** Same day n months later, clamped to the month's last day (31 Jan + 1 → 28/29 Feb). */
export function addMonths(s: string, n: number): string {
  const d = fromISODay(s)
  const day = d.getDate()
  const target = new Date(d.getFullYear(), d.getMonth() + n, 1)
  const last = new Date(target.getFullYear(), target.getMonth() + 1, 0).getDate()
  target.setDate(Math.min(day, last))
  return toISODay(target)
}

/** 0 = Sunday, 1 = Monday (Spanish locales start the week on Monday). */
export function weekStart(lang: Lang): 0 | 1 {
  return lang === 'es' ? 1 : 0
}

export function presetRange(p: DatePreset, today: string): DateRange {
  if (p === 'today') return { from: today, to: today }
  return { from: addDays(today, p === 'last7' ? -6 : -29), to: today }
}

export function matchPreset(v: DateRange, today: string): DatePreset | null {
  return DATE_PRESETS.find((p) => {
    const r = presetRange(p, today)
    return r.from === v.from && r.to === v.to
  }) ?? null
}

/** Weeks of a month as ISO days ('' = a cell of another month). */
export function monthGrid(month: string, start: 0 | 1): string[][] {
  const first = fromISODay(month.slice(0, 8) + '01')
  const lead = (first.getDay() - start + 7) % 7
  const days = new Date(first.getFullYear(), first.getMonth() + 1, 0).getDate()
  const cells: string[] = Array.from({ length: lead }, () => '')
  for (let d = 1; d <= days; d++) cells.push(toISODay(new Date(first.getFullYear(), first.getMonth(), d)))
  while (cells.length % 7 !== 0) cells.push('')
  const weeks: string[][] = []
  for (let i = 0; i < cells.length; i += 7) weeks.push(cells.slice(i, i + 7))
  return weeks
}
