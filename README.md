<div align="center">
  <img src="assets/logo.png" alt="Kongtrol" width="120" />
  <h1>Kongtrol</h1>
  <p><strong>Multi-VPN orchestration for professionals.</strong><br/>
  Policy-based routing · Security enforcement · Live monitoring — from a single binary.</p>

  <p>
    <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go" />
    <img src="https://img.shields.io/badge/Platforms-Windows%20%7C%20Linux%20%7C%20macOS-informational?style=flat-square" />
    <img src="https://img.shields.io/badge/License-MIT-green?style=flat-square" />
  </p>
</div>

---

## The Problem

Working with multiple VPNs simultaneously is painful:

- **FortiClient** overrides your entire routing table when it connects
- **OpenVPN** has two configs — one for the dev server, another for AWS
- **ProtonVPN** for geo-restricted content fights the others for DNS
- Each VPN stomps on the others' routes; you disconnect and reconnect manually dozens of times a day
- Zero visibility into what traffic goes where

**Kongtrol solves this.** One orchestrator manages all your VPN connections, routes traffic by destination (IP ranges, domains), watches for unexpected drops and reconnects automatically, and enforces security at the OS level — all from a CLI, tray icon, or browser dashboard.

---

## Features

### Core
- **Policy-based routing** — define which IP ranges or domains route through which VPN
- **Multi-VPN coexistence** — run compatible VPNs simultaneously with isolated routes
- **Auto-reconnect watchdog** — detects unexpected drops and reconnects with exponential backoff
- **8 built-in adapters** — FortiClient, OpenVPN, ProtonVPN, Cisco AnyConnect, WireGuard, GlobalProtect, Tailscale, Cloudflare WARP
- **Profile groups** — define named sets of profiles (`work`, `travel`) and connect them all with one command
- **Setup wizard** — `kongtrol init` detects installed clients, preserves existing config, stores credentials in the OS keychain
- **Diagnostics** — `kongtrol doctor` validates your full stack (binaries, certs, keychain, permissions) before you connect
- **Config export** — `kongtrol export` generates a sanitized config template for teammates (no secrets)

### Security
- **Kill switch** — OS-level firewall rules block all traffic the moment a tunnel drops
- **DNS guard** — forces DNS through the active tunnel; restores original config on disconnect
- **Leak detection** — automated IP/DNS leak check every N seconds with configurable action
- **Signed audit log** — append-only, HMAC-SHA256 per entry; key stored in OS keychain
- **Encrypted credentials** — passwords and keys in Windows Credential Manager / macOS Keychain / Linux Secret Service; never in the config file

### Monitoring
- **Embedded dashboard** — dark-themed web UI at `localhost:9741`, compiled into the binary (no external server, no Node.js)
- **Live WebSocket feed** — per-tunnel bandwidth, uptime, assigned IP, DNS servers
- **System tray app** — tunnel status and profile connect/disconnect from the menu bar
- **Alert system** — notifications on VPN drop, leak detected, or high latency

---

## Supported VPN Adapters

| Adapter | Type key | Platforms | Notes |
|---|---|---|---|
| FortiClient 6.4.x | `forticlient` | Win / Linux / macOS | CLI + passive detection fallback for EMS-locked installs |
| OpenVPN | `openvpn` | Win / Linux / macOS | Multi-instance, dynamic management port per tunnel |
| ProtonVPN | `protonvpn` | Win / Linux / macOS | JSON API (v3.10+) with human-readable fallback |
| Cisco AnyConnect / Secure Client | `ciscoanyconnect` | Win / Linux / macOS | Stdin pipe for credentials |
| WireGuard | `wireguard` | Win / Linux / macOS | `wg-quick` on Unix, `wireguard.exe` service on Windows |
| GlobalProtect (Palo Alto) | `globalprotect` | Win / macOS | SSO detection fallback |
| Tailscale | `tailscale` | Win / Linux / macOS | Auth key + optional exit node |
| Cloudflare WARP | `cloudflarewarp` | Win / Linux / macOS | Requires prior `warp-cli register` |

