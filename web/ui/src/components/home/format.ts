// Kongtrol UI — Inicio number/time formatting · v1.0.0 · 2026-11-18
//
// Locale-aware (Intl) versions of the legacy formatters (app.js
// formatBytes / formatDuration), with a narrow no-break space between number
// and unit so a value never wraps in a cell.
import type { Lang } from '../../i18n/messages'

const NNBSP = ' '
const UNITS = ['B', 'KB', 'MB', 'GB', 'TB'] as const

/** 1536 → "1,5 KB" (es) / "1.5 KB" (en); null for 0 or less (the cell shows —). */
export function formatBytes(bytes: number, lang: Lang): string | null {
  if (!bytes || bytes <= 0) return null
  const i = Math.min(UNITS.length - 1, Math.floor(Math.log(bytes) / Math.log(1024)))
  const value = bytes / 1024 ** i
  const n = new Intl.NumberFormat(lang, { maximumFractionDigits: i === 0 ? 0 : 1 }).format(value)
  return `${n}${NNBSP}${UNITS[i]}`
}

/** Seconds → "< 1s", "42s", "3m 05s", "2h 07m" (legacy app.js formatDuration). */
export function formatDuration(seconds: number): string {
  if (!seconds || seconds < 1) return '< 1s'
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = Math.floor(seconds % 60)
  const pad = (v: number) => String(v).padStart(2, '0')
  if (h > 0) return `${h}h${NNBSP}${pad(m)}m`
  if (m > 0) return `${m}m${NNBSP}${pad(s)}s`
  return `${s}s`
}

/** ISO time → local clock time in the page's language; '' if missing or invalid. */
export function formatClock(iso: string | undefined, lang: Lang): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime()) || d.getFullYear() < 2000) return ''
  return d.toLocaleTimeString(lang, { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}
