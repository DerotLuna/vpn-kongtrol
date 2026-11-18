// Kongtrol UI — sections still served by the classic dashboard · v0.2.0 · 2026-11-18
//
// One entry per route that phase 2 still has to build (docs/dashboard-parity.md
// lists every row). Delete the entry when its real page lands.
import type { PlaceholderSpec } from '../pages/PlaceholderPage'

// Every phase-2 page has landed; empty until a new section is staged here.
export const PLACEHOLDERS: Record<string, PlaceholderSpec> = {}

/** Router basename: Vite's base without the trailing slash ("/app" in builds). */
export const BASENAME = import.meta.env.BASE_URL.replace(/\/$/, '')
