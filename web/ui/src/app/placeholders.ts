// Kongtrol UI — sections still served by the classic dashboard · v0.2.0 · 2026-11-18
//
// One entry per route that phase 2 still has to build (docs/dashboard-parity.md
// lists every row). Delete the entry when its real page lands.
import { LEGACY } from '../links'
import type { PlaceholderSpec } from '../pages/PlaceholderPage'

const rulesTabs: PlaceholderSpec['tabs'] = {
  label: 'nav.rules',
  items: [
    { to: '/rules', label: 'page.rules.tab.policies' },
    { to: '/rules/schedule', label: 'page.rules.tab.schedule' },
  ],
}

export const PLACEHOLDERS: Record<string, PlaceholderSpec> = {
  '/rules': {
    title: 'nav.rules',
    lead: 'page.rules.lead',
    items: ['page.rules.item1', 'page.rules.item2', 'page.rules.item3'],
    legacy: LEGACY.studio,
    tabs: rulesTabs,
  },
  '/rules/schedule': {
    title: 'nav.rules',
    lead: 'page.rules.lead',
    items: ['page.schedule.item1', 'page.schedule.item2'],
    legacy: LEGACY.settings,
    tabs: rulesTabs,
  },
}

/** Router basename: Vite's base without the trailing slash ("/app" in builds). */
export const BASENAME = import.meta.env.BASE_URL.replace(/\/$/, '')