---

## Architecture

```
┌──────────────────────────────────────────────────────────────────────┐
│                          KONGTROL CORE                               │
│                                                                      │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │                        POLICY ENGINE                           │  │
│  │   IP longest-prefix match · domain glob · priority resolution  │  │
│  └───────────────────────────────┬────────────────────────────────┘  │
│                                  │                                   │
│  ┌───────────────────────────────▼────────────────────────────────┐  │
│  │                      VPN ADAPTER LAYER                         │  │
│  │                                                                │  │
│  │  FortiClient  OpenVPN  ProtonVPN  AnyConnect  WireGuard       │  │
│  │  GlobalProtect          Tailscale          Cloudflare WARP    │  │
│  │                                                                │  │
│  │  Common interface: Connect · Disconnect · Reconnect · Status  │  │
│  └───────────────────────────────┬────────────────────────────────┘  │
│                                  │                                   │
│  ┌───────────────────────────────▼────────────────────────────────┐  │
│  │                       ROUTE MANAGER                            │  │
│  │   Windows: netsh + iphlpapi    Linux: netlink     macOS: route │  │
│  └────────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  ┌─────────────────────────┐   ┌──────────────────────────────────┐  │
│  │     SECURITY LAYER      │   │       MONITOR / WATCHDOG         │  │
│  │  • Kill Switch (OS FW)  │   │  • Auto-reconnect (backoff)      │  │
│  │  • DNS Guard            │   │  • Health checks per tunnel      │  │
│  │  • Leak Detection       │   │  • Bandwidth / uptime metrics    │  │
│  │  • Audit Log (HMAC)     │   │  • Reference-counted DNS guard   │  │
│  └─────────────────────────┘   └──────────────────────────────────┘  │
│                                                                      │
│  ┌─────────────────────────┐   ┌──────────────────────────────────┐  │
│  │    WEB DASHBOARD        │   │        SYSTEM TRAY               │  │
│  │    localhost:9741       │   │   Status · Connect · Disconnect  │  │
│  │    Go embed · no deps   │   │   Cross-platform (Win/Mac/Linux) │  │
│  └─────────────────────────┘   └──────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────────┘
```

---

## Quick Start

### 1. Install

Download the latest release for your platform from [Releases](../../releases):

| Archive | Platform |
|---|---|
| `kongtrol_windows_amd64.zip` | Windows 64-bit |
| `kongtrol_linux_amd64.tar.gz` | Linux 64-bit |
| `kongtrol_linux_arm64.tar.gz` | Linux ARM64 |
| `kongtrol_darwin_amd64.tar.gz` | macOS Intel |
| `kongtrol_darwin_arm64.tar.gz` | macOS Apple Silicon |

Or build from source (requires Go 1.22+):

```bash
git clone https://github.com/yourorg/vpn-kongtrol
cd vpn-kongtrol
make build          # current platform
make build-all      # all platforms → build/dist/
```

### 2. Run the setup wizard

```bash
kongtrol init
```

The wizard:
- Detects VPN clients installed on your system
- Shows existing profiles if a config file is found
- Offers to refresh credentials for existing profiles
- Guides you through adding new profiles (only asks fields relevant to that adapter)
- Stores all passwords in the **OS keychain** — never in the YAML file
- Validates the resulting config before writing

### 3. Connect and go

```bash
kongtrol up office aws     # connect one or more profiles
kongtrol status            # check tunnel states
kongtrol dashboard         # open the web UI
```

---

## CLI Reference

