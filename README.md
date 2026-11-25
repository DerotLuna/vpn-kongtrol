<!-- README version 2.6.0 · 2026-11-25.
     Theme-aware logo: GitHub supports <picture> with prefers-color-scheme sources
     (docs.github.com/en/get-started/writing-on-github/getting-started-with-writing-and-formatting-on-github/quickstart-for-writing-on-github#adding-an-image-to-suit-your-visitors). -->
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="assets/brand/logo-kong-on-dark.svg" />
  <source media="(prefers-color-scheme: light)" srcset="assets/brand/logo-kong.svg" />
  <img src="assets/brand/logo-kong.svg" alt="Kongtrol logo: a monkey face drawn in concentric lines" width="96" align="left" />
</picture>

# Kongtrol

**All your VPNs under Kongtrol.** Connect your VPNs the way you always do. Kongtrol watches them,
reconnects them if they drop, and sends each site through the right one.

<sub>The program you install is <code>kongtrol</code> (formerly "VPN Kongtrol"). Public beta · README v2.2.0 · 2026-11-19.</sub>

<br clear="left" />

<div align="center">
  <p>
    <img alt="Go 1.27+" src="https://img.shields.io/badge/Go-1.27+-00ADD8?style=flat-square&logo=go" />
    <img alt="Windows, macOS and Linux" src="https://img.shields.io/badge/Platforms-Windows%20%7C%20macOS%20%7C%20Linux-informational?style=flat-square" />
    <img alt="License: Apache 2.0" src="https://img.shields.io/badge/License-Apache%202.0-green?style=flat-square" />
  </p>

  <p><a href="https://kongtrol.app/en/"><strong>kongtrol.app</strong></a> ·
  <a href="https://kongtrol.app/en/guide">Setup guide</a> ·
  <a href="https://github.com/DerotLuna/vpn-kongtrol/releases/latest">Download</a> ·
  <a href="https://kongtrol.app">Español</a></p>
</div>

<p align="center">
  <img src="assets/readme/hero.gif" width="640" alt="Kongtrol's terminal on kongtrol.app: the KONGTROL banner appears, the command kongtrol up --group work is typed, and three tunnels come up one by one (office on FortiClient, dev-server and aws on OpenVPN) with their IPs and uptime, followed by kill switch armed and DNS guard active." />
  <br/><sub><em>One command, three tunnels up, protection on. Recorded from <a href="https://kongtrol.app/en/">kongtrol.app</a>.</em></sub>
</p>

## What it is

Kongtrol runs several VPNs at once on the same computer and keeps them from fighting over routes
and DNS. **If you are not technical:** keep using your VPN apps (NordVPN, Proton VPN, Surfshark,
FortiClient…) as always; Kongtrol notices the connection, shows it in the tray and the browser
panel, reconnects it if it drops and protects you while it is down, with no terminal. **If you are
a developer or SRE:** reach client networks, WireGuard, OpenVPN, Tailscale or Cloudflare WARP at the
same time, send each destination (IP range or domain) through the right tunnel, and automate it all
from a CLI with the same features as the app. Free, open source, no account, and it never sends
your data anywhere.

## What's new

**v0.6.0:**

- **Desktop app, no terminal needed:** Windows installer, macOS app, `.deb`/`.rpm` packages and a
  tray icon. On first run the browser panel opens to create your first VPN.
- **Diagnostics and export in the panel:** check your computer, repair the network helper and
  download your configuration (without passwords).
- **WireGuard shows "no answer from the server"** when a tunnel stops getting replies, instead of
  looking connected.
- **Clean uninstall** (`kongtrol uninstall`, `--purge` to also delete your data) and an **optional
  new version notice** (off by default).
- **Clearer commands** ([ADR-0033](docs/adr/0033-cli-command-restructure.md)): `policy list` /
  `policy show`, `ip`, `routes --system`, `config default-group`, `config killswitch` and
  `config dns-guard`, `reload [profile]`; `kongtrol init` offers to install the network helper.

**v0.5.0:** 10 VPNs (NordVPN and Surfshark joined), attach mode for VPN apps, and
Kongtrol keeps running in the background after you close the window.

<!-- feature:killswitch feature:dns_guard feature:watchdog -->
## Features

<p align="center">
  <img src="assets/readme/routing.gif" width="640" alt="Routing diagram from kongtrol.app: four destinations on the left (10.10.0.0/16, *.amazonaws.com, claude.ai, gmail.com) take turns lighting up and a dashed line runs through the Kongtrol logo to the tunnel it uses on the right (office on FortiClient, aws on OpenVPN, us-content on Proton VPN, direct with no VPN); a line below names the matching rule." />
  <br/><sub><em>Each destination, its rule, its tunnel. Recorded from <a href="https://kongtrol.app/en/#map">kongtrol.app</a>.</em></sub>
</p>

- **Several VPNs at once.** Each one keeps its own routes and DNS; Kongtrol settles the conflicts.
- **Routing per destination.** Rules by IP range or domain send traffic through the tunnel you
  choose, on every VPN type. Rules inherit their profile's action while the tunnel is off.
  Allow uses the available connection, block protects known destinations, and connect also tries
  to start the VPN if TCP traffic is detected. App rules provide diagnosis today.
- **Auto-reconnect.** The watchdog retries only the tunnel that dropped, three times (2 s, 30 s,
  2 min), then tells you once with a Retry action and tries again only when your network changes or
  the computer wakes. After your first VPN it asks once whether to reconnect, when Kongtrol starts
  (sign-in, restart, after a crash), the VPNs you had connected; it waits for the network first and
  only picks up a provider app's VPN its app already connected. Both are in Settings › When starting
  ([ADR-0098](docs/adr/0098-bounded-reconnect-and-restore-at-start.md),
  [ADR-0101](docs/adr/0101-guided-restore-at-start-and-network-wait.md)). It fails over only to backups you declared (`failover:`); it never
  brings up an unrelated VPN. A provider app it cannot reconnect on its own gets a notice to
  reconnect it there.
- **Kill switch and DNS guard** at the system firewall and network level (`status` shows what is
  blocked and the DNS servers in use), with a signed local audit
  log whose signatures are checked when you read it. Turn them on in the panel or with `kongtrol config killswitch|dns-guard on|off|status`. When a VPN drops on its own, only its destinations
  are blocked and its DNS goes back to the system, so other sites keep working; a VPN that carried
  all traffic stays blocked until it returns, and the notice has a "Get internet back" button
  (it disconnects that VPN); with "Allow the
  local network" off (`allow_lan: false`) that also cuts printers and NAS on your network. The kill
  switch is off by default; the first time a VPN drops on its own, Kongtrol offers once to turn it on.
- **Browser panel and tray icon.** One panel (the tray or `kongtrol dashboard` opens it) with seven sections: Home, Profiles (and groups), Rules (and schedules), Traffic, Security, Activity (audit log and notices), Settings
  (diagnostics, export, updates); compiled into the program, no external server. Tray icon.
- **Setup wizard** (`kongtrol init`): finds the VPN clients you have and saves passwords in the
  system keychain.
- **Diagnostics** (`kongtrol doctor`): checks programs, certificates, keychain, permissions and visible VPN default-route conflicts.
  before you connect, and says how to fix each problem.
- **No administrator terminal.** Network changes go through a small helper you install once.

Full status of every feature: [docs/ROADMAP.md](docs/ROADMAP.md).

<!-- feature:providers -->
## Supported VPNs

<p align="center">
  <img src="assets/readme/attach.gif" width="720" alt="Attach demo from kongtrol.app in three panels: in 'Your VPN app' the Connect button is pressed and the shield turns green, Kongtrol's radar says 'tunnel detected · watching', and the tray menu shows '1 VPN connected' with the Kongtrol icon turning green in the taskbar." />
  <br/><sub><em>You connect in your VPN app as usual; Kongtrol detects the tunnel and takes over. Recorded from <a href="https://kongtrol.app/en/#start">kongtrol.app</a>.</em></sub>
</p>

**You connect in the app, Kongtrol attaches.** Kongtrol detects the tunnel and handles status,
reconnection, DNS and rules. *auto* = Kongtrol can also connect it for you; *manual* = you connect
in the app, Kongtrol watches it.

| VPN | `type` | Windows | macOS | Linux |
|---|---|---|---|---|
| NordVPN | `nordvpn` | auto | manual | auto |
| Proton VPN | `protonvpn` | manual | manual | auto |
| Surfshark | `surfshark` | manual | manual | manual |
| Mullvad VPN | `mullvad` | auto | CLI / interface | auto |
| Private Internet Access | `pia` | auto | CLI / interface | auto |
| ExpressVPN | `expressvpn` | manual / visible adapter | CLI / interface | auto |
| Windscribe | `windscribe` | auto | CLI / interface | auto |
| CyberGhost VPN | `cyberghost` | manual | manual / interface | manual / interface |
| FortiClient ([support by version](docs/forticlient-support.md)) | `forticlient` | auto | yes | yes |

**Kongtrol connects directly** with your config file or the official client.

| VPN | `type` | Windows | macOS | Linux |
|---|---|---|---|---|
| WireGuard | `wireguard` | yes | yes | yes |
| OpenVPN | `openvpn` | yes | partial¹ | yes |
| Tailscale | `tailscale` | yes | yes | yes |
| Cloudflare WARP | `cloudflarewarp` | yes | yes | yes |
| Cisco AnyConnect / Secure Client | `ciscoanyconnect` | yes | yes | yes |
| GlobalProtect | `globalprotect` | yes | yes | no |

¹ OpenVPN on macOS needs, for now, to run as administrator.
Proton VPN also works without its app as `type: wireguard` / `type: openvpn` with the files it lets you download.

<sub>Names are trademarks of their respective owners. Kongtrol is an independent project, not affiliated with them.</sub>

<!-- feature:distribution -->
## Install

**Desktop app.** Download from [Releases](https://github.com/DerotLuna/vpn-kongtrol/releases/latest):

| System | File |
|---|---|
| Windows | `Kongtrol-Setup-windows-amd64.exe` — per user, no admin; asks once to install the network helper. `kongtrol-windows-amd64.exe` is only the terminal command (a double click shows where to get the installer) |
| macOS | `Kongtrol-macos-universal.zip` — the app is not notarized (Apple charges $99/year): open it once with right-click › Open |
| Linux | `.deb` / `.rpm` for amd64 and arm64 |

**Or one line** (the app with its tray icon and the `kongtrol` command, on every system; run it again to update):

```bash
# macOS / Linux
curl -fsSL https://kongtrol.app/install.sh | sh
```

```powershell
# Windows (PowerShell)
iwr https://kongtrol.app/install.ps1 -useb | iex
```

Only the `kongtrol` command, without the tray: `sh -s -- --cli-only` (macOS/Linux) or
`$env:KONGTROL_CLI_ONLY=1` before the PowerShell line (or `-CliOnly`). A Linux machine with no
desktop session (a server, SSH) and Windows Server Core get only the command by themselves, and
the script says so. Updates keep your choice; `--with-tray` / `$env:KONGTROL_CLI_ONLY=0` adds the
tray later. If Kongtrol was installed with `Kongtrol-Setup`, `install.ps1` installs nothing and
tells you to update with the new `Kongtrol-Setup` instead (one copy, one updater).

The scripts check the download against `checksums.txt` and verify the
Sigstore signature of `checksums.txt` when [cosign](https://github.com/sigstore/cosign) 2.4+ is
installed (a bad signature stops the install); without cosign they say so and print the command
to verify by hand. See [Verifying releases](docs/SECURITY.md#verifying-releases).

Package managers (winget, Scoop, Homebrew, AUR, npm) are not published yet.

> **No administrator terminal.** Routing, DNS and firewall changes go through a small network
> helper (a system service) installed once: `kongtrol service install`. Your system asks for
> permission that one time. The Windows installer, `kongtrol init` and
> `kongtrol up` offer it for you.

> **Antivirus or SmartScreen warning?** Windows files are not signed yet (the free SignPath
> Foundation application is pending), and programs that change routes, DNS and firewall get
> flagged on sight. It is a known false positive. Check the download
> ([checksums, Sigstore, build provenance](docs/SECURITY.md#verifying-releases)) or build from
> source. If Defender quarantines `kongtrol.exe` (for example `Trojan:Win32/Bearfoos.A!ml`), the
> installers stop and keep your current version; restore it in Windows Security > Protection
> history only after checking it, and report it at https://www.microsoft.com/wdsi/filesubmission.
> More in [docs/SECURITY.md](docs/SECURITY.md#unsigned-binaries-and-antivirus-false-positives).

Build from source (Go 1.27+): `git clone https://github.com/DerotLuna/vpn-kongtrol && cd vpn-kongtrol && make build`
(output: `build/dist/kongtrol`). More in [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md).

<!-- feature:first_run -->
## First run

**With the app**: open Kongtrol. The panel opens in your
browser to create your first VPN ("Set up now…" in the tray icon opens it again). After saving,
use the same tab to connect or create a rule. The tray updates its menu when the profile is saved.

**From the terminal** (same features):

```bash
kongtrol init           # wizard: finds your VPN clients, saves passwords in the keychain
kongtrol up office      # connect a profile (or: --group work, --all)
kongtrol status         # see each tunnel's state, IP and traffic
kongtrol dashboard      # open the panel in your browser (already signed in)
```

Step-by-step guide: [kongtrol.app/en/guide](https://kongtrol.app/en/guide)
(Español: [kongtrol.app/guia](https://kongtrol.app/guia)). Several VPNs, groups and rules:
[docs/SETUP.md](docs/SETUP.md).

## Daily use

- Kongtrol keeps working in the background (the "daemon": the Kongtrol process that stays on after
  you close the terminal, the tray or the panel). Closing a window **never** disconnects.
- Only `kongtrol down` or the tray's "Quit and disconnect all" disconnects your VPNs; "Close icon
  (VPNs keep running)" only hides the tray icon.
<!-- feature:profile_live -->
- Added, renamed or deleted a profile (panel, `kongtrol profile add` or by hand + `kongtrol reload`)?
  It works at once, no restart. Edited a VPN that is connected? It keeps its old settings until you
  reconnect it: the panel offers "Guardar y reconectar", `kongtrol reload` asks; other VPNs are never
  touched.
<!-- feature:add_sheet -->
- New profile in the panel ("＋ Agregar"): "Guardar y activar" saves it and connects it; "Guardar"
  only saves it, like `kongtrol profile add` (connect it later with `kongtrol up <name>`).
<!-- feature:app_routing -->
In a compatible direct WireGuard profile on Linux, turn on **Executable egress** and add
app-only rules with full executable paths. Destination policies keep their priority
unless the profile explicitly takes all traffic from those programs.
[Setup and limits](docs/CONFIGURATION.md#executable-routing-linux)

<!-- feature:default_egress -->
- On Windows or Linux, edit a compatible direct WireGuard profile to use it for traffic without a more specific route.
  Check the saved file with `kongtrol routes check-default <profile>`. Disconnect the profile
  before changing the setting, then connect to apply it. More-specific routes retain priority.
<!-- feature:recent_connections -->
- Where did my traffic go? `kongtrol traffic recent` lists the destinations of the last hour, the
  program and which VPN each went through (Windows and Linux). Memory only, never saved or sent;
  `kongtrol config recent-connections off` turns it off. Narrow it with `--search <text>` (site, IP or
  program) and `--via <profile|direct>`; the dashboard's Tráfico › Recientes has the same filters.
- Something wrong? `kongtrol doctor`, then `kongtrol logs`.

## Commands

`kongtrol --help` lists every command by group, in English or Spanish; `kongtrol <command> --help`
explains it with the next step and examples. Full reference: [docs/CLI.en.md](docs/CLI.en.md)
(Español: [docs/CLI.md](docs/CLI.md)).

| Command | What it does |
|---|---|
| `kongtrol up [profile…]` | Connect profiles (`--group`, `--all`, `--fav`) and open the live view |
| `kongtrol down [profile…]` | Disconnect profiles (`--group`, `--all`) |
| `kongtrol status [-w]` | State, IP and traffic of each tunnel |
| `kongtrol routes [profile] [--system]` | What goes through each tunnel; `--system`: routes Kongtrol added |
| `kongtrol dashboard` | Open the panel in your browser |
| `kongtrol init` | Setup wizard |
| `kongtrol policy list` | Which VPN each rule uses |
| `kongtrol policy show <target\|app:<exe>>` | Which VPN a destination or app uses, and why |
| `kongtrol policy test <file>` | Test your rules against a list of destinations |
| `kongtrol config validate` | Check `kongtrol.yaml` without connecting |
| `kongtrol config default-group [group]` | Show or set the group `kongtrol up` connects |
| `kongtrol config killswitch\|dns-guard <on\|off\|status>` | Kill switch and DNS guard |
| `kongtrol config public-ip <on\|off>` | Optional public IP lookup |
| `kongtrol config update-check <on\|off\|status>` | Optional new version notice |
| `kongtrol config recent-connections <on\|off\|status>` | Recent destinations list (memory only) |
| `kongtrol config autostart <on\|off\|status>` | Open Kongtrol when you sign in |
| `kongtrol config restore-tunnels <on\|off\|status>` | Reconnect the VPNs you had connected when Kongtrol starts (off until you turn it on; asked once after your first VPN) |
| `kongtrol config lang <es\|en>` · `config favorites` · `config dashboard` | Language, favorites, panel address on this computer |
| `kongtrol reload [profile]` | Apply `kongtrol.yaml` changes without disconnecting |
| `kongtrol restart` | Restart Kongtrol and reconnect what was connected |
| `kongtrol export` | Print your config without passwords to share it |
| `kongtrol doctor [--fix]` | Check your computer and say how to fix each problem |
| `kongtrol traffic recent [--limit n] [--search text] [--via profile\|direct]` | Destinations of the last hour: program, VPN or direct |
| `kongtrol ip` | Look up your public IP now and check for leaks |
| `kongtrol logs` | Show Kongtrol's events |
| `kongtrol service install\|uninstall\|status` | Network helper (no admin terminal) |
| `kongtrol uninstall [--purge\|--data-only]` | Uninstall Kongtrol (your data stays on this computer unless you ask to delete it) |
| `kongtrol version` | Show the version |

## Privacy: local only

Kongtrol sends no telemetry, analytics or crash reports. The panel, the tray and the CLI talk to
Kongtrol on your own computer (`127.0.0.1`); nothing leaves it. Passwords live in your system
keychain, never in files. Only two calls go outside, both **off by default**:

- **Public IP check** (`kongtrol config public-ip on`): asks `api.ipify.org` (or the service you
  choose) for your public IP after a tunnel connects or disconnects, and when you ask. That service
  sees your IP.
- **New version notice** (`kongtrol config update-check on`):
  asks api.github.com at most once a day. GitHub sees your IP and `kongtrol/<version>`.

Every connection Kongtrol makes: [docs/SECURITY.md](docs/SECURITY.md#network-connections-kongtrol-makes).
Privacy policy: [kongtrol.app/en/privacy](https://kongtrol.app/en/privacy).

<!-- feature:update_check -->
## Updating

Install the new version the same way you installed it: run the one-line script or the Windows
installer again, or install the new `.deb`/`.rpm`. Your settings in `~/.kongtrol`, passwords in the
keychain and the network helper stay. If you already have it, the Windows installer and the
scripts detect it and say "Update Kongtrol <old> → <new>" (or reinstall, or warn before an older
version) without asking again what you already chose; the helper asks for permission only if it
is out of date. The Windows installer and the scripts reconnect the VPNs you had connected by
themselves; after a package update (`.deb`/`.rpm`) run
`kongtrol restart` (or Restart in the panel) to
switch to the new version with your VPNs reconnected. The new version notice
is off by default: turn it on in the panel (Settings) or with
`kongtrol config update-check on`. It never downloads or installs anything.

<!-- feature:uninstall -->
## Uninstalling

Your settings and saved passwords **stay** unless you ask, so a reinstall brings everything back.
Windows installer: Settings › Apps › Kongtrol › Uninstall (it asks whether to delete your data; default No).
Elsewhere (including `install.ps1`, whose tray, shortcut and autostart it removes too): `kongtrol uninstall` (`--purge` also deletes `~/.kongtrol` and the saved passwords,
`--dry-run` shows the plan). To only wipe your data and keep the app: panel › Settings › "Delete all
my data", or `kongtrol uninstall --data-only`.

## Documentation

| Doc | Covers |
|---|---|
| [Setup guide](https://kongtrol.app/en/guide) · [ES](https://kongtrol.app/guia) | Install and first VPN, step by step |
| [docs/SETUP.md](docs/SETUP.md) | Guided walkthrough: several VPNs, groups, rules |
| [docs/CLI.en.md](docs/CLI.en.md) · [ES](docs/CLI.md) | Every command and flag (generated from the CLI) |
| [docs/CONFIG.en.md](docs/CONFIG.en.md) · [ES](docs/CONFIG.md) | Every `kongtrol.yaml` key with type and default (generated from the schema) |
| [docs/CONFIGURATION.md](docs/CONFIGURATION.md) | `kongtrol.yaml` guide with examples |
| [docs/SECURITY.md](docs/SECURITY.md) | Network connections, network helper, kill switch, DNS guard, recovery |
| [docs/API.md](docs/API.md) | Local API used by the tray and the panel |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | System design, project layout, adding a VPN adapter |
| [docs/DOCKER.md](docs/DOCKER.md) | Headless / server use |
| [docs/ROADMAP.md](docs/ROADMAP.md) | Feature status and roadmap |
| [docs/adr/](docs/adr/README.md) · [docs/research/](docs/research/README.md) | Decisions and the research behind them |

## Security

Found a vulnerability? Please report it privately as described in [SECURITY.md](SECURITY.md)
(supported versions, scope). Summary for users: [kongtrol.app/en/security](https://kongtrol.app/en/security).

## Contributing

Build, test, lint, docs and release commands are in [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md);
rules for contributors (and coding agents) are in [AGENTS.md](AGENTS.md). The GIFs in this README
are recorded from the landing page: `node scripts/readme-media.mjs`.

## License

Apache License 2.0 — see [LICENSE](LICENSE) and [NOTICE](NOTICE). Terms of use: [kongtrol.app/en/terms](https://kongtrol.app/en/terms).

Made with ♥ by [Rubén Luna](https://github.com/DerotLuna).
