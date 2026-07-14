import { useCallback, useEffect, useState } from 'react'
import { detectLang, Lang } from '../i18n'
import { detectTheme, Theme } from '../theme'

const LANG_KEY = 'kglang'
const THEME_KEY = 'kgtheme'

export function usePreferences() {
  const [lang, setLangState] = useState<Lang>(() => {
    if (typeof window === 'undefined') return 'en'
    const stored = localStorage.getItem(LANG_KEY)
    return stored === 'es' || stored === 'en' ? stored : detectLang()
  })

  const [theme, setThemeState] = useState<Theme>(() => {
    if (typeof window === 'undefined') return 'dark'
    const stored = localStorage.getItem(THEME_KEY)
    return stored === 'light' || stored === 'dark' ? stored : detectTheme()
  })

  const setLang = useCallback((next: Lang) => {
    localStorage.setItem(LANG_KEY, next)
    setLangState(next)
  }, [])

  const setTheme = useCallback((next: Theme) => {
    localStorage.setItem(THEME_KEY, next)
    setThemeState(next)
  }, [])

  useEffect(() => {
    document.documentElement.lang = lang
  }, [lang])

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme)
  }, [theme])

  return { lang, setLang, theme, setTheme }
}
