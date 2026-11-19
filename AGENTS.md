# AGENTS.md

Version 1.24 · 2026-11-19

Guidance for coding agents (any vendor) working in this repository. `CLAUDE.md` only imports this file.

Public site: https://kongtrol.app (guide `/guia`, terms `/terminos`, privacy `/privacidad`, security `/seguridad`; English under `/en/` with localized slugs: `/en/guide`, `/en/terms`, `/en/privacy`, `/en/security`). `kongtrol.dev` and the old GitHub Pages URL only redirect there — never reference them.

## Module

`github.com/vpn-kongtrol/kongtrol` — Go 1.27+

## Common Commands

```bash
# Build
make build                  # current platform → build/dist/kongtrol
make build-all-cli          # cross-compile CLI for all platforms (CGO_ENABLED=0)
make build-tray-native      # tray app, native only (requires CGO)

# Test
go test ./...               # all packages
go test ./internal/policy/  # single package
go test -run TestEngine_ResolveIP ./internal/policy/  # single test
make test-race              # with race detector

# Docs (see docs/DOCS-TOOLING.md)
make docs                   # regenerate docs/CLI*.md, docs/cli.json, docs/CONFIG*.md
make docs-check             # report-only: stale generated docs, change impact, i18n parity

# Vet / lint
go vet ./...
golangci-lint run ./...

# Tidy
go mod tidy
```

The tray binary (`cmd/kongtrol-tray`) needs `CGO_ENABLED=1` and a native build only on macOS (Cocoa); on Windows and Linux `fyne.io/systray` is pure Go (Win32 / D-Bus StatusNotifierItem) and cross-compiles like the CLI (`cmd/kongtrol`). Packaging lives in `packaging/` (Inno Setup, macOS `.app`, `.desktop`, Homebrew formula, npm).

## Architecture

### Adapter pattern

Every VPN is a `vpn.VPNAdapter` (defined in `internal/vpn/adapter.go`). Each adapter package registers itself via `init()` using `vpn.Register(name, factory)`. The binary activates adapters through blank imports in `cmd/kongtrol/main.go`.

When adding a new adapter:
1. Create `internal/vpn/<name>/adapter.go` + `cli.go` + `util.go`
2. Call `vpn.Register("name", ...)` inside `func init()`
3. Add a blank import `_ "github.com/vpn-kongtrol/kongtrol/internal/vpn/<name>"` in `cmd/kongtrol/main.go`
4. Add the type key to the `oneof` validator in `internal/config/schema.go`
5. Add an example entry in `configs/example.yaml`

`Status()` must call `.Normalize()` on the internal status field before returning — zero value (`""`) must never escape to callers.

### Product decisions (settled — do not re-ask)

<!-- feature:providers -->
**Provider adapters vs direct configs.**
- A provider adapter (`forticlient`, `protonvpn`, `nordvpn`, `surfshark`, …) works with the
  provider's **official app**. Bringing the connection up and managing the tunnel are separate:
  however the app's connection gets up, Kongtrol manages the tunnel the same way (detects it by
  network interface — `interface:` overrides the name pattern — and attaches: status, watchdog,
  DNS, policies).
- On `up`: (1) if the tunnel is already up → attach; else (2) if `auto_connect` (default true)
  → launch the app and try to connect automatically (official CLI, or GUI automation for
  corporate clients); automation is **best effort** — if it fails (e.g. app version change) it
  is not an error: fall back to (3); (3) launch the app, tell the user what to do (translated)
  and **wait**, attaching as soon as the user connects — before or during `up`.
- If Kongtrol can't disconnect (no CLI), `down` asks the user to disconnect in the app and
  waits; the watchdog notifies instead of reconnecting.
- Never GUI-automate a consumer VPN app; GUI automation only for corporate clients with no
  alternative (e.g. FortiClient).
- The **direct** way, `type: wireguard` / `type: openvpn` with the provider's downloadable
  `.conf` / `.ovpn`, stays available and needs no app (e.g. Proton on Windows as `wireguard`).

<!-- feature:daemon_lifecycle feature:tray_exits -->
**Daemon lifecycle.**
- `kongtrol up` starts the detached hidden `kongtrol daemon` (or reuses a running one) and then
  opens a live view. Quitting any view (q / Ctrl+C, `status --watch`, tray, dashboard) **never**
  stops the daemon or its tunnels. Only `kongtrol down` (or the tray's
  "Salir y desconectar todo") stops them. No terminal has to stay open.
- After `kongtrol down` the tray stays open: header "Kongtrol detenido", an "Iniciar Kongtrol"
  item (`ensureDaemon`), and it recovers by itself when any client starts the daemon again
  (it keeps polling; refused connection = stopped, timeout = "no responde"). ADR-0037.
- There is exactly one daemon per user; CLI, tray and dashboard are clients of its local API
  (127.0.0.1, token in `~/.kongtrol/api-token` for CLI/tray; browsers get a session cookie from a
  one-time nonce, never the token: `daemonctl.(*Client).PanelURL`, ADR-0039). This local API is IPC on the same machine —
  nothing leaves the host.

<!-- feature:first_run -->
**First run and restart (ADR-0025).**
- With no `kongtrol.yaml`, only `kongtrol daemon` may start, in **setup mode**
  (`config.NewEmpty()`, `GET /api/v1/setup`): the tray starts it and opens the dashboard's
  Perfiles page (`/profiles`), whose profile editor writes the first profile to `config.DefaultPaths()[0]`. Every
  other command still points to `kongtrol init`, which stays the terminal path.
- Restart = new daemon process: the dashboard's `POST /api/v1/restart` makes the daemon spawn
  `daemon --restart <connected profiles>` from its own executable and then stop like `down`;
  `kongtrol restart` stops it and spawns the replacement from the CLI's own executable (after a
  package-manager upgrade the binary path may change). The replacement waits for the old API to
  go away (holding the spawn lock) and drops profiles that no longer exist.
- The first-run tray menu also carries "Instalar ayudante (una vez)" (shown while the helper is
  missing, checked by pinging it when no daemon answers) plus a one-time notice
  `tray.helper_first_run` (ADR-0032).