```bash
# Setup & diagnostics
kongtrol init                        # interactive wizard (create or update config)
kongtrol doctor                      # validate full stack: binaries · certs · keychain · permissions

# Tunnel lifecycle
kongtrol up <profile> [profile...]   # connect one or more profiles
kongtrol up --group work             # connect all profiles in a group
kongtrol down <profile>              # disconnect a profile
kongtrol down --group work           # disconnect all profiles in a group
kongtrol down --all                  # disconnect everything

# Status & diagnostics
kongtrol status                      # table: profile · status · IP · uptime
kongtrol status --watch              # live auto-refreshing terminal view
kongtrol check                       # run leak + integrity test now

# Routes
kongtrol routes list                 # list Kongtrol-managed routes

# Dashboard
kongtrol dashboard                   # start web UI and open browser

# Config
kongtrol config validate             # validate kongtrol.yaml without connecting
kongtrol export                      # print sanitized config template (no secrets)

# Audit log
kongtrol audit                       # audit log subcommands
```

---

## Configuration

```yaml
# ~/.kongtrol/kongtrol.yaml
# Run 'kongtrol init' to generate this file interactively.
# Passwords are NEVER stored here — use 'kongtrol init' or
# 'kongtrol config set-credential <profile> password' to store them.

vpns:
  office:
    type: forticlient
    version: "6.4"
    host: vpn.empresa.com
    port: 443
    tunnel_name: "Office"
    auth:
      method: certificate+credentials
      cert: ~/.kongtrol/certs/office.crt
      key:  ~/.kongtrol/certs/office.key
      username: your_username
      password_keychain: office.password   # stored in OS keychain
    priority: 10

  dev-server:
    type: openvpn
    config: ~/.kongtrol/configs/server.ovpn
    auth:
      method: certificate
      cert: ~/.kongtrol/certs/server.crt
      key:  ~/.kongtrol/certs/server.key
    priority: 20

  aws:
    type: openvpn
    config: ~/.kongtrol/configs/aws.ovpn
    auth:
      method: certificate
      cert: ~/.kongtrol/certs/aws.crt
      key:  ~/.kongtrol/certs/aws.key
    priority: 20

  us-content:
    type: protonvpn
    server: US
    protocol: wireguard
    auth:
      method: credentials
      username_keychain: proton.username
      password_keychain: proton.password
    priority: 5

  # Additional adapters — add any combination:
  #
  # wg-home:
  #   type: wireguard
  #   config: ~/.kongtrol/configs/wg-home.conf
  #   auth: { method: certificate }
  #
  # corporate-cisco:
  #   type: ciscoanyconnect
  #   host: vpn.corp.com
  #   auth: { method: credentials, username: you, password_keychain: cisco.password }
  #
  # tailscale-mesh:
  #   type: tailscale
  #   auth: { method: credentials }       # reuses existing 'tailscale login' session
  #
  # warp:
  #   type: cloudflarewarp
  #   auth: { method: credentials }       # run 'warp-cli register' once first


policies:
  - name: "Office servers"
    match:
      ip_ranges: ["10.10.0.0/16", "192.168.50.0/24"]
    via: office

  - name: "AWS workloads"
    match:
      ip_ranges: ["172.31.0.0/16", "10.200.0.0/16"]
      domains:   ["*.amazonaws.com", "*.aws.empresa.com"]
    via: aws

  - name: "Dev server"
    match:
      ip_ranges: ["185.0.0.0/32"]     # replace with actual IP
    via: dev-server

  - name: "US geo-restricted content"
    match:
      domains: ["netflix.com", "*.netflix.com", "hulu.com"]
    via: us-content

  # Unmatched traffic → physical interface (no VPN).
  # To force everything else through a VPN, add:
  # - name: "Default"
  #   match: { ip_ranges: ["0.0.0.0/0"] }
  #   via: office


security:
  kill_switch:
    enabled: true
    mode: strict        # strict = block ALL traffic on drop; loose = allow LAN
    allow_lan: true

  dns_guard:
    enabled: true
    fallback_dns: "1.1.1.1"

  leak_detection:
    enabled: true
    interval: "60s"
    action: notify      # notify | killswitch_and_notify

  audit_log:
    path: ~/.kongtrol/audit.log
    max_size_mb: 100
    sign: true          # HMAC-SHA256 per entry


monitor:
  enabled: true
  dashboard:
    port: 9741
    bind: "127.0.0.1"  # never expose to 0.0.0.0 without adding auth

  health_check:
    interval: "30s"
    timeout:  "10s"

  alerts:
    on_vpn_drop:
      actions: ["notify", "log"]
    on_leak_detected:
      actions: ["kill_switch", "notify", "log"]
    on_high_latency_ms: 500
    on_reconnect_attempts: 3


# Profile groups — connect/disconnect multiple profiles with one command.
# kongtrol up --group work   →  connects office + dev-server
# kongtrol up --group travel →  connects warp + us-content
groups:
  work:
    profiles: [office, dev-server]
  travel:
    profiles: [warp, us-content]
  full:
    profiles: [office, dev-server, aws]
```

