export type Theme = 'dark' | 'light'

// Only switch to dark when the browser explicitly reports it; no support,
// no preference, or a light preference all resolve to light.
export function detectTheme(): Theme {
  if (typeof window === 'undefined' || !window.matchMedia) return 'light'
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}