- First run in one command: `kongtrol init` ends by checking the helper (`offerHelperInstall`,
  default yes) and asking "¿Conectar ahora?" (`finishInit`: restarts a running daemon, then the
  `up` path). `up` on a TTY offers the helper when a connect fails with
  `service.error.not_installed|outdated`. `reload` with profiles the daemon does not know offers
  the restart path (`restartRunningDaemon`, shared with `kongtrol restart`).

<!-- feature:cli_names -->
**CLI names and questions (ADR-0033).** Old command/flag names stay as hidden deprecated aliases
(`cmd/kongtrol/deprecated.go`: `deprecatedCommand` / `warnDeprecated` print `cli.deprecated` on
stderr; cobra/pflag deprecation texts are English-only, so they are not used). Yes/no questions
only through `cmd/kongtrol/prompt.go` (`confirmAction`): asked only on an interactive terminal,
never with `--json`/`-q`, `--yes` answers for scripts, otherwise print the "what to do" line.

<!-- feature:distribution -->
**Distribution (ADR-0025).** Tags `v*` run `release.yml`: macOS app (native, cgo), Windows Inno
Setup per-user installer (native), GoReleaser OSS for the rest (CLI, Linux/Windows tray,
archives, `.deb`/`.rpm`, Scoop, winget, AUR, SBOM, cosign), then attestations, Homebrew
**formula** (never a cask: casks get the quarantine flag) and npm (esbuild-style
`optionalDependencies`, no postinstall). Every publisher skips itself without its secret.
Upgrades in place keep `~/.kongtrol`, keychain entries and the helper, and reconnect the tunnels
that were up (ADR-0050): the NEW binary runs hidden `down --all --remember` (writes
`~/.kongtrol/run/resume.json`, one-shot, 24 h), then after files + `service install --if-outdated`
hidden `resume` reconnects them (`daemon --restart <profiles>`, or the running daemon's API). The
Inno finished-page checkbox depends on profiles existing; "crear mi primera VPN" runs
`kongtrol-tray --open /profiles`, which always opens it (also from a second tray instance).
`install.sh`/`install.ps1` are
release assets too, served as `https://kongtrol.app/install.sh` / `install.ps1` (302 to releases/latest,
`landing/public/_redirects`); the one-liner base lives only in `landing/src/links.ts` (ADR-0035). They
check `checksums.txt` and, only when `cosign` 2.4+ is installed, its Sigstore bundle first (bad or
missing signature stops; never download cosign; ADR-0057).
The release body starts with the Windows installer line (`.goreleaser.yaml` `release.header`); a
double-clicked `kongtrol.exe` (Explorer, no arguments) prints `cli.explorer_notice` and waits for
Enter — cobra's own mousetrap is off because it adds an English-only line (ADR-0045).
Every installer detects an existing install (Inno: HKCU `{AppId}_is1` `DisplayVersion` in
`InitializeSetup`; scripts: `install.manifest` + `kongtrol version`) and presents itself as
update / reinstall / older version (SemVer precedence, same table in all three), skipping
license/folder/tasks and reusing previous folders and choices; an older version is confirmed
only interactively (ADR-0046). Plain `kongtrol service install` leaves a current helper alone
(no UAC/sudo) and says "Actualizando" for another version.

<!-- feature:uninstall -->
**Uninstall keeps data unless asked (ADR-0034).** `internal/uninstall`: `Detect` picks the channel
(install.sh/ps1 via `~/.kongtrol/install.manifest`, Windows installer = `unins000.exe` next to the
exe, package manager by path, else manual) and only script-installed files are ever deleted;
`Purge` deletes `~/.kongtrol` + exactly `vpn-kongtrol` keychain items `<profile>.username|password`
(profiles read from `kongtrol.yaml`, `config.yaml` and backups — go-keyring cannot list) +
`_system.audit_hmac_key`. Surfaces: `kongtrol uninstall [--purge|--data-only] [--yes] [--dry-run]`,
`install.sh --uninstall [--purge]`, `install.ps1 -Uninstall [-Purge]`, the Inno uninstaller
(asks, default No; silent never deletes), web/ui Ajustes "Borrar todos mis datos" →
`POST /api/v1/data/purge {"confirm":true}` (daemon keeps api-token/run/logs, then restarts into
setup). The tray has no uninstall item on purpose.

<!-- feature:update_check -->
**Update notice: opt-in, never auto-update (ADR-0034).** `updates.check_enabled` (default false;
`kongtrol config update-check on|off|status`, web/ui Ajustes, one question in `init`).
`internal/update` asks `api.github.com/repos/DerotLuna/vpn-kongtrol/releases/latest` at most once
per 24 h (state `~/.kongtrol/update-check.json`), generic `User-Agent: kongtrol/<version>` only,
https, 5 s, 1 MiB, pre-releases ignored. Newer → `monitor.Event` `update_available`
(`update.event.available`, args latest/current/url) once per version; `version`/`status` show a
line from the remembered state (no request).

**Standards first, then improve.** Before designing a mechanism, research how others already
solve it — official docs, RFCs and standards, mature open-source projects, vendor docs,
papers and known open problems; never limit the search to a fixed list of names. Build on
and reuse what works (prefer a vetted library over our own code); innovating is welcome
when it improves on that base or solves problems others still have, and we can say why.
Cite sources in proposals and say when something is reasoned but unverified.

**Verify, then record.** Nothing is proposed, claimed or applied without verifying it
(official docs, code, tests); say "no verificado" when you can't. Every decision gets an ADR in
`docs/adr/` and every investigation a note with sources and access date in `docs/research/`,
in the same work as the change.

<!-- feature:public_ip -->
**Nothing stale stays.** Unused or obsolete code, assets, keys, docs and scripts are deleted
(git is the history). The only exception is ADRs: per the ADR standard (Nygard; MADR status
`superseded by ADR-NNNN`) a changed decision gets a NEW ADR and the old one only flips its
status to superseded — never edited otherwise, never deleted, numbers never reused. Research
notes stay only while an ADR cites them.

**Local only.** The app never sends or processes user data off the machine. Any outbound call
(other than the VPN connection itself) must be opt-in, documented in `/privacidad` (EN
`/en/privacy`) and listed in `docs/SECURITY.md`.

**Docs stay in sync.** Every change updates, in the same work, every surface it touches: CLI
`--help` and `docs/CLI.md`, README, `AGENTS.md`, the dashboard (`web/ui`),
the tray, the wizard, `configs/example.yaml`, `docs/` and the landing (`landing/`, ES+EN,
including the guide). The landing only advertises what already works.

**Generated docs — never edit by hand.** `docs/CLI.md`, `docs/CLI.en.md`, `docs/cli.json` (from the
cobra tree + `internal/i18n`) and `docs/CONFIG.md`, `docs/CONFIG.en.md` (from
`internal/config/schema.go` + descriptions in `internal/docgen/text.go`) are written by `make docs`
(`go generate ./cmd/kongtrol`). The landing guide's command reference reads `docs/cli.json`. Change
the source (help keys, flag texts, schema, `text.go`), then run `make docs`. A new config field needs
an ES+EN description in `internal/docgen/text.go`.

**Change impact check (required, non-blocking).** Before finishing any change, check
`docs/impact.yml`: for every feature whose code you touched, update all its surfaces or write
`no aplica: <motivo>` in the PR/commit description. Run `make docs` if CLI/config changed. CI only
warns; it never blocks for docs. Mark new user-facing paragraphs with `<!-- feature:<name> -->`
(Markdown) or `// feature:<name>` (TS/TSX/JS) and list them as `path#name` surfaces; a new
top-level package goes into a feature or into `internal` (tested). See
`docs/DOCS-TOOLING.md`.

**Build only what changed.** CI and deploys are scoped by path: a landing-only change builds
and deploys only the landing; a Go-only change never rebuilds the landing; `web/ui` builds only
when its sources change. `web/**` and `assets/**` are part of the Go binary (embedded), so they
do trigger the Go pipeline. Even when a resource is free or unlimited (CI minutes, API quotas,
tokens, network calls, polling, binary size), use it with judgment — optimal, never at the cost
of quality, correctness or coverage: cut waste, not checks.

**Brand and icon standard (owner-approved, ADR-0047/0051/0052).** Every logo, icon and image
follows DESIGN.md "Logo (name and sizes)": all bitmaps generated from the SVG masters by
`assets/brand/generate.go` (never hand-made, never a PNG logo on a filled disc); solid mark up to
~64 px (favicon, tray, panel marks, landing header), line art from ~128 px; the app icon (Ink
squircle plate + Bone solid mark, plain mid-tone mark ≤ 24 px) is the only logo with a background;
web logos are transparent and follow the theme. A new surface (store listing, social card, PWA,
docs) reuses these assets; any deviation is proposed first and gets an ADR.

**Product model.** One product, GUI-first, CLI-complete. Everyone (devs included) uses the
app: installer, tray, dashboard to set up and connect. The CLI has full parity for
automation. No feature may exist only in the terminal (or only in the GUI).

**User guidance.** Non-technical users rely on Kongtrol: guide a lot, in short lines.
Every command states what it's doing and suggests the next step; every error adds one
"what to do" line (never a raw OS/Go error alone); every wait says what's expected from
the user. Avoid jargon (daemon, API, token) or explain it in a few words. The daemon sends
phase + i18n key/args; clients render.

**No admin terminal (target).** Kongtrol must work entirely without an elevated terminal: all
or nothing, never "some commands need admin". Target design: a privileged helper service
installed once (`kongtrol service install`) runs only a closed list of network operations;
user-session parts (tray, CLI, dashboard, keychain, GUI automation of corporate clients) stay
unprivileged because services can't reach the user desktop or keychain.

### Data flow: connect lifecycle

Clients (CLI, tray, dashboard) call the daemon's local API; the daemon runs
`internal/app/profile_service.go`:

```
ProfileService.Connect(name)
  → resolves credentials from OS keychain (config.GetCredential), preflight
  → adapter.Connect(ctx, AdapterConfig)          // blocks until tunnel up (privileged work via privops)
  → watchdog.MarkActive(name)                    // re-arms reconnect
  → dnsMgr.OnConnect(name, info.DNS)             // applies DNS guard if enabled
  → policy routes + kill switch state updated
```

```
ProfileService.Disconnect(name)
  → watchdog.MarkIntended(name)                  // suppresses reconnect; kill switch won't block
  → adapter.Disconnect(ctx)
  → dnsMgr.OnDisconnect(name)                    // restores DNS when last tunnel drops
```

### Key subsystems

| Package | Responsibility |
|---|---|
| `internal/vpn` | `VPNAdapter` interface, `Status` type, registry, `Set` (the daemon's live adapter set, ADR-0053) |
| `internal/config` | YAML schema (`Config`, `VPNConfig`), loader (env expansion, `~/` paths), keychain access |
| `internal/policy` | IP longest-prefix match + domain glob → VPN profile name |
| `internal/app` | `ProfileService` (connect/disconnect lifecycle), `KillSwitchService`, `AdapterSync` (profiles applied without a restart) |
| `internal/routing` | Route manager (writes via privops) and no-admin route listing per OS |
| `internal/security` | Kill switch, DNS guard, blocked destinations (via privops), leak tester, HMAC-signed audit log |
| `internal/privops` | Closed list of privileged operations: `Local` (elevated) or `Client` (to the helper); per-OS `sys_*.go` |
| `internal/privhelper` | Privileged helper service (named pipe / unix socket, caller auth, lease, audit) |
| `internal/daemonctl` | Shared daemon client + spawn logic used by CLI and tray |
| `internal/vpn/appattach` | Attach mode shared by provider adapters |
| `internal/vpndetect` | Per-OS detection of VPN client binaries (wizard, preflight, doctor); `FileVersion` (Win32 version API, never launches the binary) |
| `internal/update` | Opt-in new version notice: GitHub Releases latest, ≤1 request/24 h, semver compare, never downloads (ADR-0034) |
| `internal/uninstall` | Uninstall plan per channel (install.manifest, Windows installer, package managers) and data purge with the exact keychain entries (ADR-0034) |
| `internal/doctor` | Diagnostics shared by `kongtrol doctor` and `GET /api/v1/doctor`: `Run(Options)` → translated checks, each with a "what to do" `Hint` and an optional `Fix` (only `helper_install`, offered by the GUI) |
| `internal/configexport` | `Write(w, cfg, lang, Style)`: the config as YAML that loads back unchanged (no secrets: passwords live in the keychain); `kongtrol export` and `GET /api/v1/config/export` |
| `internal/netstats` | No-admin per-interface traffic counters |
| `internal/monitor` | `Collector` (metrics snapshot), `Watchdog` (auto-reconnect with backoff), `DNSManager` (reference-counted DNS guard), `RecentConnections` (memory-only recent destinations, ADR-0054) |
| `internal/api` | Embedded HTTP server + REST handlers (tunnels, routes, policies, VPN profiles, groups, security toggles, settings, scheduler rules, audit) + WebSocket live feed; imports `web` package for `//go:embed` |
| `web/` | the dashboard `web/ui/` (see below), embedded by `web/ui_embed.go` and served at `/` |
| `assets/` | `assets/brand/` is the single source of logos (regenerate with `go run -tags brandgen assets/brand/generate.go`); `assets/embed.go`; tray / menu bar icons come from `brand/tray-<px>.png` (transparent masks that `generate.go` rasterizes from `source/logo-kong-solid.svg`, the solid glyph it derives from `source/logo-kong.svg`, ADR-0044), tinted and badged at run time by `TrayMark`/`TrayICO`/`TrayPNG` in `assets/icon.go` and `assets/state_icons.go` (dot size and gap: `BadgeGeometry`); the app icon (ADR-0052, the only logo with a background: Ink plate, solid mark in Bone) is generated too — `assets/logo.ico` (embedded in `kongtrol.exe` and `kongtrol-tray.exe` by `akavel/rsrc` into gitignored `rsrc_windows_<arch>.syso` in `make build` / `build-tray-native`, `release.yml`, `.goreleaser.yaml`; also `SetupIconFile`), `brand/app-icon-1024.png` (macOS `.icns`), `app-icon-512/256.png` (Linux), `app-icon-180.png` (landing touch icon) and the installer images `brand/installer/` |
| `landing/` | Public site (Vite + React + TS, multi-page, ES at `/…`, EN at `/en/…`), deployed by Cloudflare Workers Builds as static assets (`landing/wrangler.jsonc`, ADR-0022) |
| `cmd/kongtrol` | cobra CLI: `main.go` (commands + wiring) + `wizard.go` (`kongtrol init`) |
| `cmd/kongtrol-tray` | systray app; client of the shared daemon (spawns `kongtrol daemon` via `internal/daemonctl` if absent). Two exit items: "Salir y desconectar todo" (shutdown daemon + close tray) and "Cerrar ícono (las VPN siguen)" (close tray only). Left click on Windows opens the flyout (`flyout*.go`, `web/ui` `/tray`, ADR-0036); right click and other OSes keep the native menu. Native menu (ADR-0037): header → actions → version → exits; VPN lines are always "name · <state word>" (`tray_state.go`, never a raw status), connected ones checked and first, a DESIGN.md state dot per line; icon badge = aggregate state; per-OS icon code in `trayicon_<os>.go` |

### OS-specific files

Build tags control OS implementations — never use runtime `switch runtime.GOOS` for system-level operations:

- `internal/privops/sys_windows.go` / `sys_linux.go` / `sys_darwin.go` / `sys_unix.go` — all privileged per-OS work (routes, DNS, kill switch, WireGuard, OpenVPN, hosts)
- `internal/routing/windows.go` (route listing only) / `list_other.go`; `manager.go` is cross-OS via privops
- `internal/security/killswitch_priv.go` / `dnsguard_priv.go` are cross-OS via privops
- `internal/vpn/openvpn/process_windows.go` / `process_linux.go` / `process_darwin.go`
- `internal/vpn/wireguard/wg_windows.go` / `wg_unix.go` (+ `wg_priv.go` via privops)

### `//go:embed` constraints

Go embed paths are relative to the source file — no `../` allowed. Embed points:
- `web/ui_embed.go` embeds `ui/dist/`
- `assets/embed.go` embeds from `assets/brand/`
- Never commit duplicate brand files: web clients get logos from `assets/brand` (bundled by Vite from `@brand` or copied at build time).

### Dashboard (`web/ui/`)

Vite + React + TS (pnpm, oxlint, Vitest), the only dashboard (ADR-0043 replaced the classic
`web/dashboard`). Built to `web/ui/dist`, embedded by `web/ui_embed.go`, served at `/` by
`internal/api/ui_assets.go`: build files as they are, a missing file → 404, any other non-`/api`
path → `index.html` (client routes); `/app/*` and the classic page names (`/studio.html`, …)
answer 308 to the same page at `/` (the `#k=` fragment survives). Page CSP has no
`'unsafe-inline'`: no inline script or style in `web/ui`. Only `dist/.gitkeep` is tracked, so `go build`
works without Node (shows a "not built" page); release builds always build the UI. `make ui-dev`
proxies `/api` to the daemon. External URLs only in `src/links.ts`.

Architecture (ADR-0038); `docs/dashboard-parity.md` is the closed record of the migration:
- Routes: `react-router` data router (`createBrowserRouter`, `basename` `/`, ADR-0041) in
  `src/App.tsx`, pages loaded with route `lazy` (Inicio eager); sections in `src/app/sections.ts`.
  `/tray` is the tray flyout (ADR-0036), outside the shell.
- A form with edits calls `useUnsavedChanges(dirty)` (`src/app/unsaved.ts`): the tab-close prompt
  and the shell's "Tienes cambios sin guardar" dialog (`useBlocker`) follow from it.
- Data only via `src/data` (TanStack Query, cache keyed by resource in `keys.ts` or a
  per-domain `src/data/<domain>.ts`); the one WebSocket (`live.ts`) writes into the same cache.
  Pages never fetch on their own; changes use `useOptimisticMutation` (rollback + toast with the
  "what to do" line); errors render through `describeError` (on screen: kit `ErrorState`).
  Requests only through `http.get/post/put/delete` in `src/api/client.ts` (language header,
  auth, reachability, `ApiError`); data modules keep their endpoints next to their hooks.
- Auth only in `src/api/auth.ts` (no token handling anywhere else; 401 → "open from Kongtrol").
- Every "add" (profile, imported file, rule) opens the one shell-hosted "＋ Agregar" sheet
  (`src/components/add`, ADR-0055) through the deep link `?add=menu|vpn|file|rule[&via=&target=&app=]`
  (`src/app/addLink.ts` `useAddSheet`); never a page-local create form. New profiles always
  "Guardar y activar" (create; the sheet restarts itself only on `restart_required: true`);
  editing a connected profile saves with
  "Guardar y reconectar" (`PUT ?reconnect=true`). The sheet reads "Sitio, IP o programa" like
  `policy.ClassifyTargets` (shared table `internal/policy/testdata/classify_targets.json`).
- UI kit in `src/components/ui` (review at `/_kit`, dev builds only); shell in
  `src/components/layout`. Brand images are SVG/CSS only, theme-adaptive.
- Strings: `src/i18n/common.ts` (shared) + `src/i18n/pages/<page>.ts`, composed in
  `src/i18n/messages.ts`; ES+EN parity and no duplicate key across modules are tested.

### Config validation

`internal/config/schema.go` uses `go-playground/validator` tags. The `vpns` field has `validate:"required,min=1,dive"` — the `dive` tag is required for validator to recurse into `map[string]VPNConfig` values. Adding a new adapter type requires updating the `oneof` tag on `VPNConfig.Type`.

`config.Validate(cfg)` (in `internal/config/loader.go`) exposes the same struct-tag + semantic
validation `Load()` runs, for callers that mutate an already-loaded `*Config` in memory before
persisting it — used by the dashboard API's trial-then-commit pattern (build a modified copy,
`config.Validate(&trial)`, only write if it passes).

User-facing validation errors are `*config.ValidationError` (`internal/config/errors.go`): an
i18n key (`config.err.*`, with a `.fix` "what to do" line) + args; `Error()` stays English.
Clients render it via `config.LocalizeError`: CLI `localizeConfigErr`/`localizedErrText`
(`cmd/kongtrol/localize.go`), API `writeLocalizedError` (language from `X-Kongtrol-Lang`, then
`Accept-Language`; body adds `hint` + `error_key`). Provider, failover, policy, group and
scheduler checks use it, and so do the group/scheduler API errors (`api.err.*`); remaining
plain-English errors migrate the same way.

### Dashboard REST API + machine-local preferences

`internal/api/handlers_*.go` follow one repeated pattern for every dashboard CRUD endpoint (policies,
VPN profiles, groups, scheduler rules, settings): `s.loadRuntimeConfig()` → mutate a copy →
`config.Validate` → `s.saveRuntimeConfig()` (marshals YAML, hot-swaps the policy engine, calls
`onPolicyUpdate`). Security toggles (`POST /api/v1/security/killswitch|dnsguard`) additionally call
`s.onSecurityToggle`, which re-runs `applyKillSwitchState()`/`applyDNSGuardState()` in
`cmd/kongtrol/daemon_runtime.go` so the change takes effect immediately, not just on the next connect.

Profile rename (PUT with another `name`) moves every reference + keychain entries and is refused
while the tunnel is up; DELETE lists all references (`api.err.vpn_in_use`).
<!-- feature:profile_import -->
`POST /api/v1/vpns/import` and `kongtrol profile import` share `internal/profileimport`: a
`.conf`/`.ovpn` validated by the helper's own `privops` validators, refusals mapped to specific
`api.err.import_*` keys (hook, directive, script_security, file_ref, not_wireguard), stored 0600 in
`~/.kongtrol/configs/` (ADR-0042). A file only root can read (`/etc/wireguard/wg0.conf`) is never read
through the helper: `config.CheckFileReadable` → `config.err.file_root_only` (profile save, preflight,
doctor), and the user copies it with one OS prompt — CLI/wizard `sudo`, dashboard `{"path","elevate":true}`
→ `pkexec` — to the hidden `kongtrol profile read-protected`, which releases it only after
`profileimport.ReadProtected` (absolute, .conf/.ovpn, no symlink, link count 1, size cap, helper validators, `.ovpn` needs `remote` + `client`/`tls-client`; every refusal is the generic `api.err.import_protected_path`, never a file detail) (ADR-0058, ADR-0063). Audit entries carry `signed`/`verified` (key stays in the daemon); the audit logger and
the scheduler are reconfigured live on every save (`applyAuditConfig`, `applySchedulerConfig`).
<!-- feature:policies -->
Policy create/delete and profile create live in `internal/configedit` (`AddPolicy`, `RemovePolicy`,
`AddVPNProfile`/`SaveVPNProfile`, `Save` = Validate + policy engine + atomic write), shared by the
API handlers and the CLI with no daemon (`kongtrol policy add|remove`, `kongtrol profile add`,
`profile import --create`); with a daemon the CLI calls the same endpoints. `policy.ClassifyTargets`
is the "＋ Agregar" input classifier (URL → host, `*.host` unless `--no-subdomains`, IP/CIDR,
`app:`); web/ui mirrors it. `kongtrol profile detect` = `GET /api/v1/vpns/detect` (vpndetect).

<!-- feature:profile_live -->
VPN profile CRUD (and `POST /api/v1/policies/reload` after a hand edit) applies live (ADR-0053):
the daemon's adapters are a `vpn.Set` (copy-on-write map behind `atomic.Pointer`; readers never
lock, never index a raw map) and `app.AdapterSync.Sync(cfg)` runs after every save
(`commitRuntimeConfig` → `configedit.Save`): new profiles get an adapter (`app.NewProfileAdapter`),
removed ones are dropped (`Watchdog.Forget`), an edited one gets a new adapter only when its
adapter seed changed **and** it is disconnected. A busy profile is never replaced, renamed or
deleted (409 `api.err.vpn_rename_busy` / `vpn_delete_busy`); editing a connected one answers
`reconnect_required: true` and is applied by `Resync()` after its next disconnect.
`PUT /api/v1/vpns/{name}?reconnect=true` reconnects only that tunnel (`restartProfilesInPlace`);
never reconnect without the client asking. `kongtrol reload` asks per changed connected profile.
Responses carry `restart_required` (true only in setup mode or when an adapter cannot be built);
`POST /api/v1/restart` / `kongtrol restart` stay. `GET /api/v1/vpns/detect` and `profile detect`
add `active`/`interface` from `appattach.ActiveTunnel` (provider matchers registered with
`appattach.RegisterMatcher`; others `null`).