---

## Security Model

| Layer | Windows | Linux | macOS |
|---|---|---|---|
| **Kill Switch** | `netsh advfirewall` block + tunnel allow | `iptables OUTPUT -j DROP` + ACCEPT | `pf` anchor at `/etc/pf.anchors/kongtrol` |
| **DNS Guard** | `netsh interface ip set dns` per interface | `/etc/resolv.conf` rewrite (backup kept) | `networksetup -setdnsservers` per service |
| **Credentials** | Windows Credential Manager | libsecret / D-Bus Secret Service | macOS Keychain |
| **Audit log** | HMAC-SHA256 per entry; signing key in OS keychain | ← same | ← same |
| **Dashboard** | Localhost-only bind (`127.0.0.1`) | ← same | ← same |

### DNS Guard recovery

If Kongtrol crashes without cleanly restoring DNS, run:

```bash
# Windows
netsh interface ip set dns "Ethernet" dhcp

# Linux
sudo cp /etc/resolv.conf.kongtrol.bak /etc/resolv.conf

# macOS
networksetup -setdnsservers Wi-Fi empty
```

### Auto-reconnect watchdog

The watchdog polls every 5 seconds per profile. On unexpected disconnect it waits `2s → 4s → 8s → … → 5min` (exponential backoff) before each retry. Intentional disconnects via `kongtrol down` suppress reconnect automatically — the watchdog only fires on drops you didn't request.

---

## Dashboard

The web UI is compiled into the binary with `//go:embed`. No web server required.

```
open http://localhost:9741
```

```
┌─ KONGTROL ─────────────────────────────────────────────────────────┐
│                                                                     │
│  TUNNELS                                                            │
│  ┌────────────┬─────────────┬──────────┬──────────┬──────────┐    │
│  │  PROFILE   │   STATUS    │  ↑ MB/s  │  ↓ MB/s  │  UPTIME  │    │
│  ├────────────┼─────────────┼──────────┼──────────┼──────────┤    │
│  │  office    │  ● ACTIVE   │   2.3    │   8.1    │  4h 22m  │    │
│  │  aws       │  ● ACTIVE   │   0.1    │   0.4    │  4h 22m  │    │
│  │  us-content│  ○ IDLE     │   —      │   —      │    —     │    │
│  └────────────┴─────────────┴──────────┴──────────┴──────────┘    │
│                                                                     │
│  ACTIVE ROUTES                    SECURITY STATUS                  │
│  10.10.0.0/16      → office       ✓ Kill Switch    ON              │
│  172.31.0.0/16     → aws          ✓ DNS Guard      ON              │
│  *.amazonaws.com   → aws          ✓ Leak Check     CLEAN           │
│  *.netflix.com     → us-content   ✓ Audit Log      ACTIVE          │
│  (other)           → direct                                        │
│                                                                     │
│  LAST EVENTS                                                        │
│  14:02  office connected (10.10.0.1)                               │
│  14:03  aws connected (172.31.4.7)                                  │
│  14:05  leak check passed — no leaks                               │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Docker

For headless / server deployments. The container requires elevated network privileges to manage kernel tunnel interfaces.

```bash
docker build -f build/docker/Dockerfile -t kongtrol .

