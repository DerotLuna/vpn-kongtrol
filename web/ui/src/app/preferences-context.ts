import { createContext, useContext } from 'react'
import type { Lang, MessageKey } from '../i18n/messages'
import type { Theme } from '../theme/theme'

export interface PreferencesValue {
  lang: Lang
  setLang: (lang: Lang) => void
  t: (key: MessageKey, vars?: Record<string, string | number>) => string
  theme: Theme
  toggleTheme: () => void
}

export const PreferencesContext = createContext<PreferencesValue | null>(null)

export function usePreferences(): PreferencesValue {
  const value = useContext(PreferencesContext)
  if (!value) throw new Error('usePreferences must be used inside <PreferencesProvider>')
  return value
}