<!-- feature:recent_connections -->
Recent destinations (ADR-0054): `monitor.RecentConnections` reads the OS TCP table every 15 s
(`tcpConnections`, shared with PolicyWatch; macOS `supported:false`), memory only (500 entries,
1 h, emptied when off and on shutdown; never on disk, in logs or the audit log), host only from
`PolicyResolver.HostOf` (no reverse DNS). `GET /api/v1/connections/recent`, `kongtrol traffic
recent`, setting `monitor.recent_connections.enabled` (default on; `config recent-connections`,
settings API `recent_connections_enabled`).

Settings that are genuinely local to one machine — not the shared `kongtrol.yaml` — live in
`~/.kongtrol/preferences.json` instead (`cmd/kongtrol/preferences.go`): CLI display language,
favorites, default group, and the dashboard's own bind/port override
(`kongtrol config dashboard set-port/set-bind`, applied by `config.ApplyDashboardPreferences`
in both the CLI and the tray). This is intentionally **not** editable from the dashboard itself —
changing the port from the page serving that request would cut the connection mid-response.

### Credential rule

`AdapterConfig.Password` must be zeroed immediately after `Connect()` returns. Passwords are fetched from the OS keychain (`config.GetCredential`) immediately before calling `Connect()` — never stored in config structs at rest.