docker run -d \
  --name kongtrol \
  --privileged \
  --cap-add NET_ADMIN \
  --device /dev/net/tun \
  --network host \
  -v ~/.kongtrol:/etc/kongtrol:ro \
  -p 127.0.0.1:9741:9741 \
  kongtrol
```

Or with Compose:

```bash
docker compose -f build/docker/docker-compose.yml up -d
```

> **Note:** `--privileged` and `--cap-add NET_ADMIN` are required for tunnel and routing management. Never expose port 9741 externally without adding authentication.

---

## Adding a New VPN Adapter

The adapter interface is the only contract:

```go
type VPNAdapter interface {
    Connect(ctx context.Context, cfg AdapterConfig) error
    Disconnect(ctx context.Context) error
    Reconnect(ctx context.Context) error
    Status() Status
    TunnelInfo() (*TunnelInfo, error)
    Name() string
    Version() string
    Capabilities() Capabilities
}
```

Steps:

1. Create `internal/vpn/<name>/adapter.go` — implement the interface
2. Register via `init()`:
   ```go
   func init() {
       vpn.Register("myvpn", func() vpn.VPNAdapter { return &Adapter{} })
   }
   ```
3. Add a blank import in `cmd/kongtrol/main.go`:
   ```go
   _ "github.com/vpn-kongtrol/kongtrol/internal/vpn/myvpn"
   ```
4. Add `type: myvpn` as a valid value in `internal/config/schema.go`
5. Add the adapter entry in `configs/example.yaml`

No changes to routing, security, monitoring, or policy engine required.

---

## Development

```bash
# Test
go test ./...
make test

# Build current platform
make build

# Cross-compile (CGO_ENABLED=0 — pure Go, no C deps)
make build-all-cli

# Build tray app (requires CGO — native build only)
make build-tray-native

# Docker
make docker-build
make docker-up

