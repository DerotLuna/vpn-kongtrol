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
  '/traffic': {
    title: 'nav.traffic',
    lead: 'page.traffic.lead',
    items: ['page.traffic.item1', 'page.traffic.item2', 'page.traffic.item3'],
    legacy: LEGACY.home,
  },
  '/security': {
    title: 'nav.security',
    lead: 'page.security.lead',
    items: ['page.security.item1', 'page.security.item2', 'page.security.item3', 'page.security.item4'],
    legacy: LEGACY.security,
  },
  '/activity': {
    title: 'nav.activity',
    lead: 'page.activity.lead',
    items: ['page.activity.item1', 'page.activity.item2', 'page.activity.item3'],
    legacy: LEGACY.audit,
  },
  '/settings': {
    title: 'nav.settings',
    lead: 'page.settings.lead',
    items: ['page.settings.item1', 'page.settings.item2', 'page.settings.item3'],
    legacy: LEGACY.settings,
  },
}

/** Router basename: Vite's base without the trailing slash ("/app" in builds). */
export const BASENAME = import.meta.env.BASE_URL.replace(/\/$/, '')