<!-- feature:watchdog -->
### Watchdog internals

`monitor.Watchdog` polls adapter `Status()` every 5 seconds per profile goroutine. Call `MarkIntended(name)` **before** `Disconnect()` to suppress reconnect. Call `MarkActive(name)` **after** a successful `Connect()` to re-arm it. Backoff: `2s × 2^attempt`, capped at 5 minutes.

- It only reconnects profiles that were brought up (`MarkActive`). **No implicit failover** (ADR-0018): profiles reach different networks and are never linked by name, type or `priority`. The only other profiles it may connect are those the user declared on the dropped profile, `vpns.<name>.failover: [backup, …]` (validated: exist, not itself, once, no cycles), tried in order once per outage; a backup that takes over pauses retries of the original until the user connects it again.
- No declared backup (or none worked): keep retrying the same profile and notify once per outage — `monitor.Event` kind `tunnel_watchdog`, keys `watchdog.event.*` (tray, dashboard toasts and Actividad, live view render them). Health-check failures only notify.
- `MarkIntended` cancels the watchdog's in-flight reconnect/failover for that profile; `ProfileService.Disconnect` also cancels and waits for any running `Connect` of the profile (`trackConnect`/`cancelConnects`), so `down` reports a final status and no provider app opens afterwards.
- WireGuard liveness (ADR-0030): an existing interface is not "connected". The adapter reads `privops.WGStats` every 10s and `healthTracker` decides `vpn.TunnelHealth` (`ok` / `idle` / `no_response` / `unknown`, `vpn.HealthReporter`, `TunnelMetrics.Health` json `health`). `no_response` (handshake older than 200 s with PersistentKeepalive, or tx growing with rx flat for 15 s, or never handshaked but sent something; 30 s grace after connect) makes `Status()` return `StatusError`, so watchdog reconnect/failover, kill switch and `on_tunnel_down` treat it like a drop; the reconnect replaces the tunnel (`wgUp` restart). An unused tunnel without keepalive is `idle`, never flagged. Stats unreadable (helper missing/outdated) → `unknown` + `ReasonKey`, status stays connected. The watchdog emits `no_response` once per outage and `responding` when the handshake is fresh again (`watchdog.event.*`); clients render health with `health.*` keys.
- `priority` is only the policy-engine tie-break (`policy.New`): higher number first for domain/app rules and equal-length IP prefixes; unset = position (earlier first).

