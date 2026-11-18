// Kongtrol UI external links · v0.5.0 · 2026-11-18
//
// The public site is https://kongtrol.app (kongtrol.dev only redirects, never
// link to it). Change the site or guide location here, nowhere else.

const SITE_URL = 'https://kongtrol.app'
export const GUIDE_URL = `${SITE_URL}/guia`

/** Section anchors on the guide (ids from landing/src/content/guideSections.ts). */
export type GuideSection = 'quickstart' | 'install' | 'connect' | 'dashboard' | 'trouble'

export function guideLink(section?: GuideSection): string {
  return section ? `${GUIDE_URL}#${section}` : GUIDE_URL
}

/** Legal and policy pages, in the reader's language (EN uses localized slugs under /en/). */
export type SitePage = 'guide' | 'privacy' | 'security'

const SITE_PATHS: Record<SitePage, { es: string; en: string }> = {
  guide: { es: '/guia', en: '/en/guide' },
  privacy: { es: '/privacidad', en: '/en/privacy' },
  security: { es: '/seguridad', en: '/en/security' },
}

export function siteLink(page: SitePage, lang: 'es' | 'en'): string {
  return `${SITE_URL}${SITE_PATHS[page][lang]}`
}
