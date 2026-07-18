<div align="center">
  <img src="assets/logo.svg" alt="Kongtrol" width="120" />
  <h1>Kongtrol</h1>
  <p><strong>Multi-VPN orchestration for professionals.</strong><br/>
  Policy-based routing · Security enforcement · Live monitoring — from a single binary.</p>

  <p>
    <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go" />
    <img src="https://img.shields.io/badge/Platforms-Windows%20%7C%20Linux%20%7C%20macOS-informational?style=flat-square" />
    <img src="https://img.shields.io/badge/License-Apache%202.0-green?style=flat-square" />
  </p>
</div>

---

## The problem

Working with multiple VPNs simultaneously is painful: each client overrides your routing table when it connects, they fight each other for DNS, and you're manually reconnecting and rerouting traffic dozens of times a day — with zero visibility into what's actually going where.

**Kongtrol solves this.** One orchestrator manages all your VPN connections, routes traffic by destination (IP ranges, domains, or apps), watches for unexpected drops and reconnects automatically, and enforces security at the OS level — from a CLI, tray icon, or browser dashboard.

## Features

- **Policy-based routing** by IP range, domain, or app executable, with priority-based conflict resolution
- **8 built-in adapters** — FortiClient, OpenVPN, ProtonVPN, Cisco AnyConnect, WireGuard, GlobalProtect, Tailscale, Cloudflare WARP
- **Auto-reconnect watchdog** with exponential backoff and priority failover between profiles
- **Kill switch + DNS guard** enforced at the OS firewall/network layer, with a signed audit log
- **Embedded web dashboard** and system tray app — no external server, no Node.js, compiled into the binary
- **Setup wizard** (`kongtrol init`) that auto-detects installed VPN clients and stores credentials in the OS keychain
- **Diagnostics** (`kongtrol doctor`) that validate your whole stack before you connect

See [docs/ROADMAP.md](docs/ROADMAP.md) for the full feature/status matrix.

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

Download the latest release for your platform from [Releases](../../releases), or build from source (requires Go 1.22+):

```bash
git clone https://github.com/vpn-kongtrol/kongtrol
cd vpn-kongtrol
make build   # → build/dist/kongtrol
```

> **Windows:** run your shell as **Administrator** for `kongtrol init`, `up`, `down`, and `doctor` — they need elevated permissions for routing, DNS, and firewall.

```bash
kongtrol init           # interactive wizard: detects clients, stores credentials in OS keychain
kongtrol up office aws  # connect one or more profiles
kongtrol status         # check tunnel states
kongtrol dashboard      # open the web UI at localhost:9741
```

For a full guided walkthrough (multiple VPNs, profile groups, routing policies), see **[docs/SETUP.md](docs/SETUP.md)**.

## Documentation

| Doc | Covers |
|---|---|
| [docs/SETUP.md](docs/SETUP.md) | Guided first-run walkthrough, from install to your first policy |
| [docs/CLI.md](docs/CLI.md) | Full command reference |
| [docs/CONFIGURATION.md](docs/CONFIGURATION.md) | Full `kongtrol.yaml` reference, including ProtonVPN modes |
| [docs/SECURITY.md](docs/SECURITY.md) | Kill switch, DNS guard, watchdog, and recovery procedures |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | System design, project layout, adding a new VPN adapter |
| [docs/DOCKER.md](docs/DOCKER.md) | Headless / server deployment |
| [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) | Build, test, lint, and release commands |
| [docs/ROADMAP.md](docs/ROADMAP.md) | Feature status matrix and roadmap |

## License

Apache License 2.0 — see [LICENSE](LICENSE).