### DNS manager internals

`monitor.DNSManager` is reference-counted across simultaneous tunnels. `OnConnect` merges + deduplicates DNS servers from all active profiles and applies the guard. `OnDisconnect` re-applies with the remaining set, or restores original DNS when the last profile disconnects. A tunnel that drops on its own (watchdog `reconnect_attempt` / `manual_reconnect_required`) calls `Release(profile)` unless `KillSwitchService.HoldsAllTraffic` (full tunnel held by the kill switch); the reconnect re-applies it (ADR-0058). On Linux with systemd-resolved the helper sets DNS per tunnel link (`resolvectl dns`/`domain ~.`, undone with `revert`, dropped by resolved with the link) instead of rewriting `/etc/resolv.conf`. Always call `ForceRestore()` on the daemon's SIGTERM/panic path (`daemon_cmd.go`); `up` is only a client of the daemon.

<!-- feature:openvpn -->
### OpenVPN adapter internals

- Binary resolution (`FindBinary` in `internal/vpn/openvpn/binary*.go`): profile `binary_path` (must exist) → `PATH` → Windows registry `HKLM\SOFTWARE\OpenVPN\exe_path` / `%ProgramFiles%\OpenVPN\bin\openvpn.exe`; on Unix `/usr/sbin`, `/usr/local/sbin`, `/opt/homebrew/sbin`.
- openvpn runs with `--cd <config dir>`, so relative `ca`/`cert`/`key`/`tls-crypt` paths inside the `.ovpn` resolve like OpenVPN GUI. Self-contained configs need no `auth.cert`/`auth.key`.
- Server-pushed routes are always accepted; Kongtrol adds policy routes on top (see Policy routing) and skips a destination the server already routes on the tunnel interface.
- `binary_path` reaches the adapter via `AdapterConfig.Extra`. `TunnelInfo` reports `InterfaceName` and pushed DNS.
- Runs via the privileged helper when not elevated (`privops.OpenVPNStart`); Kongtrol answers the management password prompt; interface/DNS come from the helper log tail, falling back to an IP match.
- An openvpn that exits on its own (killed, crashed) is reaped on the next `Connect`/`Disconnect` (`reapExitedLocked`: management connection closed, helper session stopped), so the watchdog can reconnect instead of getting "already connected".

