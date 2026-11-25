<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0115: "Guardar" saves a new profile without connecting; "Guardar y activar" saves and connects

Version 1.0.0 · 2026-11-25

Status: Accepted · Date: 2026-11-25 · Decision-makers: Ruben Luna (owner, approved), implemented by agent

Amends [ADR-0055](0055-add-sheet-one-entry-point.md) and the AGENTS.md rule "New profiles always
'Guardar y activar'".

## Context and Problem Statement

The "＋ Agregar" sheet had one button for a new profile, "Guardar y activar". In code it created the
profile (`POST /api/v1/vpns`, applied live by ADR-0053; a restart only on `restart_required: true`)
and then the Done step offered "Conectar <name>": "activar" never connected anything, so the label
promised more than it did. Users who want to set a profile up now and connect later (a work VPN
prepared at home, a second provider kept for later) had no plain "save" either. How should the
sheet save a new profile?

## Decision Drivers

- One primary action per step (DESIGN.md); the common path stays one click.
- A label says exactly what happens (user guidance; non-technical users).
- Same result in the panel and the CLI (product model: GUI-first, CLI-complete).

## Considered Options

1. Keep one button; rename it "Guardar" (honest, but the common "set up and connect" takes two clicks).
2. **Chosen:** two buttons: "Guardar y activar" (primary: create, then connect) and "Guardar"
   (outlined: create only).
3. A "Conectar al guardar" check box (one more control to read; a hidden state the button does not show).

## Decision Outcome

Chosen option 2.

- **Panel** (`web/ui/src/components/add`): the VPN form step and the import step show "Volver",
  "Guardar" (outlined) and "Guardar y activar" (filled, the form's submit). Both create the profile
  the same way (`useSaveAndActivate`, which remembers the button as `intent`, so a retried restart
  keeps the choice and the spinner sits on the pressed button). "Guardar y activar" then asks the
  daemon to connect (`useTunnelActions` → `POST /api/v1/tunnels/<name>/connect`, the same call as
  Inicio) and the Done step says "«name» está guardada y se está conectando", without the
  "Conectar" choice. "Guardar" leads to the Done step as before, with "Conectar <name>" first.
  After a restart that needs a new browser session (reauth) nothing connects by itself; the user
  connects from Inicio. Editing an existing profile is unchanged ("Guardar" / "Guardar y
  reconectar", ADR-0053).
- **CLI parity** (verified in `cmd/kongtrol/cmd_profile_add.go`, `createProfile`): `kongtrol profile
  add` and `profile import --create` already create without connecting and print
  `cli.profile.add.next` ("Conéctalo: kongtrol up <name>"). "Guardar y activar" = `kongtrol profile
  add …` followed by `kongtrol up <name>`. No CLI change.
- **Docs**: AGENTS.md "Dashboard (web/ui/)", DESIGN.md, README, the landing guide (ES+EN).

### Consequences

- Good: the label matches what happens; preparing a profile without connecting is one click.
- Good: the common path (save and connect) stays one click and primary.
- Bad: one more button in the footer; on phones the kit stacks them full width.
- Neutral: provider-app profiles connect the way Inicio does (the app may open and wait for the
  user, AGENTS.md "Provider adapters"), now right after saving when the user chose "activar".

### Confirmation

`web/ui/src/components/add/AddSheet.test.tsx` ("Guardar" creates without a connect request;
"Guardar y activar" creates and then posts `/connect`); `FileStep.test.tsx` (import step "Guardar").
`pnpm lint`, `pnpm test --run`, `pnpm build` in `web/ui`.

**No verificado:** the owner's manual run in the panel with a provider app that waits for the user.
