// Theme resolution (same rules as web/dashboard/shell.js and CLAUDE.md):
// explicit choice wins and persists (localStorage + preferences.json);
// otherwise follow prefers-color-scheme live. Picking the theme the system
// already uses clears the explicit choice, so later OS switches still apply.
import { readStored, writeStored } from '../lib/storage'

export type Theme = 'light' | 'dark'

const THEME_KEY = 'kongtrol-dashboard-theme'
const DARK_QUERY = '(prefers-color-scheme: dark)'

export function isTheme(value: unknown): value is Theme {
  return value === 'light' || value === 'dark'
}

export function storedTheme(): Theme | '' {
  const saved = readStored(THEME_KEY)
  return isTheme(saved) ? saved : ''
}

export function storeTheme(theme: Theme | ''): void {
  writeStored(THEME_KEY, theme)
}

export function systemTheme(): Theme {
  return typeof window !== 'undefined' && window.matchMedia?.(DARK_QUERY).matches ? 'dark' : 'light'
}

export function initialTheme(): Theme {
  return storedTheme() || systemTheme()
}

export function applyTheme(theme: Theme): void {
  const root = document.documentElement
  root.dataset.theme = theme
  root.style.colorScheme = theme
}

/** Calls onChange when the OS/browser color scheme flips. Returns unsubscribe. */
export function watchSystemTheme(onChange: (theme: Theme) => void): () => void {
  const mq = window.matchMedia?.(DARK_QUERY)
  if (!mq) return () => {}
  const listener = () => onChange(mq.matches ? 'dark' : 'light')
  mq.addEventListener('change', listener)
  return () => mq.removeEventListener('change', listener)
}
