<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0104: After adding a provider app, offer to move the rules of that provider's files to it

Version 1.0.0 · 2026-11-23

Status: Accepted · Date: 2026-11-23 · Decision-makers: Ruben Luna (owner), implemented by agent

Builds on [ADR-0103](0103-proton-same-account-and-shared-tunnel-address.md) and
[ADR-0102](0102-kill-switch-offered-after-first-drop-and-recover-button.md).

## Context and Problem Statement

ADR-0103 hands the traffic over to Proton's app when the user connects it, and its notice
(`watchdog.event.proton_handover_add`, Offer `add_provider`) opens the Add sheet on
`/?add=vpn&type=protonvpn`. Once the user adds the app as a profile, their rules still point at
the direct profile made from a Proton file (e.g. `proton-ch`), which can no longer run next to
the app: those sites lose their VPN until each rule is edited by hand. The tray cannot pass the
direct profile's name in the link (its messages are a closed list with no free values).

## Decision Drivers

- One guided step instead of editing every rule; the user decides, nothing moves silently.
- No GUI-only feature: CLI parity.
- Reuse the provider detection of ADR-0103 (no config field) and the configedit save path.
- The direct profile is kept: the user may go back to it.

## Considered Options

1. Tell the user to edit each rule in Reglas.
2. Pass `from=<profile>` in the Add sheet link.
3. The sheet finds the candidates itself; one bulk, validated move endpoint.

## Decision Outcome

Chosen: option 3 (option 2 fails for the tray, and a stale link would name a wrong profile).

- `GET /api/v1/vpns` adds `provider` per profile (`app.DirectProfileProvider`, empty when
  unknown).
- `configedit.MovePolicies(cfg, from, to)`: both profiles must exist and differ
  (`api.err.policy_via_unknown`, `api.err.policy_move_same` + `.fix`); every rule of `from` is
  re-validated against `to`; one save. `POST /api/v1/policies/move {"from","to"}` →
  `{"status":"moved","from","to","moved":[rule]}`, audit action `config.policy.move`.
  CLI: `kongtrol policy move <from> <to>` (daemon API, or the same edit on `kongtrol.yaml`).
- Panel: after "Guardar y activar" of a provider-app profile P, the Done step
  (`web/ui/src/components/add/MoveRules.tsx`) shows one card per direct profile X with
  `provider == P.type` and at least one rule: "¿Usar P en tus reglas de X?", the count
  (singular / plural), "Pasar reglas" / "Dejarlas en X"; success toast "Listo: N reglas ahora
  usan P.". "Dejarlas" only hides the card.
- CLI: `kongtrol profile add <provider>` asks the same per candidate
  (`app.RuleMoveCandidates`) on an interactive terminal; `--yes` does not answer it (moving
  rules is its own choice, not part of creating the profile); otherwise it prints
  `kongtrol policy move "X" "P"`.

## Consequences

- Good: one click (or one answer) puts the user's sites back on a VPN that works.
- Good: the move is generic: any provider whose files ADR-0103 detects gets the offer.
- Bad: detection only knows Proton's files today; files of other providers get no offer.
- Neutral: rules keep their own `on_tunnel_down`; a value `to` refuses stops the whole move
  with the usual "what to do" line.

## Confirmation

- `internal/configedit/move_test.go`, `internal/app/rulemove_test.go`.
- `internal/api/handlers_policy_move_test.go`: `provider` in the list, move, refusal key.
- `cmd/kongtrol/cmd_policy_move_test.go`: hint without a terminal (and with `--yes`), move on
  "s", `policy move` offline.
- `web/ui/src/components/add/AddSheet.test.tsx`: cards only with candidates, both buttons,
  singular / plural.
- `internal/api/audit_requests_test.go`: the new route has an audit action.

## More Information

Code: `internal/configedit/policy.go`, `internal/api/handlers_policy.go`,
`internal/api/handlers_vpn.go`, `internal/app/rulemove.go`, `cmd/kongtrol/cmd_policy_move.go`,
`internal/i18n/rule_move.go`, `web/ui/src/data/rules.ts`, `web/ui/src/i18n/pages/add.ts`.
No external research: the decision reuses ADR-0103's detection and the existing configedit path.
