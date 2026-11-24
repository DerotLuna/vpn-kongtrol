<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0108: Show and change each profile's priority where rules overlap (panel + `kongtrol profile priority`)

Version 1.0.0 · 2026-11-24

Status: Accepted · Date: 2026-11-24 · Decision-makers: Ruben Luna (owner), implemented by agent

Research: [how profile priority orders overlapping rules](../research/2026-11-24-profile-priority.md).

## Context and Problem Statement

`vpns.<name>.priority` orders a profile's rules against other rules that cover the same site or the
same IP range (`policy.New`). It was hidden under "Más opciones" in the profile editor, the Reglas
page only said "change it in Perfiles", and the CLI could set it only in `kongtrol init`. Users
could not see why one overlapping rule won, nor change it from where they saw the overlap.

## Decision

1. **Reglas, "Sale por" cell.** A small link under the VPN state, "Prioridad N" or "por posición",
   with the kit Tooltip ("Solo decide entre tus reglas cuando dos cubren el mismo sitio o el mismo
   rango de IP: gana el número mayor…"). It appears **only on rows in an overlap the order settles**
   (same domain, wildcard, same-size range). Every new profile starts at 10, so a hint on every row
   would read "Prioridad 10" everywhere and say nothing; on rows without such an overlap the order
   changes nothing. Nested ranges never show it (the most specific range wins). The link opens that
   profile's editor with the field focused. The cell stays a column of short lines, so the
   container-query cards keep working.
2. **Overlap note.** Each overlap line adds who decided: priority ("oficina (20) va antes que casa
   (10)"), position (neither has one, or both the same), or one of each ("el resultado puede cambiar
   si mueves reglas; pon prioridad a las dos"). For position and mixed it offers "Cambiar prioridad"
   for the VPN that lost (so it can win).
3. **Deep link.** `/profiles?edit=<name>&focus=priority` opens that profile's editor; the
   parameters are dropped once read. The rule editor's priority note uses it for the chosen VPN.
4. **Profile editor.** "Prioridad" moves out of "Más opciones" into its own small section "Orden de
   sus reglas": a narrow number field, empty = por posición (placeholder), the same one-line hint
   and the YAML key. Checks before saving and from the daemon (`config.err.profile_priority`) land on
   the field with a "what to do" line. New profiles keep 10 (no change in routing).
5. **Perfiles list.** The VPN cell's type line adds "· Prioridad N" when set. Unset shows nothing:
   it is the default meaning ("por posición") and the list stays as short as before.
6. **Range.** `internal/config` refuses a priority below 0 or above `MaxPriority` (1000) with
   `config.err.profile_priority` (+ `.fix`). 0 = unset. The wizard validates the same way.
7. **CLI parity.** `kongtrol profile priority <name> [N|none]` shows or changes it (0 or `none`
   clears), through `PUT /api/v1/vpns/{name}` when the daemon runs and `configedit.Save` otherwise —
   the same shape as `kongtrol policy state <name> [on|off]` and `profile on-tunnel-down`, chosen
   over a one-flag `profile set --priority` because the CLI has one subcommand per setting and no
   general profile editor. `kongtrol profile list` (new; there was no profile listing) shows each
   profile's type and order. `kongtrol policy list` adds an ORDER column and a hint line; `--json`
   of all three carries `priority` (0 = by position; the `profile` commands also `by_position`).
   The deprecated `kongtrol map` keeps its old columns.

No API change: `GET /api/v1/vpns` already returns `priority`, `PUT /api/v1/vpns/{name}` already
accepts it.

## Consequences

- Good: the order is visible exactly where it matters, with the reason and a one-click way to
  change it; panel and CLI say the same thing.
- Good: an out-of-range number is caught everywhere with what to do.
- Bad: a config with a negative or >1000 priority, which loaded before, now fails validation with
  that message. Kongtrol itself never writes such a value (hand edits: **no verificado**).
- Neutral: the Reglas page now also reads `GET /api/v1/vpns` (already cached by other pages).

## More information

Explicit and positional ranks share one scale in `policy.New` (research finding 2): with only one
of two overlapping profiles numbered, the result also depends on rule positions. This ADR shows that
case honestly ("una de cada") instead of changing the engine. Making any explicit priority beat
every positional rank, as Little Snitch does, would change routing for existing mixed
configurations and needs its own owner decision and ADR.

Code: `internal/config/semantics.go`, `cmd/kongtrol/cmd_profile_priority.go`,
`cmd/kongtrol/traffic_map.go`, `cmd/kongtrol/wizard_profiles.go`,
`web/ui/src/components/rules/{model,preview}.ts`, `web/ui/src/components/rules/PolicyList.tsx`,
`web/ui/src/components/profiles/{ProfileForm,ProfileList,ProfileEditor}.tsx`,
`web/ui/src/components/profiles/draft.ts`, `web/ui/src/pages/ProfilesPage.tsx`.
