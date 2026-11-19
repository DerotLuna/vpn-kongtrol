<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# Architecture Decision Records

Version 1.26.0 · 2026-11-19

Every product or architecture decision in Kongtrol is recorded here as an ADR, using the
[MADR 4.0.0](https://adr.github.io/madr/) template (why MADR: [ADR-0001](0001-record-decisions-with-madr.md),
research: [ADR templates](../research/2026-11-17-adr-templates.md)). Investigations that back a
decision live in [`docs/research/`](../research/README.md).

## Index

| ADR | Title | Status | Date |
|---|---|---|---|
| [0001](0001-record-decisions-with-madr.md) | Record decisions as MADR ADRs, research as dated notes | Accepted | 2026-11-17 |
| [0002](0002-agents-md-cross-vendor-instructions.md) | `AGENTS.md` as the cross-vendor agent instructions | Accepted | 2026-11-17 |
| [0003](0003-product-model-gui-first-cli-complete.md) | One product, GUI-first, CLI-complete | Accepted | 2026-11-17 |
| [0004](0004-detached-daemon-lifecycle.md) | Detached daemon: views never stop tunnels, only `down` | Accepted | 2026-11-17 |
| [0005](0005-privileged-helper-service.md) | Privileged helper service: closed op list, IPC auth, lease, fail-open | Accepted | 2026-11-17 |
| [0006](0006-provider-adapters-attach-mode.md) | Provider adapters with attach mode; direct WireGuard/OpenVPN way | Accepted | 2026-11-17 |
| [0007](0007-openvpn-pushed-routes-binary-cd.md) | OpenVPN: accept pushed routes, binary resolution, `--cd` | Accepted | 2026-11-17 |
| [0008](0008-policy-routing-on-tunnel-down.md) | Policy routing for all adapters, `on_tunnel_down`, notices | Accepted | 2026-11-17 |
| [0009](0009-kill-switch-split-tunneling.md) | Kill switch for split tunneling: block-only, per destination | Accepted | 2026-11-17 |
| [0010](0010-public-ip-check-opt-in.md) | Public IP check is opt-in (Local only) | Accepted | 2026-11-17 |
| [0011](0011-dashboard-vite-react-ts.md) | Dashboard migration to Vite + React + TS by phases, at `/app/` | Superseded by ADR-0043 | 2026-11-17 |
| [0012](0012-landing-cloudflare-pages-multilingual.md) | Landing on Cloudflare Pages, `kongtrol.app`, ES + EN | Accepted; hosting superseded by 0022 | 2026-11-17 |
| [0013](0013-brand-single-source.md) | Brand assets: single source in `assets/brand` | Accepted | 2026-11-17 |
| [0014](0014-i18n-language-resolution.md) | i18n: saved choice, then OS; dashboards follow the browser | Accepted | 2026-11-17 |
| [0015](0015-ci-scoped-builds-generated-docs.md) | CI: build only what changed, generated docs, non-blocking doc checks | Accepted | 2026-11-17 |
| [0016](0016-legal-contact-model.md) | Legal and contact: individual open-source project, Apache-2.0 | Accepted | 2026-11-17 |
| [0017](0017-dependabot-and-github-security.md) | Dependabot grouped weekly; GitHub security features | Superseded by 0021 | 2026-11-17 |
| [0018](0018-failover-only-to-declared-profiles.md) | Failover only to explicitly declared profiles | Accepted | 2026-11-17 |
| [0019](0019-track-supported-go-release.md) | Track a supported Go release (1.27) | Accepted | 2026-11-17 |
| [0020](0020-forticlient-multi-version-strategies.md) | FortiClient multi-version support: detect the version, pick a strategy, keep attach fallback | Accepted | 2026-11-17 |
| [0021](0021-renovate-for-dependency-updates.md) | Renovate for dependency updates; Dependabot only for alerts | Accepted | 2026-11-17 |
| [0022](0022-landing-on-workers-static-assets.md) | Landing on Cloudflare Workers static assets (Workers Builds) | Accepted | 2026-11-17 |
| [0023](0023-provider-logos-on-landing.md) | No third-party logos on the landing: our own monograms for every VPN | Accepted | 2026-11-18 |
| [0024](0024-landing-motion-a11y-performance.md) | Landing polish: CSS-only motion without layout shift, WCAG 2.2 AA, Core Web Vitals | Accepted (boot resume superseded by 0026) | 2026-11-18 |
| [0025](0025-desktop-release-channels-first-run.md) | Desktop release, package managers, signing, safe upgrades and first run without a terminal | Accepted; uninstall data and update check superseded by 0034 | 2026-11-18 |
| [0026](0026-landing-hydrate-prerendered-html.md) | Landing hydrates its prerendered HTML (hydrateRoot) with a hydration check in CI | Accepted | 2026-11-18 |
| [0027](0027-landing-download-ctas.md) | Landing download buttons: one primary per view, recommended OS, phones send the page to a computer | Accepted | 2026-11-18 |
| [0028](0028-landing-hig-material3-review.md) | Landing review with Apple HIG and Material 3: 11px text floor, 44px targets, state layers, readable guide on phones | Accepted | 2026-11-18 |
| [0029](0029-landing-device-class-before-first-paint.md) | Landing marks phones before first paint (boot script `data-device`) so the hero never shifts | Accepted | 2026-11-18 |
| [0030](0030-wireguard-liveness-by-handshake.md) | WireGuard liveness by latest handshake (read-only helper op `wg.stats`); a silent server is a drop, unreadable counters say "unknown" | Accepted | 2026-11-18 |
| [0031](0031-landing-load-performance.md) | Landing load performance: inline CSS, hashed fonts, immutable caching, composited and on-screen-only demo loops, Suspense-split hydration | Accepted | 2026-11-18 |
| [0032](0032-shared-diagnostics-and-export.md) | Shared diagnostics (`internal/doctor`, hint + only `helper_install` fix) and round-trip config export (`internal/configexport`) for CLI, API and dashboard; tray offers the helper without a daemon | Accepted | 2026-11-18 |
| [0033](0033-cli-command-restructure.md) | Reestructura de comandos del CLI: nombres unificados (policy list/show, ip, routes --system, config default-group, reload [perfil]), alias viejos ocultos con aviso traducido, preguntas solo en terminal (--yes), init/up ofrecen el ayudante, config killswitch/dns-guard | Accepted | 2026-11-18 |
| [0034](0034-uninstall-and-update-notice.md) | Uninstall keeps data unless asked (purge with exact keychain entries, never package-manager files); opt-in update notice from GitHub Releases, ≤1/24 h, never auto-update | Accepted | 2026-11-18 |
| [0035](0035-install-scripts-pinned-and-release-assets.md) | One-line install scripts: pinned to the published tag from `landing/src/links.ts` now; from v0.6.0 release assets (+ checksums) behind `kongtrol.app/install.sh` (302 to `releases/latest/download`) | Accepted | 2026-11-18 |
| [0036](0036-tray-flyout-window.md) | Branded tray flyout on left click: `web/ui` `/app/tray` in a pure-Go WebView2 popup next to the icon on Windows (loopback-only, origin filter, nonce bridge); native menu on right click and on macOS/Linux | Accepted | 2026-11-18 |
| [0037](0037-native-tray-menu-states-and-installer-upgrade.md) | Native tray menu: translated states, connected first with state dots, stopped-daemon state with "Iniciar Kongtrol", version line, icon from the vector mark with a state badge; installer PATH on by default and helper updated on every upgrade when outdated; doctor warns about other kongtrol copies in PATH | Accepted | 2026-11-18 |
| [0038](0038-new-dashboard-architecture-and-patterns.md) | New dashboard: react-router + TanStack Query fed by the WebSocket, isolated auth, shared helper state, IA of 7 sections, component kit, vector theme-adaptive brand | Accepted | 2026-11-18 |
| [0039](0039-browser-sessions-one-time-nonce.md) | Browser sessions: one-time nonce (`#k=`, 60 s) minted by native clients, traded for an in-memory HttpOnly session cookie; pages never hand out credentials; Host checked on every route | Accepted | 2026-11-18 |
| [0040](0040-local-hardening-review-2026-11.md) | Local hardening: purge/uninstall delete only Kongtrol's files, WireGuard tunnels bound by key, update link built from the tag, safe names, CI secrets per step and SHA-pinned actions | Accepted | 2026-11-18 |
| [0041](0041-dashboard-data-router-and-unsaved-guard.md) | New dashboard: data router (`createBrowserRouter`), "Tienes cambios sin guardar" guard with `useBlocker` + `beforeunload`, pages loaded on demand; supersedes ADR-0038 decision 1 | Accepted | 2026-11-18 |
| [0042](0042-dashboard-api-completeness.md) | Dashboard API completeness: audit verified/rotated by size/retention/date query, leak action removed (leak only notifies, connected profiles only), profile rename + delete with references + file import, localized group/scheduler errors, scheduler hot reload + rule `enabled`, stable policy order, manual-reconnect notice, gzip for `/app/` | Accepted | 2026-11-18 |
| [0043](0043-new-dashboard-replaces-classic.md) | The new dashboard replaces the classic one and moves to `/`: parity gate closed, `/app/*` and classic pages redirect (308), `/tray`, classic dashboard + `/brand/` deleted, CSP without `unsafe-inline`; supersedes ADR-0011 | Accepted | 2026-11-18 |
| [0044](0044-tray-glyph-relleno.md) | Tray glyph relleno: the tray / menu bar icon is the solid mark derived from the logo (`source/logo-kong-solid.svg`), status dot 17 % with a gap ring (`assets.BadgeGeometry`); refines ADR-0037 §3 | Accepted | 2026-11-18 |
| [0045](0045-explorer-double-click-notice.md) | Double click on `kongtrol.exe` (Explorer, no arguments) shows a translated notice pointing to the installer and waits for Enter; cobra mousetrap off; release body starts with the Windows installer line | Accepted | 2026-11-19 |
| [0046](0046-installers-detect-existing-install.md) | Every installer (Inno Setup, `install.sh`, `install.ps1`) detects an existing install and says update / reinstall / older version, skips questions already answered; `service install` leaves a current helper alone; refines ADR-0037 §7 | Accepted | 2026-11-19 |
| [0047](0047-product-name-kongtrol-and-logo-sizes.md) | The product name is "Kongtrol" ("VPN" only in the tagline; repo, module, keychain unchanged); solid mark up to 32 px (favicon), line art from 36 px | Accepted | 2026-11-19 |
| [0048](0048-tagline-todas-tus-vpn-bajo-kongtrol.md) | Tagline "Todas tus VPN bajo Kongtrol" / "All your VPNs under Kongtrol" (play on "control"); refines ADR-0047 point 1 | Accepted | 2026-11-19 |
| [0049](0049-installer-language-from-system.md) | Windows installer `ShowLanguageDialog=auto`: system language for ES/EN, dialog only for other languages, also on updates | Accepted | 2026-11-19 |
| [0050](0050-installer-resume-tunnels-and-open-request.md) | Updates reconnect the tunnels that were up (`down --all --remember` → `resume`, reusing `daemon --restart`); installer checkbox by profiles, "crear mi primera VPN" = `kongtrol-tray --open /profiles` (always opens) | Accepted | 2026-11-19 |
| [0051](0051-logo-size-cutoff-solid-to-64px.md) | Logo cutoff: solid mark up to ~64 px (landing header 40/36 px), line art from ~128 px (footer ≥ 160 px); refines ADR-0047 points 4–5 | Accepted | 2026-11-19 |
| [0052](0052-app-icon-and-installer-branding.md) | App icon: Ink squircle plate with the solid mark in Bone (the only logo with a background), plain mid-tone mark up to 24 px, per-platform geometry; icon in `kongtrol.exe` too (`rsrc`); Inno wizard images per DPI, tagline as translated text, cut-off Ready text fixed; refines ADR-0044 §2, ADR-0047 §4 | Accepted | 2026-11-19 |
| [0053](0053-profiles-applied-live.md) | VPN profiles created, edited, renamed and deleted without restarting the daemon (`vpn.Set` copy-on-write + `app.AdapterSync`); a connected profile keeps its settings until reconnected (`reconnect_required`, `?reconnect=true`); detect says if an app's tunnel is up | Accepted | 2026-11-19 |
| [0054](0054-recent-destinations-memory-only.md) | Recent destinations (`/api/v1/connections/recent`, `kongtrol traffic recent`): memory only, 500 entries, 1 h, on by default with a switch | Accepted | 2026-11-19 |
| [0055](0055-add-sheet-one-entry-point.md) | One "＋ Agregar" sheet hosted by the dashboard shell for every add (VPN I use, file import, site through a VPN), deep link `?add=…`; "Guardar y activar" restarts only when the daemon cannot apply it live; connected edits "Guardar y reconectar"; input read like `policy.ClassifyTargets` (shared test table) | Accepted | 2026-11-19 |
| [0056](0056-tray-flyout-ready-fallback.md) | Tray flyout: WebView2 controller visible on every show; the page answers `ready` only when on screen, after load and every show; no answer in 3 s → close it, open the native menu, log the cause, native menu on left click until restart (refines ADR-0036) | Accepted | 2026-11-19 |
| [0057](0057-installers-verify-release-signature.md) | Install scripts verify the Sigstore signature of `checksums.txt` with cosign 2.4+ when it is installed (bad or missing signature stops; `KONGTROL_SKIP_SIGNATURE=1` for pre-v0.6.0 releases); otherwise checksum only plus a one-line manual hint; cosign never downloaded | Accepted | 2026-11-19 |
| [0058](0058-linux-dropped-tunnel-dns-and-root-only-configs.md) | Linux: a dropped tunnel releases its DNS (kept for a full tunnel the kill switch holds); systemd-resolved per-link DNS; one kill switch notice per outage; root-only profile files copied with one OS prompt (sudo/pkexec) to a validating root child, never through the helper | Accepted | 2026-11-19 |
| [0059](0059-ci-parallel-jobs-and-superseded-runs.md) | CI: jobs wait only for output they use (UI build vs UI checks, release tests beside the macOS/Windows builds, Pages redirect without the landing build); superseded PR/branch runs cancelled, never tags/releases; `actions/*` pinned by SHA | Accepted | 2026-11-19 |
| [0060](0060-landing-single-theme-bands-and-line-art-marks.md) | Landing in one designed theme: Bone page, Ink bands, one Signal-amber download band, no theme switch (dashboard keeps its themes); key sections fit one laptop screen; downloads listed once; round back-to-top button; ES ↔ EN keeps the section with cross-document View Transitions; line-art mark in the header (54 px) and the in-page illustrations, solid for favicon/tray/tray mock | Accepted | 2026-11-19 |
| [0061](0061-landing-guide-gui-first-steps.md) | Setup guide rewritten GUI-first: panel place first, command second; numbered steps with a "you will see" line; problems as symptom → what to do → command per OS; installer/script per OS from links.ts; two Ink blocks (quickstart, how it works) on the Bone page | Accepted | 2026-11-19 |
| [0062](0062-origin-bound-sessions-token-rotation-listener-owner.md) | Browser session = cookie + per-page token (sessionStorage, `X-Kongtrol-Session`) + one-time 30 s WebSocket ticket; API token rotated on every daemon start and read per request; CLI/tray check the owner of the loopback listener before sending it (Windows SID, Linux uid, macOS unknown); CSP names the one ws origin | Accepted | 2026-11-19 |
| [0063](0063-kill-switch-lan-protected-read-and-release-hardening.md) | Review fixes: a full-tunnel kill-switch block covers the LAN unless `allow_lan` (no DNS fallback to the router); the root read releases only a WireGuard/OpenVPN client profile with one link and refuses with one generic message; release jobs without caches, in the `release` environment, exact GoReleaser/pnpm/govulncheck; install scripts never sudo into a manifest folder (reuse a default or user-writable one) and run from a function | Accepted | 2026-11-19 |
| [0064](0064-audit-log-retention-and-coverage.md) | Audit log: retention_days unset/0 = 30, new max_total_mb (default 500, oldest first), cleanup at start, per new file and every midnight; every mutating API request audited at metadata level (who, what, result, no body), refused credentials throttled, daemon start/stop and kill switch state; `kongtrol config audit-log` | Accepted | 2026-11-19 |
| [0065](0065-windows-provider-app-detection.md) | Proton VPN / NordVPN / Surfshark on Windows found by the uninstall entry and default folders, newest `v<version>` folder compared numerically; one lookup shared by detection and the adapters' launch | Accepted | 2026-11-19 |
| [0066](0066-landing-gold-accent-footer-and-fixed-hero-terminal.md) | Landing accent is one gold `#ffb020` on every tone (fills with Ink text, Ink accent text with gold marks on Bone, a gold download band); footer GitHub once and facts above the wordmark; fixed-height hero terminal log | Accepted | 2026-11-19 |

