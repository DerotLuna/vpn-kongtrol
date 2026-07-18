export type Lang = 'es' | 'en'

export function detectLang(): Lang {
  if (typeof navigator === 'undefined') return 'en'
  return navigator.language.toLowerCase().startsWith('es') ? 'es' : 'en'
}