<!-- feature:helper -->
### Privileged helper internals

- All privileged ops go through `privops.Default()` (`Local` when elevated, `Client` to the helper otherwise; `ErrHelperNotInstalled` → `GuidanceKey`). Installed once with `kongtrol service install`.
- GUI install (always one OS permission prompt, never silent elevation): tray item "Instalar ayudante (una vez)" ("Actualizar ayudante (una vez)" when `helper/status` says `outdated`) on every OS, also in the first-run menu; after success the tray notifies "Ayudante listo" and hides the item at once; with no daemon answering the tray pings the helper itself, and tells the user once (`tray.helper_first_run`). The dashboard's Diagnóstico card offers "Reparar" for checks with `fix: helper_install`; both use `POST /api/v1/helper/install`.
- The protocol is versioned (`privops.ProtocolVersion`; bump it on any wire change — an older installed helper is reported as outdated and the user re-runs `kongtrol service install`; the Windows installer runs `service install --if-outdated` on every upgrade, ADR-0036) and carries a per-process lease (renew 5s, expire 20s); expiry undoes OpenVPN, DNS, hosts, blocked destinations, kill switch (fail-open, so a crashed daemon never leaves the user offline). WireGuard and routes aren't leased.
- `block.set` (`SetBlockedDestinations`) replaces the whole set of blocked outbound prefixes: Windows Firewall BLOCK rules `KongtrolBlock`, Linux `KONGTROL_BLOCK` iptables/ip6tables chains (REJECT), macOS pf anchor `com.kongtrol` table. Only block rules: in Windows Defender Firewall a block rule always beats an allow rule, so "block all + allow exceptions" cannot work — a full-tunnel block is sent as the complement of its exceptions. An older helper answers "unknown operation" → `ErrHelperOutdated`.
- `wg.stats` (`WGStats`, protocol 4) is read-only: `wg show <iface> dump` for a tunnel the helper manages, or the caller's own profile tunnel (same checks as `wg.remove`, nothing adopted). Only peer public key, latest handshake, rx/tx and keepalive leave the helper (`ParseWGDump` drops private/preshared keys; a failure never echoes the output). Successful read-only ops are not audited (polled every 10s); failures are.
- The helper only runs root/admin-owned binaries; Homebrew binaries are refused (user-swappable dylibs), so macOS OpenVPN via the helper needs a bundled signed openvpn (pending).

