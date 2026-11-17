<div align="center">
  <img src="assets/brand/logo-512.png" alt="VPN Kongtrol" width="120" />
  <h1>VPN Kongtrol</h1>
  <p><strong>Multi-VPN orchestration for professionals.</strong><br/>
  Policy-based routing · Security enforcement · Live monitoring — from a single binary.</p>

  <p><sub>Kongtrol is the short name. The binary you install is <code>kongtrol</code>.</sub></p>

  <p>
    <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go" />
    <img src="https://img.shields.io/badge/Platforms-Windows%20%7C%20Linux%20%7C%20macOS-informational?style=flat-square" />
    <img src="https://img.shields.io/badge/License-Apache%202.0-green?style=flat-square" />
  </p>

  <p><a href="https://kongtrol.app"><strong>kongtrol.app</strong></a> · <a href="https://kongtrol.app/en/guide">Setup guide</a> · <a href="https://github.com/DerotLuna/vpn-kongtrol/releases/latest">Download</a></p>
</div>

---

## What's new

- **New home: [kongtrol.app](https://kongtrol.app)** — the website moved from GitHub Pages; old links redirect.
- **10 VPNs** — NordVPN and Surfshark join the list.
- **Attach mode** — connect as usual in your VPN app (NordVPN, Proton VPN, Surfshark, FortiClient); Kongtrol detects the tunnel and takes over status, auto-reconnect, DNS and routing rules.
- **Runs in the background** — `kongtrol up` starts it, closing the window does not disconnect, and the tray icon offers "Quit and disconnect all" or "Close icon (VPNs keep running)".

## What is VPN Kongtrol

**VPN Kongtrol** (short name: *Kongtrol*) is an open-source multi-VPN orchestrator and VPN
manager for Windows, Linux, and macOS. It runs several VPN tunnels at once — WireGuard,
OpenVPN, FortiClient, Cisco AnyConnect, ProtonVPN, GlobalProtect, Tailscale, and Cloudflare
WARP — routes traffic per destination with policy-based routing, and enforces a kill switch
and DNS guard at the OS level.

## The problem

Working with multiple VPNs simultaneously is painful: each client overrides your routing table when it connects, they fight each other for DNS, and you're manually reconnecting and rerouting traffic dozens of times a day — with zero visibility into what's actually going where.

**Kongtrol solves this.** One orchestrator manages all your VPN connections, routes traffic by destination (IP ranges, domains, or apps), watches for unexpected drops and reconnects automatically, and enforces security at the OS level — from a CLI, tray icon, or browser dashboard.

<!-- feature:killswitch feature:dns_guard -->
## Features

<div align="center">
  <img src="assets/switchboard-demo.gif" alt="Kongtrol policy engine resolving a destination to the correct VPN tunnel in real time" width="720" />
  <p><em>One destination. One rule. The right tunnel.</em></p>
</div>

- **Policy-based routing** by IP range and domain for WireGuard, with priority-based conflict resolution; app rules currently provide flow resolution and diagnostics, not OS-level per-process enforcement
- **8 built-in adapters** — FortiClient, OpenVPN, ProtonVPN, Cisco AnyConnect, WireGuard, GlobalProtect, Tailscale, Cloudflare WARP
- **Auto-reconnect watchdog** with exponential backoff and priority failover between profiles
- **Kill switch + DNS guard** enforced at the OS firewall/network layer, with a signed audit log
- **Embedded web dashboard** — full management UI (profiles, groups, policies, live security toggles, settings, audit log, traffic charts), plus a system tray app; no external server, no Node.js, compiled into the binary
- **Setup wizard** (`kongtrol init`) that auto-detects installed VPN clients and stores credentials in the OS keychain
- **Diagnostics** (`kongtrol doctor`) that validate your whole stack before you connect

See [docs/ROADMAP.md](docs/ROADMAP.md) for the full feature/status matrix.

<!-- feature:providers -->
## Supported VPN adapters

| Adapter | Type key | Platforms |
|---|---|---|
| FortiClient 6.4.x | `forticlient` | Win / Linux / macOS |
| OpenVPN | `openvpn` | Win / Linux / macOS |
| ProtonVPN | `protonvpn` | Win / Linux / macOS |
| Cisco AnyConnect / Secure Client | `ciscoanyconnect` | Win / Linux / macOS |
| WireGuard | `wireguard` | Win / Linux / macOS |
| GlobalProtect (Palo Alto) | `globalprotect` | Win / macOS |
| Tailscale | `tailscale` | Win / Linux / macOS |
| Cloudflare WARP | `cloudflarewarp` | Win / Linux / macOS |

## Quick start

Install with one command (downloads the latest release, verifies its checksum):

```bash
# Linux / macOS
curl -fsSL https://raw.githubusercontent.com/DerotLuna/vpn-kongtrol/main/install.sh | sh
```

```powershell
# Windows (PowerShell)
iwr https://raw.githubusercontent.com/DerotLuna/vpn-kongtrol/main/install.ps1 -useb | iex
```

Or grab the binary for your platform from [Releases](https://github.com/DerotLuna/vpn-kongtrol/releases), or build from source (requires Go 1.25+):

```bash
git clone https://github.com/DerotLuna/vpn-kongtrol
cd vpn-kongtrol
make build   # → build/dist/kongtrol
```

> **Windows:** run your shell as **Administrator** for `kongtrol init`, `up`, `down`, and `doctor` — they need elevated permissions for routing, DNS, and firewall.

> **Antivirus/SmartScreen warning?** Kongtrol is unsigned (no paid code-signing cert — this is an
> unfunded open-source project) and touches routing/DNS/firewall, which AV heuristics flag on sight.
> It's a known false positive, not malware. Verify the download against the release's
> `checksums.txt`, or build from source and judge the code yourself. See
> [docs/SECURITY.md](docs/SECURITY.md#unsigned-binaries-and-antivirus-false-positives).

```bash
kongtrol init           # interactive wizard: detects clients, stores credentials in OS keychain
kongtrol up office aws  # connect one or more profiles
kongtrol status         # check tunnel states
kongtrol dashboard      # open the web UI at localhost:9741
```

`kongtrol --help` lists every command by group (Connection, Setup, Diagnostics, System), in Spanish
or English, and `kongtrol <command> --help` shows what it does, the next step and examples. Full list
of commands and flags: **[docs/CLI.en.md](docs/CLI.en.md)** (Español: [docs/CLI.md](docs/CLI.md)).

For a full guided walkthrough (multiple VPNs, profile groups, routing policies), see **[docs/SETUP.md](docs/SETUP.md)**.

## Documentation

| Doc | Covers |
|---|---|
| [docs/SETUP.md](docs/SETUP.md) | Guided first-run walkthrough, from install to your first policy |
| [docs/CLI.en.md](docs/CLI.en.md) · [ES](docs/CLI.md) | Full command reference (generated from the CLI) |
| [docs/CONFIG.en.md](docs/CONFIG.en.md) · [ES](docs/CONFIG.md) | Every `kongtrol.yaml` key with type and default (generated from the schema) |
| [docs/CONFIGURATION.md](docs/CONFIGURATION.md) | `kongtrol.yaml` guide with examples, including ProtonVPN modes |
| [docs/API.md](docs/API.md) | Local API used by the tray and dashboards |
| [docs/DOCS-TOOLING.md](docs/DOCS-TOOLING.md) | How docs stay current: generators, impact map, link check |
| [docs/adr/](docs/adr/README.md) | Architecture decisions (ADRs) |
| [docs/research/](docs/research/README.md) | Research notes behind those decisions, with sources |
| [SECURITY.md](SECURITY.md) | Security policy: how to report a vulnerability, supported versions, scope |
| [docs/SECURITY.md](docs/SECURITY.md) | Security architecture: network connections Kongtrol makes, privileged helper and leases, kill switch, DNS guard, recovery procedures |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | System design, project layout, adding a new VPN adapter |
| [docs/DOCKER.md](docs/DOCKER.md) | Headless / server deployment |
| [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) | Build, test, lint, and release commands |
| [docs/ROADMAP.md](docs/ROADMAP.md) | Feature status matrix and roadmap |

## License

Apache License 2.0 — see [LICENSE](LICENSE).
