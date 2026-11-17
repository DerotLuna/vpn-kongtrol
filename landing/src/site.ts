// Site-wide constants and routes — version 2.4.0 · 2026-11-17.
//
// Every page exists once per language, each at its own URL (Spanish at the
// root, English under /en/), as Google recommends for multilingual sites:
// https://developers.google.com/search/docs/specialty/international/managing-multi-regional-sites
// Slugs are localized (/guia vs /en/guide) because Google recommends readable,
// language-appropriate words in URLs:
// https://developers.google.com/search/docs/crawling-indexing/url-structure
import type { Lang } from './i18n'

export const SITE_URL = 'https://kongtrol.app'
export const SITE_VERSION = 'v2.3.0'
export const SITE_UPDATED = '2026-11-17'

export type PageId = 'home' | 'guia' | 'terminos' | 'privacidad' | 'seguridad'

export const PATHS: Record<PageId, Record<Lang, string>> = {
  home: { es: '/', en: '/en/' },
  guia: { es: '/guia', en: '/en/guide' },
  terminos: { es: '/terminos', en: '/en/terms' },
  privacidad: { es: '/privacidad', en: '/en/privacy' },
  seguridad: { es: '/seguridad', en: '/en/security' },
}

export const pathFor = (page: PageId, lang: Lang) => PATHS[page][lang]
export const urlFor = (page: PageId, lang: Lang) => SITE_URL + PATHS[page][lang]

// The language of the page being rendered comes from <html lang>, which each
// HTML entry sets (index.html = es, en/index.html = en, ...).
export function pageLang(): Lang {
  if (typeof document === 'undefined') return 'es'
  return document.documentElement.lang.toLowerCase().startsWith('en') ? 'en' : 'es'
}