<!-- feature:killswitch feature:policies feature:routes -->
### Policy routing, kill switch, on_tunnel_down

The user decides which tunnels are up; a policy whose tunnel is off uses the normal internet unless `on_tunnel_down` says otherwise.
- `monitor.PolicyResolver` resolves policy domains (tunnel up or not), installs OS routes for every adapter type through the tunnel interface (WireGuard also gets AllowedIPs), follows tunnels via `TunnelProbe` (attach mode, reconnects) and removes routes when the tunnel goes down. Overlaps follow the policy engine order. Resolved IPs expire after 6h unseen.
- `app.KillSwitchService` owns the blocked set: a protected tunnel that dropped unexpectedly (was up, user did not run `down` — `ProfileService` reports intent via `OnIntent`) blocks only its captured routes + policy destinations; a full-tunnel profile blocks everything but endpoints and `neverBlock` (broadcast/multicast/link-local); the directly connected subnets and private ranges stay open only with `allow_lan` / legacy `mode: loose` (otherwise the LAN router, the usual DNS fallback, would leak lookups, ADR-0063). Split-tunnel blocks always exempt the directly connected subnets. Policies with `on_tunnel_down: block` add theirs. A connected tunnel's destinations are never blocked. States `off`/`armed`/`active` (`security.KillSwitchStatus`, rendered with `killswitch.state.*`). For domains the block covers only IPs resolved so far. When a dropped tunnel starts being blocked, `SetOnBlock` emits one notice per outage (`killswitch.event.blocking` / `blocking_full` / `blocking_full_lan` when the LAN is blocked too, kind `tunnel_watchdog`) with what to do (ADR-0058).
- `monitor.PolicyWatch` checks open TCP connections (Windows GetExtendedTcpTable, Linux /proc/net/tcp; macOS unsupported) every 15s and emits `monitor.Event`s (i18n key + args, throttled 30 min per policy; `connect` rate-limited 3 min) served by `GET /api/v1/events?after=<seq>` to tray, dashboard (toasts + Actividad, `web/ui/src/data/home.ts`) and the live view.

### Public IP lookup (opt-in)

- `security.LeakTester` (`internal/security/leaktest.go`) is the only code that asks an external service for the public IP. Off by default: `Config.PublicIPCheckEnabled()` = `security.public_ip_check.enabled` or the legacy `security.leak_detection.enabled` (its old `interval`/`action` keys are gone, ADR-0042). The kill switch and DNS guard never enable or need it. Leak check: the public IP must equal `integrity_check.expected_ips` of a **connected** profile (`LeakTesterOptions.Active`); a leak only notifies (alert + audit), never blocks.
- Never periodic: `Trigger()` after a successful connect/disconnect (`cmd/kongtrol/daemon_runtime.go`; ≥30s apart, coalesced, 3s settle) and `CheckNow()` on demand (`kongtrol ip`, `POST /api/v1/security/publicip/check`, dashboard "Revisar", tray line). Disabled → no request, `LastResult()` nil, clients show "—" + `cli.public_ip.enable_hint`.
- Endpoints: `security.public_ip_check.endpoints` (https only, `config.ValidatePublicIPEndpoint`), default `https://api.ipify.org`; 5s timeout, body ≤64 bytes, must parse as an IP. Toggle: `POST /api/v1/security/publicip`, `kongtrol config public-ip on|off`, wizard, dashboard Security page. Enabling shows `cli.public_ip.notice` ("Usa un servicio externo (<host>) que ve tu IP pública").

### Provider attach mode internals

