// Language + theme for the whole UI. A choice is kept in this browser and
// in ~/.kongtrol/preferences.json via /api/v1/preferences, which is
// authoritative once loaded.
import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { api, setApiLang } from '../api/client'
import { browserLang, initialLang, isLang, storeLang, storedLang } from '../i18n/lang'
import { format, messages, type Lang, type MessageKey } from '../i18n/messages'
import {
  applyTheme,
  initialTheme,
  isTheme,
  storeTheme,
  storedTheme,
  systemTheme,
  watchSystemTheme,
  type Theme,
} from '../theme/theme'
import { PreferencesContext, type PreferencesValue } from './preferences-context'

export function PreferencesProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<Lang>(initialLang)
  const [theme, setThemeState] = useState<Theme>(initialTheme)

  useEffect(() => {
    document.documentElement.lang = lang
    setApiLang(lang)
  }, [lang])

  useEffect(() => applyTheme(theme), [theme])

  // Follow the system / browser while there is no explicit choice.
  useEffect(
    () =>
      watchSystemTheme((next) => {
        if (!storedTheme()) setThemeState(next)
      }),
    [],
  )
  useEffect(() => {
    const onLanguageChange = () => {
      if (!storedLang()) setLangState(browserLang())
    }
    window.addEventListener('languagechange', onLanguageChange)
    return () => window.removeEventListener('languagechange', onLanguageChange)
  }, [])

  // Server copy wins: empty there clears a stale browser copy.
  useEffect(() => {
    let cancelled = false
    api.preferences().then(
      (prefs) => {
        if (cancelled) return
        const chosenLang = isLang(prefs.language) ? prefs.language : ''
        const chosenTheme = isTheme(prefs.theme) ? prefs.theme : ''
        storeLang(chosenLang)
        storeTheme(chosenTheme)
        setLangState(chosenLang || browserLang())
        setThemeState(chosenTheme || systemTheme())
      },
      () => {
        // Daemon unreachable: keep the local resolution; pages show the error.
      },
    )
    return () => {
      cancelled = true
    }
  }, [])

  const setLang = useCallback((next: Lang) => {
    storeLang(next)
    setLangState(next)
    api.savePreferences({ language: next }).catch(() => {})
  }, [])

  const toggleTheme = useCallback(() => {
    const next: Theme = theme === 'dark' ? 'light' : 'dark'
    // Choosing what the system already uses means "follow the system".
    const explicit = next === systemTheme() ? '' : next
    storeTheme(explicit)
    setThemeState(next)
    api.savePreferences({ theme: explicit }).catch(() => {})
  }, [theme])

  const t = useCallback(
    (key: MessageKey, vars?: Record<string, string | number>) => format(messages[lang][key], vars),
    [lang],
  )

  const value = useMemo<PreferencesValue>(
    () => ({ lang, setLang, t, theme, toggleTheme }),
    [lang, setLang, t, theme, toggleTheme],
  )

  return <PreferencesContext.Provider value={value}>{children}</PreferencesContext.Provider>
}
