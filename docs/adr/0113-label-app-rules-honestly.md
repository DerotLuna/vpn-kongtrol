<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0113: Label program (`app:`) rules honestly until per-program routing exists

Version 1.0.0 · 2026-11-24

Status: Accepted · Date: 2026-11-24 · Decision-makers: Ruben Luna (owner), implemented by agent

## Context and Problem Statement

A rule can list programs (`match.apps`, typed as `app:claude.exe`). They are saved and matched
only by `GET /api/v1/resolve`, `POST /api/v1/policies/test` and `GET /api/v1/policies/explain`
(`internal/api/handlers_policy.go`). Nothing routes, blocks or connects by program:
`monitor.PolicyResolver`, `app.KillSwitchService` and `monitor.PolicyWatch` use domains and IP
ranges only (verified in code, 2026-11-24). Yet the "＋ Agregar" sheet's Done step tested an
`app:` rule with explain and said "Probado: claude.exe sale por brainbox", the input hints pushed
users toward `app:`, `kongtrol policy add|show|test` printed a check mark and "→ VPN", the
landing's demo terminal answered "app policy (experimental) → us-content" and the config reference
only said "(experimental)". The product claimed something it does not do.

## Decision

Update before release (2026-11-25): ADR-0111 routes program rules through a direct WireGuard profile on Linux with `app_route_mode`. The label below is hidden for rules whose profile does that (panel `useAppRoutedProfiles`, CLI `appRouted`), and the shared text now names that one case: "Programas: solo salen por la VPN en Linux, con un perfil WireGuard que tenga «Salida por ejecutable». En los demás casos Kongtrol solo los reconoce: usa sitios o IPs."

Until per-program routing exists (roadmap; it needs per-process traffic steering, which no
current Kongtrol component does), every surface that names a program rule says the same thing:
"Programas: aún no se dirigen por la VPN; por ahora Kongtrol solo los reconoce. Usa sitios o IPs
para que salgan por la VPN." (EN: "Programs: not sent through the VPN yet; for now Kongtrol only
recognises them. Use sites or IPs to send traffic through the VPN.")

- **Add sheet.** The Done step never claims an `app:` target goes through a VPN: a program
  target shows that line instead of "Probar". The rule step's "Hoy / Al activar" preview shows the
  line for a program, and the hints that suggest `app:` add the limit.
- **Reglas.** A program chip carries a small "aún no se aplica" tag with the kit Tooltip
  explaining why.
- **Tráfico › ¿Por dónde sale?** A program matched by a rule reads "La regla «X» reconoce
  app:…, pero aún no se aplica" (never "sale por").
- **CLI.** `kongtrol policy add` with an `app:` target prints the line; `policy show` / `policy
  test` mark a matched program "(aún no se aplica)" without the check mark and print the line;
  `policy list` adds the line when a rule lists programs. Help texts say the same.
- **Docs.** `policies[].match.apps` description (`internal/docgen/text.go` → `docs/CONFIG*.md`),
  `configs/example.yaml`, `docs/CONFIGURATION.md`, `AGENTS.md`, the landing demo terminal. The
  landing guide already said "solo explican, no enrutan".

Program rules stay valid in the schema and the API (no migration, no data loss): when routing by
program lands, the labels go and a new ADR supersedes this one.

## Considered Options

1. **Label honestly (chosen).** Cheap, keeps saved rules, matches "the landing only advertises
   what works".
2. **Refuse `app:` targets.** Breaks existing configs and the diagnosis use (`policy show`).
3. **Hide `app:` from the UI.** The CLI and YAML would still accept it silently; worse.

## Consequences

- Good: no surface says a program goes through a VPN when it does not.
- Good: users are steered to sites or IPs, which do work.
- Bad: one more line where programs appear; it goes away with per-program routing.
