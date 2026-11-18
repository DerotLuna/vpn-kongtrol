import { useCallback, useEffect, useState, useSyncExternalStore } from 'react'
import { detectLang, Lang } from '../i18n'
import { Theme } from '../theme'
import { PageId, pathFor } from '../site'

// Preference rule (same as the dashboard): follow the browser/OS until the
// visitor explicitly picks; an explicit pick is stored and wins from then on.
// Nothing is written to storage unless the visitor clicks something, so a
// stored value always means "the visitor chose this".
const LANG_KEY = 'kglang'
const THEME_KEY = 'kgtheme'

function read(key: string): string | null {
  try { return localStorage.getItem(key) } catch { return null }
}
function write(key: string, value: string) {
  try { localStorage.setItem(key, value) } catch { /* private mode: choice lasts this page only */ }
}

const darkQuery = () => (typeof window !== 'undefined' && window.matchMedia ? window.matchMedia('(prefers-color-scheme: dark)') : null)
const systemTheme = (): Theme => (darkQuery()?.matches ? 'dark' : 'light')
const storedTheme = (): Theme | null => {
  const v = read(THEME_KEY)
  return v === 'light' || v === 'dark' ? v : null
}

// The theme lives on <html data-theme>, not in React state. The inline boot
// script in <head> (THEME_BOOT in src/seo.ts) sets it before first paint from
// the stored pick or the OS; React reads it with useSyncExternalStore and only
// writes it on an explicit pick or a live OS change. The prerender has no
// document, so the server snapshot is the default 'light': hydrateRoot renders
// that first (identical to the HTML, no mismatch) and then re-renders with the
// real value. The page colour never flashes: nothing writes 'light' back.
// https://react.dev/reference/react/useSyncExternalStore#adding-support-for-server-rendering
const themeListeners = new Set<() => void>()
const html = () => document.documentElement
const applyTheme = (t: Theme) => {
  html().setAttribute('data-theme', t)
  themeListeners.forEach(l => l())
}
const themeSnapshot = (): Theme => {
  const t = html().getAttribute('data-theme')
  return t === 'light' || t === 'dark' ? t : systemTheme()
}
const themeServerSnapshot = (): Theme => 'light'
function subscribeTheme(onChange: () => void) {
  // The boot script skips data-theme when storage throws; set it here then.
  if (!html().hasAttribute('data-theme')) html().setAttribute('data-theme', themeSnapshot())
  themeListeners.add(onChange)
  // Live system changes apply while the visitor has not picked a theme.
  const q = darkQuery()
  const onSystem = () => { if (q && !storedTheme()) applyTheme(q.matches ? 'dark' : 'light') }
  q?.addEventListener('change', onSystem)
  return () => {
    themeListeners.delete(onChange)
    q?.removeEventListener('change', onSystem)
  }
}

export function useTheme() {
  const theme = useSyncExternalStore(subscribeTheme, themeSnapshot, themeServerSnapshot)

  const setTheme = useCallback((next: Theme) => {
    write(THEME_KEY, next)
    applyTheme(next)
  }, [])

  return { theme, setTheme }
}

const storedLang = (): Lang | null => {
  const v = read(LANG_KEY)
  return v === 'es' || v === 'en' ? v : null
}

// Records an explicit language pick (language toggle, suggestion banner).
export const chooseLang = (lang: Lang) => write(LANG_KEY, lang)

// Language is decided by the URL. Client-side only, never for crawlers (they
// keep no storage): a visitor who explicitly picked the other language is sent
// to that URL; a visitor who never picked gets a dismissible suggestion when
// the browser language differs. No automatic redirect on a guess — Google:
// "Avoid automatically redirecting users from one language version of a site
// to a different language version" (managing-multi-regional-sites).
export function useLangRouting(page: PageId, lang: Lang) {
  const [suggest, setSuggest] = useState<Lang | null>(null)

  useEffect(() => {
    const chosen = storedLang()
    if (chosen && chosen !== lang) {
      window.location.replace(pathFor(page, chosen) + window.location.hash)
      return
    }
    if (!chosen && detectLang() !== lang) setSuggest(detectLang())
  }, [page, lang])

  const dismiss = useCallback(() => {
    chooseLang(lang) // "stay here" is an explicit pick too
    setSuggest(null)
  }, [lang])

  return { suggest, dismiss }
}