# Release (requires goreleaser)
make release
```

### Project layout

```
vpn-kongtrol/
├── assets/                        # Brand assets
│   ├── logo.png                   # Source logo (1024×1024)
│   ├── embed.go                   # //go:embed logo.png → assets.LogoPNG
│   └── icon.go                    # TrayIcon(size) — resizes for systray
│
├── cmd/
│   ├── kongtrol/                  # CLI binary
│   │   ├── main.go                # cobra root + all commands
│   │   └── wizard.go              # kongtrol init interactive wizard
│   └── kongtrol-tray/             # Tray app binary
│       └── main.go
│
├── internal/
│   ├── vpn/
│   │   ├── adapter.go             # VPNAdapter interface + Status type
│   │   ├── registry.go            # Register() / New() / Registered()
│   │   ├── ciscoanyconnect/
│   │   ├── cloudflarewarp/
│   │   ├── forticlient/
│   │   ├── globalprotect/
│   │   ├── openvpn/
│   │   ├── protonvpn/
│   │   ├── tailscale/
│   │   └── wireguard/
│   │
│   ├── routing/
│   │   ├── manager.go             # RouteManager interface
│   │   ├── windows.go             # netsh
│   │   ├── linux.go               # netlink
│   │   └── darwin.go              # /sbin/route
│   │
│   ├── policy/
│   │   ├── engine.go              # ResolveIP / ResolveDomain
│   │   └── rules.go               # Rule · MatchSpec · ParseRule
│   │
│   ├── security/
│   │   ├── killswitch*.go         # OS implementations
│   │   ├── dnsguard*.go           # OS implementations
│   │   ├── leaktest.go
│   │   └── audit.go
│   │
│   ├── monitor/
│   │   ├── collector.go           # Aggregate metrics snapshot
│   │   ├── watchdog.go            # Auto-reconnect with backoff
│   │   └── dnsmanager.go          # Reference-counted DNS guard
│   │
│   ├── config/
│   │   ├── schema.go              # Config structs + validation tags
│   │   ├── loader.go              # YAML load + env expansion + defaults
│   │   └── keychain.go            # OS keychain read/write
│   │
│   └── api/
│       ├── server.go
│       ├── handlers.go            # REST: tunnels · routes · security
│       └── ws.go                  # WebSocket live metrics
│
├── web/
│   ├── assets.go                  # //go:embed dashboard
│   └── dashboard/
│       ├── index.html
│       ├── app.js
│       ├── style.css
│       └── logo.png
│
├── configs/
│   └── example.yaml
│
├── build/
│   ├── docker/
│   │   ├── Dockerfile
│   │   └── docker-compose.yml
│   └── ...
│
├── .goreleaser.yaml
├── Makefile
├── go.mod
└── go.sum
```

---

## Project Status

| Component | Status |
|---|---|
| VPNAdapter interface + registry | ✅ |
| Config schema · loader · env expansion · `~/` paths | ✅ |
| OS keychain (Win Credential Manager / macOS Keychain / libsecret) | ✅ |
| FortiClient 6.4.x (CLI + passive fallback for EMS-locked installs) | ✅ |
| OpenVPN (multi-instance · dynamic management port) | ✅ |
| ProtonVPN CLI (JSON v3.10+ · human-readable fallback) | ✅ |
| Cisco AnyConnect / Secure Client | ✅ |
| WireGuard (wg-quick / wireguard.exe) | ✅ |
| GlobalProtect (SSO detection fallback) | ✅ |
| Tailscale (auth key · exit node) | ✅ |
| Cloudflare WARP | ✅ |
| Route manager (Windows netsh · Linux netlink · macOS route) | ✅ |
| Policy engine (IP longest-prefix · domain glob · priority) | ✅ |
| Kill switch (WFP · iptables · pf) | ✅ |
| DNS guard (netsh · resolv.conf · networksetup) | ✅ |
| DNS guard wired into connect/disconnect lifecycle | ✅ |
| Auto-reconnect watchdog (exponential backoff) | ✅ |
| Leak detection (periodic · configurable action) | ✅ |
| Audit log (append-only · HMAC-SHA256 signed) | ✅ |
| Metrics collector (status · bandwidth · uptime) | ✅ |
| Embedded web dashboard (dark theme · logo · live WebSocket) | ✅ |
| System tray app (logo icon · per-profile menu) | ✅ |
| `kongtrol init` wizard (detect · preserve existing · keychain) | ✅ |
| Docker + docker-compose (privileged · /dev/net/tun) | ✅ |
| goreleaser multi-platform pipeline | ✅ |
| Unit + integration tests (60+ tests across all packages) | ✅ |
| `kongtrol doctor` (full stack diagnostics) | ✅ |
| Profile groups (`kongtrol up --group work`) | ✅ |
| `kongtrol status --watch` (live terminal view) | ✅ |
| `kongtrol export` (sanitized config template for teammates) | ✅ |
| Per-app routing rules (experimental) | 🔲 |

---

## Roadmap

### v0.1 — Core ✅
FortiClient · OpenVPN · ProtonVPN · Route manager · Policy engine · Basic CLI

### v0.2 — Security ✅
Kill switch · DNS guard · Leak detection · Audit log

### v0.3 — Dashboard & Tray ✅
Embedded web dashboard · WebSocket feed · System tray · Brand logo

### v0.4 — Adapters & Resilience ✅
Cisco AnyConnect · WireGuard · GlobalProtect · Tailscale · Cloudflare WARP · Auto-reconnect watchdog · DNS guard lifecycle integration

### v1.0 ✅
`kongtrol init` wizard · OS keychain · goreleaser pipeline · Integration test suite

### v1.1 ✅
`kongtrol doctor` · Profile groups · `status --watch` · `kongtrol export`

### v1.2 🔲
Per-app routing (experimental) · Full E2E test suite with live VPN daemons

---

## License

MIT — see [LICENSE](LICENSE)
