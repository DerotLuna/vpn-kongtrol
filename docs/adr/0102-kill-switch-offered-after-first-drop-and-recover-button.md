<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0102 · Offer the kill switch after the first drop, and a "Recuperar internet" button

Version 1.0.0 · 2026-11-23

Status: Accepted · Date: 2026-11-23 · Decision-makers: project maintainer

## Context and Problem Statement

The kill switch is off by default (`security.kill_switch.enabled` unset). A new user learns that
it exists only by reading Seguridad. When a VPN drops on its own, the watchdog reconnects it
(ADR-0098), and in the meantime that VPN's traffic goes out over the normal internet. That is
exactly the moment the user can understand what the kill switch does, but nothing told them.

The opposite case also needs a fix. With the kill switch on, a dropped tunnel stays blocked (on
purpose, security first). The notices (`killswitch.event.blocking*`,
`watchdog.event.gave_up_blocking*`) said "desconéctala" in words only, with no button.

## Decision

1. `monitor.Event` gets one optional field `Offer` (`json:"offer,omitempty"`): an extra action
   clients show as a button. Values: `enable_killswitch`, `release_block`, `add_provider`
   (`Args[len-1]` = adapter type). An unknown value is ignored by every client.
2. **Kill switch offered once.** On a profile's first unexpected drop (the watchdog's
   `reconnect_attempt`; a user disconnect never emits it, ADR-0097), if the kill switch does
   not protect that profile (`KillSwitchService.ProtectsProfile`: global setting or the
   profile's own `kill_switch`), the daemon emits `killswitch.event.offer` with
   `Offer: enable_killswitch` (`cmd/kongtrol/daemon_killswitch_offer.go`). It sets
   `killswitch_offered: true` in `~/.kongtrol/preferences.json` **when it offers**, so the offer
   never repeats, whatever the answer, and also when the notice is ignored or the daemon restarts.
   If preferences.json cannot be written, the offer is still made at most once per daemon run.
   - "Activar protección" calls the existing `POST /api/v1/security/killswitch {"enabled":true}`
     (CLI: `kongtrol config killswitch on`). "No, gracias" only closes the notice in that client.
     The answer is not needed for the "asked" record, so there is no new endpoint and no new
     audit action. Turning the kill switch on is audited as before (`security.killswitch.set`).
   - Text (ES, the owner's wording; title and body joined with ` · ` like every notice):
     "X se cayó sin que lo pidieras · mientras volvía a conectar, tu tráfico salió sin VPN.
     ¿Quieres que Kongtrol lo impida? Si X se cae, Kongtrol bloquea su tráfico hasta que vuelva.
     Si no vuelve, te aviso y con un clic recuperas internet."
3. **"Recuperar internet".** Every kill switch block notice (`killswitch.event.blocking`,
   `blocking_full`, `blocking_full_lan`, `watchdog.event.gave_up_blocking`,
   `gave_up_blocking_full`) carries `Offer: release_block`. The button disconnects that profile
   through the normal disconnect API. That is a user disconnect: the watchdog stops, the kill
   switch lifts that block (ADR-0097) and the profile is removed from `wanted.json` (ADR-0098).
   CLI: `kongtrol down <profile>`.
4. **"Agregar <App>"** (`add_provider`, emitted by another change) opens the "＋ Agregar" sheet
   with the provider chosen: `/?add=vpn&type=<type>` (`src/app/addLink.ts`, already supported).
   The tray flyout cannot send free values. It sends one closed message per provider
   (`open-add-protonvpn|nordvpn|surfshark|forticlient`, `flyoutMessages` in
   `cmd/kongtrol-tray/flyout.go`, mirrored in `web/ui/src/tray/bridge.ts`). Protonvpn is labeled
   "Agregar Proton". CLI: `kongtrol profile add <type>`.
5. Where each client renders offers:
   - Panel: Inicio › Avisos and Actividad show the offer button next to "Reintentar". "No,
     gracias" shows for every offer except `release_block`. The toast's single action is the
     offer when there is one. A button hides once it no longer means anything: the kill switch
     is already on, or the profile is no longer in `kill_switch_status.blocking`.
   - Tray flyout: the latest useful offer appears as a notice with its button.
   - Native tray menu: one item next to "Conectar <perfil>". It hides when clicked, after 30
     minutes, or when the blocked profile is connected again.
   - CLI live view: the notice line ends with the equivalent command.

Moving the dropped profile's rules to a newly added provider profile is **pending**. The sheet
does not offer it yet.

## Considered Options

- **Turn the kill switch on by default.** Rejected: a full-tunnel drop would cut a new user's
  whole internet with no explanation. ADR-0058 and ADR-0099 keep the block opt-in.
- **Record "asked" only on an answer** (new endpoint for "No, gracias"). Rejected: an ignored
  notice, or a daemon restart before the answer, would ask again on the next drop. The owner
  asked for once.
- **Offer on every drop until answered.** Rejected: that is noise. The once-per-outage notice
  rule (ADR-0098) is the reference.

## Consequences

- One more machine-local field in preferences.json (`killswitch_offered`). The data purge
  deletes it with the file, so after a purge Kongtrol offers again.
- Clients older than this change ignore `offer` (the JSON field is omitted when empty). The text
  of `killswitch.event.offer` falls back to their generic line.
- Not verified: how the WebView2 flyout and every desktop notification backend render the
  longer offer text (no verificado).

## Links

- Code: `cmd/kongtrol/daemon_killswitch_offer.go`, `cmd/kongtrol/daemon_watchdog.go`
  (`watchdogNotice`), `internal/app/killswitch_service.go` (`ProtectsProfile`),
  `internal/config/preferences.go`, `internal/i18n/killswitch_offer.go`,
  `cmd/kongtrol/policy_notice.go` (`offerHintCLI`), `cmd/kongtrol-tray/tray_offer.go`,
  `cmd/kongtrol-tray/flyout.go`, `web/ui/src/data/offers.ts`,
  `web/ui/src/components/home/OfferActions.tsx`, `web/ui/src/tray/TrayOffer.tsx`.
- Related: [ADR-0058](0058-linux-dropped-tunnel-dns-and-root-only-configs.md),
  [ADR-0097](0097-user-disconnect-never-a-drop.md),
  [ADR-0098](0098-bounded-reconnect-and-restore-at-start.md),
  [ADR-0099](0099-refuse-second-full-tunnel-and-switch-aware-kill-switch.md).
- No external research: the change reuses the existing notice, toggle and disconnect
  mechanisms.
