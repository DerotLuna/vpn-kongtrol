<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0014: i18n — saved choice, then OS; dashboards follow the browser

Version 1.0.0 · 2026-11-17

Status: Accepted · Date: 2026-11-17 · Decision-makers: project maintainer

## Context and Problem Statement

Kongtrol is Spanish-first with English always present. The wizard used to ask for a language at
start. How should each surface pick the language (and, for web UIs, the theme)?

## Decision Drivers

- No question the user must answer before anything works.
- An explicit choice always wins over a guess.
- Every surface consistent; adding a language is data, not code.

## Considered Options

1. Ask in the wizard every time — friction.
2. Fixed Spanish default — wrong for non-Spanish users.
3. Resolve: `KONGTROL_LANG` → saved choice in `preferences.json` → OS language; web UIs follow
   the browser until the user picks.

## Decision Outcome

Chosen: **option 3** (`AGENTS.md`, "Internationalization").

- CLI, wizard, tray: `KONGTROL_LANG` → saved choice (`~/.kongtrol/preferences.json`) → OS
  language (`i18n.System()`, build-tagged, e.g. macOS reads `AppleLanguages`); non-Spanish → EN.
  The wizard does not ask; it ends with a hint: `kongtrol config lang <es|en>`.
- All user-visible strings via `internal/i18n` (`ES` and `EN` maps both required). A new
  language is a new map.
- Daemon sends i18n keys + args; clients render in their own language (e.g. `attach.*`,
  `config.err.*`).
- API errors: language from `X-Kongtrol-Lang`, then `Accept-Language`.
- Dashboards and landing follow the browser language and `prefers-color-scheme` until the user
  picks explicitly; the pick is stored and wins.

### Consequences

- Good: zero-question start; consistent rendering across clients.
- Bad: every new string costs two translations; parity must be tested.

### Confirmation

ES/EN parity tests (`internal/i18n`, `web/ui/src/i18n/messages.ts`).

## More Information

- Code: `internal/i18n/i18n.go`, `internal/i18n/syslang*.go`, `cmd/kongtrol/preferences.go`,
  `cmd/kongtrol/localize.go`
- No external sources: internal design decision.
