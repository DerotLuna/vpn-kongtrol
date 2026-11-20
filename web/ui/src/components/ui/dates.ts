// Kongtrol UI kit — local-day helpers for the DateRangePicker · v2.0.0 · 2026-11-19
//
// Days are local calendar days as "YYYY-MM-DD" (what the audit API takes as
// from / to). The calendar itself (grid, week start, month moves) is React
// Aria's RangeCalendar (ADR-0068); these only name the presets and the
// trigger's label. Pure functions, tested in kit.test.tsx.

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
