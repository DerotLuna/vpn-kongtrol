// Kongtrol UI — sections still served by the classic dashboard · v0.1.0 · 2026-11-18
//
// One entry per route that phase 2 still has to build (docs/dashboard-parity.md
// lists every row). Delete the entry when its real page lands.
import { LEGACY } from '../links'
import type { PlaceholderSpec } from '../pages/PlaceholderPage'

const profilesTabs: PlaceholderSpec['tabs'] = {
  label: 'nav.profiles',
  items: [
    { to: '/profiles', label: 'page.profiles.tab.vpns' },
    { to: '/profiles/groups', label: 'page.profiles.tab.groups' },
  ],
}
const rulesTabs: PlaceholderSpec['tabs'] = {
  label: 'nav.rules',
  items: [
    { to: '/rules', label: 'page.rules.tab.policies' },
    { to: '/rules/schedule', label: 'page.rules.tab.schedule' },
  ],
}

export const PLACEHOLDERS: Record<string, PlaceholderSpec> = {
  '/profiles': {
    title: 'nav.profiles',
    lead: 'page.profiles.lead',
    items: ['page.profiles.item1', 'page.profiles.item2', 'page.profiles.item3'],
    legacy: LEGACY.studio,
    tabs: profilesTabs,
  },
  '/profiles/groups': {
    title: 'nav.profiles',
    lead: 'page.profiles.lead',
    items: ['page.groups.item1', 'page.groups.item2'],
    legacy: LEGACY.studio,
    tabs: profilesTabs,
  },
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
}

/** Router basename: Vite's base without the trailing slash ("/app" in builds). */
export const BASENAME = import.meta.env.BASE_URL.replace(/\/$/, '')