- `internal/vpn/appattach`: shared by `forticlient`, `protonvpn`, `nordvpn`, `surfshark`. Each adapter embeds an `appattach.Session` and builds an `appattach.Provider` per call: `App` (launch locations), `Matcher` (interface name/description globs, `Exclude`, `RequireIPv4`, macOS `Services` for `scutil --nc list`), `Location`, optional `AutoConnect` / `Disconnect` funcs (nil = none on this OS).
- `Session.Connect`: (1) interface already up → `PhaseAttached`; (2) `auto_connect` + `AutoConnect` → wait `AutoWait` → `PhaseAuto`; any failure falls through; (3) launch app → `PhaseWaitingUser` (Status `connecting`, `Reason` = short automation error) → wait `wait_timeout` (default 5m) → `PhaseUser`, or `vpn.ErrUserActionTimeout`. No lock is held while waiting.
- `Session.Disconnect`: `Disconnect` func then wait for the drop; otherwise launch app, `PhaseWaitingDisconnect`, return `vpn.ErrManualDisconnect` (Status stays `connected` until the interface drops). `ProfileService` runs DNS / kill-switch cleanup in the background once it drops; the API answers 202 `{"status":"waiting_user"}`; `daemonctl.Client.Disconnect` maps that back to `ErrManualDisconnect`; `kongtrol down` polls and shows the message.
- `Session.Status`: interface detection on every call; an external tunnel becomes `PhaseAttached`; an unrequested drop becomes `PhaseDropped` when `Capabilities().ManualReconnect` (no automatic path, or `auto_connect: false`). The watchdog then emits `manual_reconnect_required` once per drop instead of reconnecting; the daemon turns it into the notice `watchdog.event.manual_reconnect` (tray, both dashboards, live view).
- User-facing state: `vpn.AttachState` (`AttachReporter`) → `monitor.TunnelMetrics.Attach` (`json:"attach"`) in `/api/v1/tunnels` and the WebSocket snapshot. Clients render `AttachState.MessageKey()` with `internal/i18n` (`attach.*`) in their own language; the dashboard mirrors it in `attachKey()` (`web/ui/src/components/home/model.ts`). Never pre-render these strings in the daemon.
- Config: standard fields `auto_connect` (*bool, default true), `country`, `city`, `server` (exact), `interface`, `wait_timeout`, `app_path`; nordvpn-only `group`. `VPNConfig.AdapterExtra()` is the single mapping to `AdapterConfig.Extra`; `validateProvider` (`internal/config/semantics.go`) rejects location fields on providers that cannot select (surfshark, forticlient).
- FortiClient versions (ADR-0020, matrix in `docs/forticlient-support.md` — update it with every change): `DetectInstall` (folder from `app_path` / registry `INSTALLDIR`, `FortiClient.exe` ProductVersion via the Win32 version API, `FortiVPN.exe` presence) → `SelectPlan` (pure, one test case per matrix row): `FortiVPN.exe` → `fortivpn-cli` (`--cli --connect --tunnel` + native auth dialog, contributor-reported); 6.x or unknown → `gui-6.4` (verified on 6.4.10 only); other versions → `attach` (no automation). Never click a layout nobody measured; never pass credentials on a command line (`FortiSSLVPNclient.exe -u user:pass` is rejected for that). `kongtrol doctor` shows version · strategy · status.
- FortiClient GUI automation (`internal/vpn/forticlient/script.go` generates, `gui_windows.go` runs): the script text holds no user data — FortiClient path, tunnel, username and password go on stdin as base64 lines (a typographic quote also ends a PowerShell single-quoted string); booleans render as `$true`/`$false`; written as UTF-8 **with BOM** (Windows PowerShell 5.1 reads a BOM-less script in the ANSI code page, which broke parsing and mangled non-ASCII text); golden files in `testdata/` (`go test ./internal/vpn/forticlient -run Golden -update` after a reviewed change) and the Windows PowerShell parser test (fails, not skips, in CI); its failure becomes one line (`scriptErrorLine`: the script's `ERROR:` line or PowerShell's message + script line, never the source excerpt); cancelling the connect kills the script and clears the clipboard.
- Interface names: FortiClient, NordVPN (NordLynx, TAP-NordVPN) and Proton (TAP-ProtonVPN, Linux proton0/pvpn*) are documented or known; Surfshark ("Surfshark DCO adapter" per its docs, Linux `surfshark*`) and the macOS Network Extension services are best effort, so `interface:` overrides them.

<!-- feature:i18n -->
<!-- feature:tray_flyout -->
### Tray flyout (ADR-0036, ADR-0056)

- Left click on the tray icon (Windows, WebView2 Runtime present) opens a frameless topmost
  window next to the icon showing `web/ui` `/tray`; right click, macOS, Linux and a missing
  runtime keep the native menu. Portable rules in `cmd/kongtrol-tray/flyout.go` (tested), the
  window in `flyout_windows.go` (`github.com/jchv/go-webview2/pkg/edge`, pure Go: the tray
  still builds with `CGO_ENABLED=0`).
- Only `http://<loopback>:<port>/tray` loads (`flyoutURL`); requests outside the daemon's
  origin get 403; an injected script pins the page and exposes `window.kongtrolTray.post` with
  a per-launch nonce. Page → tray messages are the closed list in `flyoutMessages`
  (`web/ui/src/tray/bridge.ts` mirrors it); add one in both places and in the tests.
- The page uses the same API and cookie as the dashboard; window-only controls ("Salir",
  Escape) appear only when the bridge exists, so `/tray` also works in a browser tab.
- ADR-0056: the WebView2 controller's `IsVisible` follows the window on every show/hide
  (a window created hidden never paints otherwise: the empty flyout of #40). `ready` means
  "on screen" (`announceReady`: visible + one painted frame), sent after load and after every
  `kongtrol:shown`; a show without it in 3 s (`flyoutWatch`) closes the flyout, opens the
  native menu, logs the cause in `tray.log`, and keeps the left click on the native menu
  until the tray restarts.

## Internationalization (i18n)

All user-visible strings — prompts, error messages, status output, banners — must be internationalized using `internal/i18n`. **Never hardcode strings directly in user-facing code.**

Rules:
- Add new keys to **both** `ES` and `EN` maps in `internal/i18n/i18n.go`
- In wizard/CLI code use `i18n.T(lang, key)` or `i18n.F(lang, key, args...)` — never `fmt.Printf("hardcoded string")`
- `ES` and `EN` are both required (keys in both maps); the language comes from the user's choice or the OS (non-Spanish → `EN`). A new language means a new map, not special cases
- Language resolution (CLI, wizard, tray): `KONGTROL_LANG` → saved choice in `preferences.json` → OS language (`i18n.System()`, build-tagged). The wizard doesn't ask; it ends with a one-line hint to change it (`kongtrol config lang <es|en>`). Wizard output goes through `w.t(key)` / `w.tF(key, args...)`. The dashboard follows the browser's language and `prefers-color-scheme` until the user picks one explicitly.

## Build on Windows

`make` targets use `SHELL := /usr/bin/bash` and Unix env-var syntax — they must be run from **Git Bash**, not from PowerShell or cmd.exe. Running `make build-all-cli` from PowerShell will fail with "The syntax of the command is incorrect".
