<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0052: "＋ Agregar": one sheet for every add in the dashboard

Version 1.0.0 · 2026-11-19

Status: Accepted · Date: 2026-11-19 · Decision-makers: project maintainer (owner)

## Context and Problem Statement

Adding things to Kongtrol was spread over three page-local forms: Perfiles had a two-step
"Nueva VPN" editor (type picker, then the full form) and a separate first-run guide; Reglas had
an inline editor with three templates; importing a `.conf` / `.ovpn` was a button inside the
WireGuard/OpenVPN form. A non-technical user who sees "claude.ai sale por internet normal" on
Inicio, Tráfico or Actividad had to know which page and which form turns that into a rule, and
a new profile then needed a separate "Reiniciar ahora" before it could connect.

## Decision Drivers

- GUI-first, plain language, one obvious place (AGENTS.md "Product model", "User guidance").
- Ask only what is needed; guide the rest (Windows "Add a VPN connection", research note).
- Start from what the user is looking at (Little Snitch builds rules from alerts and the
  connection list).
- The CLI has the same verbs (`kongtrol policy add/remove`, `profile add`,
  `profile import --create`, `profile detect`): same input rules on both.
- Data layer, kit, CSP and accessibility rules of ADR-0038/0041 unchanged.

## Considered Options

1. Keep page-local forms and add more buttons to them.
2. A new `/add` page.
3. **One sheet hosted by the shell, opened by a deep link from every entry.**

## Decision Outcome

Option 3, owner-approved (phases 1–2, UI part). Sources in
[the research note](../research/2026-11-19-add-flow-patterns.md).

1. **Entry points.** A tonal (not amber: the one amber action stays inside each page) "＋
   Agregar" button in the top bar; the empty states of Inicio, Perfiles and Reglas; Perfiles
   "Nueva VPN" and the first-run tiles; Reglas "Nueva regla"; "＋" on each Inicio tunnel row
   ("Hacer que un sitio salga por {vpn}"); "Hacer que esto salga por…" on a Tráfico lookup
   result and "Hacer que un sitio salga por {vpn}" in Tráfico's tunnel routes; "＋" on an
   Actividad notice about a profile. All open the same sheet.
2. **Deep link.** `?add=menu|vpn|file|rule[&via=<perfil>&target=<sitio>&app=<exe>&type=<tipo>]`
   on the current page (`web/ui/src/app/addLink.ts`). Only the query changes, so it is not
   "leaving a form"; closing replaces the entry (Back does not reopen it). `type` lets the
   first-run tiles skip to "Usar <app>".
3. **Sheet** (`web/ui/src/components/add`): native `<dialog>` (kit Dialog), lazy-loaded; focus
   starts on each step's first control and returns to the opener; Esc / close with a draft asks
   "Tienes cambios sin guardar · Seguir editando / Descartar"; `useUnsavedChanges` guards the
   tab. Menu: "Conectar una VPN que ya uso" (found apps listed), "Importar un archivo de VPN",
   "Hacer que un sitio salga por una VPN".
4. **A — VPN I already use.** Found apps first with "Usar"; "Otra VPN" opens the existing type
   picker (WireGuard/OpenVPN go to B). Only the name (from the app's label, cleaned and made
   unique: "NordVPN 2") and, for corporate clients, the gateway; the rest under "Más opciones".
   **"Guardar y activar" never needs a separate restart** (owner decision): POST /api/v1/vpns;
   when the daemon applied it live (`restart_required: false`, ADR-0053) it is ready at once,
   otherwise (setup mode, an older daemon) the sheet runs the shared restart itself (one line says
   what happens: tunnels that were up reconnect after a brief cut). A
   failed restart keeps the profile and offers only "Reintentar el reinicio". Done: "Conectar",
   "Hacer que un sitio salga por ella", "Ver en Inicio".
5. **B — import a file.** Button + drop zone, and a window-wide drop overlay (announced in a
   polite live region; off while another dialog is open). Type from the extension
   (`profileimport.TypeForFile`); name from the file name, valid for the stored copy too. An
   `.ovpn` with `auth-user-pass` shows user/password with that sign-in preselected (a client-side
   hint). Save = POST /api/v1/vpns/import, then POST /api/v1/vpns, then restart; `409
   api.err.import_exists` offers "Reemplazar"; other `api.err.import_*` read through
   `describeError`. The stored path appears only under "Más opciones" and in Done.
