# Research: how profile priority orders overlapping rules

Version 1.0.0 · 2026-11-24

## Question

What does `vpns.<name>.priority` actually decide in Kongtrol, how do other rule-based network
tools rank overlapping rules, and what can the Reglas page and the CLI say truthfully?

## Sources

| Source | Accessed |
|---|---|
| Kongtrol code: `internal/policy/engine.go` (`policy.New`), `internal/config/schema.go` (`VPNConfig.Priority`), `internal/api/handlers_policy.go` (`GET /api/v1/policies` order), `web/ui/src/components/profiles/draft.ts`, `cmd/kongtrol/wizard_profiles.go` | 2026-11-24 |
| [Little Snitch 5 — Rule precedence](https://help.obdev.at/littlesnitch5/ref-rule-precedence) | 2026-11-24 |
| [Research: VPN routes and what wins](2026-11-24-vpn-routes-precedence.md) (priority never reaches the OS route table) | 2026-11-24 |

## Findings

1. **What the engine does (verified in code).** `policy.New` gives each active rule a rank: the
   profile's `priority` when it is greater than 0, otherwise its position among the active rules
   (`len(active) - i`, so the first rule ranks highest). Rules are sorted by rank (higher first),
   then by prefix specificity, keeping file order on ties. Domains and apps: the first match in that
   order wins. IP ranges: the most specific range always wins; the rank only orders ranges of the
   same size. Paused rules are left out of the positional rank.
2. **Explicit and positional ranks share one scale (verified).** A profile with `priority: 5`
   competes with the positional rank of rules without one: with 10 active rules, the first rule of a
   profile without priority ranks 10 and beats the profile with 5. So "the higher number wins" is
   only exactly true when both profiles have a priority (or neither does).
3. **Defaults (verified).** The panel's "＋ Agregar" sheet sends `priority: 10` for new profiles, and
   the `init` wizard writes 10 by default. Hand-written profiles usually have none. Most
   configurations therefore have every profile at 10 (ties, decided by position).
4. **Range (verified).** No range was checked before: a negative value silently meant "unset" and
   the wizard's help said 1–100 without enforcing it (a non-number broke the YAML).
5. **Other tools.** Little Snitch ranks overlapping rules by an explicit priority flag first ("A rule
   with Priority set beats one without it"), then by specificity, then deny over allow. That is the
   behaviour the word "priority" suggests; Kongtrol's shared scale (finding 2) differs.
6. **No OS effect.** Priority never reaches the OS routing table (ADR-0107 research), so it never
   decides against a VPN's own routes.

## Consequences for the design

- Show the order only where it changes something (an overlap the order settles), say which of the
  three cases decided it (priority, position, or one of each), and link to the number.
- Validate the range (0 = unset, 1..1000).
- Making any explicit priority beat every positional rank (Little Snitch's model) would change
  routing for existing mixed configurations; reasoned, not decided here (ADR-0108 "More information").

## Confidence

High for findings 1–4 and 6 (read in code, covered by tests). Finding 5 from the vendor's own docs.

## Open questions

- Should any explicit priority beat every positional rank (owner decision)?

## Used by

- [ADR-0108](../adr/0108-show-and-change-profile-priority.md)
