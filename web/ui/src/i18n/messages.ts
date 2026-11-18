// Kongtrol UI strings · v1.0.0 · 2026-11-18
//
// Composed from one module per page (./pages/<page>.ts) plus ./common.ts for
// everything shared (shell, errors, helper, tunnels…), so phase-2 agents
// working on different pages never edit the same file. Add a key to the
// module of the page that shows it; a key two pages show goes to common.
//
// Spanish is the default and the source of the key set; English must have
// exactly the same keys (each module's `Strings<typeof es>` type and
// messages.test.ts, which also rejects a key defined in two modules). Tone
// and terms follow internal/i18n/i18n.go: "tú", "ayudante" for the
// privileged helper, "Instalar ayudante" as the button the daemon's hints
// refer to.
import common from './common'
import activity from './pages/activity'
import home from './pages/home'
import profiles from './pages/profiles'
import rules from './pages/rules'
import security from './pages/security'
import settings from './pages/settings'
import traffic from './pages/traffic'
import tray from './pages/tray'

export type Lang = 'es' | 'en'

/** Every module, for the duplicate-key test. Order does not matter. */
export const MODULES = { common, home, profiles, rules, traffic, security, activity, settings, tray } as const

const es = {
  ...common.es,
  ...home.es,
  ...profiles.es,
  ...rules.es,
  ...traffic.es,
  ...security.es,
  ...activity.es,
  ...settings.es,
  ...tray.es,
}

export type MessageKey = keyof typeof es
export type Messages = Record<MessageKey, string>

const en: Messages = {
  ...common.en,
  ...home.en,
  ...profiles.en,
  ...rules.en,
  ...traffic.en,
  ...security.en,
  ...activity.en,
  ...settings.en,
  ...tray.en,
}

export const messages: Record<Lang, Messages> = { es, en }

/** Replaces {name} placeholders; unknown placeholders are left as-is. */
export function format(template: string, vars: Record<string, string | number> = {}): string {
  return template.replace(/\{(\w+)\}/g, (match, name: string) =>
    name in vars ? String(vars[name]) : match,
  )
}
