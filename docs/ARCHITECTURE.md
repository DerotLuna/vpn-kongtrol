# Architecture

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

The `up` command owns the daemon lifecycle: it starts the metrics collector, watchdog, DNS manager, and embedded API server, and blocks in a live terminal view until interrupted. `status --watch` is a separate read-only viewer — it discovers a running daemon over the API/WebSocket instead of managing tunnels itself.

## Project layout

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
│   │   ├── engine.go              # ResolveIP / ResolveDomain / ResolveApp
│   │   └── rules.go               # Rule · MatchSpec · ParseRule
│   │
│   ├── security/
│   │   ├── killswitch*.go         # OS implementations
│   │   ├── dnsguard*.go           # OS implementations
│   │   ├── leaktest.go
│   │   └── audit.go
│   │
│   ├── monitor/
│   │   ├── collector.go           # Aggregate metrics snapshot + change broadcast
│   │   ├── watchdog.go            # Auto-reconnect with backoff
│   │   ├── dnsmanager.go          # Reference-counted DNS guard
│   │   ├── splitdns.go            # Transparent split-DNS hosts injection
│   │   └── scheduler.go           # Time-based profile scheduler
│   │
│   ├── config/
│   │   ├── schema.go              # Config structs + validation tags
│   │   ├── loader.go              # YAML load + env expansion + defaults
│   │   └── keychain.go            # OS keychain read/write
│   │
│   └── api/
│       ├── server.go
│       ├── handlers.go            # REST: tunnels · routes · policies · vpns · groups ·
│       │                          #       security toggles · settings · scheduler · audit
│       └── ws.go                  # WebSocket live metrics feed
│
├── web/
│   ├── assets.go                  # //go:embed dashboard
│   └── dashboard/                 # full management UI, vanilla JS/CSS, no bundler
│       ├── shell.js               # shared sidebar (collapsible, full-height) + topbar
│       ├── toast.js               # toast notifications
│       ├── select.js              # custom-styled <select> replacement, used everywhere
│       ├── charts.js              # dependency-free canvas charts (sparklines, time series)
│       ├── index.html / app.js    # Overview: tunnels, traffic charts, routes, policy
│       │                          #   resolver, groups quick-launch
│       ├── studio.html / .js      # Studio: tabbed policy CRUD + VPN profile CRUD +
│       │                          #   groups CRUD/connect/disconnect (shared tabs.js-less
│       │                          #   initTabs() helper in shell.js)
│       ├── security.html / .js    # live kill switch/DNS guard toggles, per-profile override
│       ├── settings.html / .js    # tabbed: general/security tuning + scheduler rules CRUD
│       ├── audit.html / .js       # audit log viewer (filter by profile/level)
│       ├── style.css              # shared styles (light/dark theme via [data-theme])
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
├── docs/                          # Reference documentation (this directory)
├── .goreleaser.yaml
├── Makefile
├── go.mod
└── go.sum
```

## Adding a new VPN adapter

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

1. Create `internal/vpn/<name>/adapter.go` — implement the interface.
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
4. Add `type: myvpn` as a valid value in `internal/config/schema.go`.
5. Add the adapter entry in `configs/example.yaml`.

No changes to routing, security, monitoring, or the policy engine are required.