## Statuses

From MADR 4.0.0 and Nygard (see [research](../research/2026-11-17-adr-templates.md)):

| Status | Meaning |
|---|---|
| Proposed | Under discussion, not yet in force. |
| Accepted | In force. A suffix such as "(implementation in progress)" says the code does not fully match yet. |
| Rejected | Considered and not adopted; kept for the record. |
| Deprecated | No longer applies; nothing replaces it. |
| Superseded by ADR-NNNN | Replaced; the new ADR links back with "Supersedes ADR-NNNN". |

ADRs are immutable once accepted: to change a decision, write a new ADR that supersedes the old
one and only edit the old one's status line. Numbers are never reused (Nygard).

## How to add an ADR

1. Copy the structure of an existing ADR (MADR 4.0.0 sections: Context and Problem Statement,
   Decision Drivers, Considered Options, Decision Outcome, Consequences, Confirmation, More
   Information). The full template: https://github.com/adr/madr/blob/4.0.0/template/adr-template.md
2. Name it `NNNN-title-with-dashes.md` with the next free number.
3. Start with the logo line and `Version X.Y.Z · YYYY-MM-DD`, then a
   `Status · Date · Decision-makers` line.
4. Link the code that implements it and the research note(s) that back it. Every external fact
   needs a URL; if it could not be verified, write **no verificado** (not verified).
5. Add a row to the index above in the same change.
