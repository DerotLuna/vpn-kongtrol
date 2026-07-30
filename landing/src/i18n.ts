export type Lang = 'es' | 'en'

// Only Spanish gets a dedicated experience; every other browser language
// (including unset/unsupported) falls back to English.
export function detectLang(): Lang {
  if (typeof navigator === 'undefined') return 'en'
  const langs = navigator.languages && navigator.languages.length > 0
    ? navigator.languages
    : [navigator.language]
  return langs.some(l => l?.toLowerCase().startsWith('es')) ? 'es' : 'en'
}
