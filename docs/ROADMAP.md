# Project Status & Roadmap

## Status

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
| Policy engine (IP longest-prefix · domain glob · app+flow matching) | ✅ |
| Kill switch (WFP · iptables · pf) — timeout-bounded subprocess calls | ✅ |
| Per-profile kill-switch overrides | ✅ |
| DNS guard (netsh · resolv.conf · networksetup) — timeout-bounded | ✅ |
| Transparent split-DNS (policy domain host injection) | ✅ |
| Auto-reconnect watchdog (exponential backoff, timeout-bounded reconnects) | ✅ |
| Health-check + priority failover | ✅ |
| Profile scheduler (weekday/time windows) | ✅ |
| Leak detection (periodic · configurable action) | ✅ |
| Audit log (append-only · HMAC-SHA256 signed, fsnotify-based live tail) | ✅ |
| Metrics collector + persistent profile history (event-driven change broadcast) | ✅ |
| Embedded web dashboard — full management UI (dark/light theme, sidebar nav, live WebSocket feed) | ✅ |
| Dashboard: VPN profile + group CRUD (config + keychain) | ✅ |
| Dashboard: live security toggles (kill switch, DNS guard, per-profile override) | ✅ |
| Dashboard: settings page (monitor/security tuning, scheduler rules CRUD) | ✅ |
| Dashboard: audit log viewer, per-tunnel traffic charts | ✅ |
| Dashboard bind/port: CLI-local override (`kongtrol config dashboard`) | ✅ |
| System tray app (logo icon · per-profile menu) | ✅ |
| Graceful daemon shutdown (`kongtrol down` → `POST /api/v1/shutdown`) | ✅ |
| `kongtrol init` wizard (detect · preserve existing · keychain) | ✅ |
| Docker + docker-compose (privileged · /dev/net/tun) | ✅ |
| goreleaser multi-platform pipeline | ✅ |
| `kongtrol doctor` (full stack diagnostics, timeout-bounded checks) | ✅ |
| Profile groups (`kongtrol up --group work`) | ✅ |
| `kongtrol status --watch` (live terminal view, remote daemon streaming) | ✅ |
| `kongtrol export` (sanitized config template for teammates) | ✅ |
| Per-app routing rules (experimental) | ✅ |
| Unit, integration, and race-detector test coverage | ✅ |
| Full E2E test suite with live VPN daemons | ✅ |

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

### v1.2 ✅
Per-app routing (experimental) · Full E2E test suite with live VPN daemons

### v1.3 ✅
Event-driven architecture (fsnotify log tailing, metrics change broadcast, live remote status streaming) · Graceful cross-platform daemon shutdown · Timeout-bounded system calls throughout the security layer

### v1.4 ✅
Dashboard rebuilt as a full management UI: sidebar navigation, light/dark theme, per-tunnel traffic charts · VPN profile + group CRUD · live Kill Switch/DNS Guard toggles + per-profile override · Settings page (scheduler rules, split DNS, audit log tuning) · Audit log viewer · CLI-local dashboard bind/port override (`kongtrol config dashboard`)
