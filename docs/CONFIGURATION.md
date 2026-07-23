# Configuration Reference

Full reference for `~/.kongtrol/kongtrol.yaml`. For a guided walkthrough that builds this file interactively, run `kongtrol init` or see [SETUP.md](SETUP.md).

```yaml
# ~/.kongtrol/kongtrol.yaml
# Run 'kongtrol init' to generate this file interactively.
# Passwords are NEVER stored here — use 'kongtrol init'
# to store/update credentials in the OS keychain.

vpns:
  office:
    type: forticlient
    version: "6.4"
    host: vpn.empresa.com
    port: 443
    tunnel_name: "Office"
    # binary_path: "C:\Program Files\Fortinet\FortiClient\FortiClient.exe"
    # Optional: override auto-detected binary location. Leave unset to let
    # Kongtrol find it automatically (searches standard + non-standard paths).
    auth:
      method: certificate+credentials
      cert: ~/.kongtrol/certs/office.crt
      key:  ~/.kongtrol/certs/office.key
      username: your_username
      password_keychain: office.password   # stored in OS keychain
    priority: 10
    kill_switch: true      # optional per-profile override (fallback: security.kill_switch.enabled)

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

  - name: "Corporate apps (experimental)"
    match:
      apps: ["chrome", "teams*", "*\\Code.exe"]
    via: office

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
    bind: "127.0.0.1"  # loopback only; non-local addresses are rejected
    # Machine-local override (doesn't touch this file): `kongtrol config
    # dashboard set-port <port>` / `set-bind <address>`, stored in
    # ~/.kongtrol/preferences.json. Not editable from the dashboard itself.

  health_check:
    interval: "30s"
    timeout:  "10s"

  history:
    path: ~/.kongtrol/history.json
    flush_interval: "30s"

  split_dns:
    enabled: true
    interval: "60s"

  scheduler:
    enabled: false
    interval: "1m"
    rules:
      - name: "work-hours"
        profiles: ["office"]
        weekdays: ["mon","tue","wed","thu","fri"]
        start: "09:00"
        end: "18:00"

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

## ProtonVPN: GUI vs WireGuard

Two ways to configure a `protonvpn`-routed profile, depending on your team's workflow.

### A) Manual via Proton GUI (quick start)

Use this when users already connect from the Proton app and just need a manual workflow.

1. Install the ProtonVPN desktop app.
2. Create/connect once manually in the GUI to verify account and connectivity.
3. In Kongtrol, add a `protonvpn` profile (`kongtrol init`) and set server/protocol.
4. This path still depends on Proton client behavior outside Kongtrol.

### B) Automatic via WireGuard config (recommended for stable automation)

Use this when you want deterministic automation with Kongtrol.

1. Install WireGuard runtime/tools (`wg`, `wg-quick`, or the WireGuard app/service).
2. In your Proton account/dashboard, generate a WireGuard config file (`.conf`).
3. Save that `.conf` locally (e.g. `~/.kongtrol/configs/proton-us.conf`).
4. In `kongtrol init`, add a profile of type `wireguard` pointing to that file.

```yaml
vpns:
  proton-us:
    type: wireguard
    config: ~/.kongtrol/configs/proton-us.conf
    auth:
      method: certificate
```

### Combining ProtonVPN with other adapters

- If Proton is only for selected domains/apps, keep it in its own profile (e.g. `us-content`) and route only those policies through it.
- Keep corporate traffic (`forticlient`, `openvpn`, etc.) in separate profiles with explicit policy targets.
- Prefer policy-based separation over a catch-all default route when combining Proton with work VPNs.
- Use profile `priority` intentionally if two policies may overlap.
