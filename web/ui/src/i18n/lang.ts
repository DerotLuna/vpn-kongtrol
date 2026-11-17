// Language resolution (same rules as web/dashboard/i18n.js and CLAUDE.md):
// explicit choice (toggle or `kongtrol config lang`, both saved in
// ~/.kongtrol/preferences.json) wins; otherwise the browser's first
// preferred language decides (es* -> es, anything else -> en).
import { readStored, writeStored } from '../lib/storage'
import type { Lang } from './messages'

const LANG_KEY = 'kongtrol-dashboard-lang'

export function isLang(value: unknown): value is Lang {
  return value === 'es' || value === 'en'
}

export function storedLang(): Lang | '' {
  const saved = readStored(LANG_KEY)
  return isLang(saved) ? saved : ''
}

export function storeLang(lang: Lang | ''): void {
  writeStored(LANG_KEY, lang)
}

export function browserLang(nav: Pick<Navigator, 'languages' | 'language'> = navigator): Lang {
  const first = nav.languages?.[0] ?? nav.language ?? ''
  return first.toLowerCase().startsWith('es') ? 'es' : 'en'
}

export function initialLang(): Lang {
  return storedLang() || browserLang()
}