6. **C — a site through a VPN, one sentence.** "Sitio, IP o programa" is read exactly like
   `policy.ClassifyTargets` (URL → host, domain + `*.domain` when "incluye subdominios" is on,
   IP/CIDR, `app:x`); both Go and TS tests run the same table
   (`internal/policy/testdata/classify_targets.json`). A bare word or `x.exe` is accepted like the
   CLI does but gets a hint. "Sale por" defaults to the deep link's profile, else the only one,
   else the first connected. "Si esa VPN está apagada": normal connection (default) / block. The
   name is generated like `policy.AutoName` ("claude.ai por Trabajo") and sits with "más sitios"
   under "Más opciones". Preview "Hoy / Después" from GET /api/v1/policies/explain and POST
   /api/v1/policies/test. With no profile, A runs first and the rule comes back with the draft
   kept and the new VPN chosen. Done: "Probar", "Conectar {vpn}" if off, "Agregar otro sitio".
7. **Editing stays where it was**, minus the create-only parts (the profile editor's type step
   and the rule templates are removed). Editing a **connected** profile saves with "Guardar y
   reconectar" (`PUT /api/v1/vpns/{name}?reconnect=true`, "Se corta unos segundos solo esta
   VPN", "Reconectando…" while saving); an older daemon that answers `reconnect_required` gets a
   toast with "Reconectar" (the tunnel reload). A disconnected profile saves with "Guardar".
8. **Phase 3 contract (backend in ADR-0053 / ADR-0054).** Found apps carry `active` /
   `interface` (GET /api/v1/vpns/detect): the one connected now is listed first with "Conectada
   ahora". Deleting a connected profile is refused (`409 api.err.vpn_delete_busy`) and reads in
   the same dialog as the other delete refusals. Tráfico gets **Recientes** (GET
   /api/v1/connections/recent: destination, program, tunnel or "Conexión normal", when; "Hacer
   que esto salga por…" opens C prefilled), on by default (ADR-0054) with a switch in Ajustes › Privacidad
   (`recent_connections_enabled`, "Se guarda solo en memoria, se borra al cerrar Kongtrol");
   `supported: false` says the OS does not allow it yet; an older daemon (404) shows no card.

### Consequences

- Good: one place to learn; every page leads to it with the context prefilled; same input rules
  as the CLI, tested on one table.
- Good: a new profile is usable right after saving (no separate restart step).
- Bad: when the daemon cannot apply a new profile live, "Guardar y activar" restarts Kongtrol even
  if another tunnel is up (brief cut, said
  before saving). An older daemon without `?reconnect=true` answers `reconnect_required`; then
  the dashboard falls back to the "Reconectar" toast (also on `reconnect_error`).
- Neutral: creating a profile with every advanced field now takes create + edit; all fields stay
  in the editor, the CLI and the YAML.

### Confirmation

Vitest: `web/ui/src/components/add/model.test.ts` (shared classification table, AutoName,
names, files, default VPN, deep-link parser), `AddSheet.test.tsx` (every entry opens the sheet
prefilled, focus / Esc / focus return / aria, create → restart → Done, no profile → A → back to
the rule, import 409 + `auth-user-pass`, unsaved draft), `pages/ProfilesPage.test.tsx` ("Save
and reconnect" / "Save", older daemon), `pages/RulesPage.test.tsx`. Go:
`internal/policy/classify_test.go` on the same table.

## More Information

- Code: `web/ui/src/components/add/`, `web/ui/src/app/addLink.ts`, `web/ui/src/data/add.ts`,
  `web/ui/src/components/layout/AppShell.tsx`, `internal/policy/classify.go`.
- Builds on [ADR-0038](0038-new-dashboard-architecture-and-patterns.md),
  [ADR-0041](0041-dashboard-data-router-and-unsaved-guard.md),
  [ADR-0042](0042-dashboard-api-completeness.md).
- Research: [Add-flow patterns](../research/2026-11-19-add-flow-patterns.md).
