// Kongtrol UI external and legacy links · v0.2.0 · 2026-11-18
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

/** Legacy dashboard pages (web/dashboard, served at /) used until migrated. */
export const LEGACY = {
  home: '/',
  studio: '/studio.html',
  security: '/security.html',
  settings: '/settings.html',
  audit: '/audit.html',
} as const
