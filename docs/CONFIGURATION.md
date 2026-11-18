# Configuration Reference

Full reference for `~/.kongtrol/kongtrol.yaml`. For a guided walkthrough that builds this file interactively, run `kongtrol init` or see [SETUP.md](SETUP.md).
Every key with its type and default, generated from the schema: [CONFIG.en.md](CONFIG.en.md) (Español: [CONFIG.md](CONFIG.md)).

Names of VPN profiles, policies, groups and scheduler rules: 1–64 characters; letters, numbers,
spaces and `- _ .` are fine; `<`, `>`, `"`, `'`, `` ` ``, `&`, `\` and control characters are refused (they are shown in
the dashboards and the tray; [ADR-0040](adr/0040-local-hardening-review-2026-11.md)).

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

  # Public IP lookup — OFF by default (opt-in). When on, Kongtrol asks an
  # external service for your public IP (that service sees it), only when a
  # tunnel connects or disconnects (at most once per 30 s) and when you ask
  # (kongtrol ip, "Revisar" in the dashboard, the tray). The kill switch
  # and DNS guard do not need it. Also enables the leak check against
  # integrity_check.expected_ips. Toggle: kongtrol config public-ip on|off.
  public_ip_check:
    enabled: false
    # endpoints:                  # https only, tried in order; default below
    #   - https://api.ipify.org
    #   - https://icanhazip.com

  # Legacy: leak_detection.enabled: true still opts in to the lookup above;
  # its old interval and action keys are ignored and dropped on save: a
  # leak only notifies, it never drives the kill switch (ADR-0041).

  audit_log:
    path: ~/.kongtrol/audit.log   # a file, never a folder; daily files audit-YYYY-MM-DD.log
    # max_size_mb: one file per day up to this size; past it the day goes on in
    # audit-YYYY-MM-DD.1.log, .2… (default 100). retention_days: days kept,
    # today included (0 = today and yesterday, max 365). Both apply live.
    max_size_mb: 10
    retention_days: 30
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

<!-- feature:providers feature:wireguard -->
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
- Use profile `priority` intentionally if two policies may overlap: the **higher** number is evaluated first (domain and app rules: first match wins; IPs: the longest prefix always wins and `priority` only breaks ties between prefixes of the same length). Without it, the policy's position counts (earlier first). `priority` never connects or switches profiles.

<!-- feature:watchdog -->
### Failover (`failover:`)

When a profile drops and cannot reconnect, Kongtrol keeps retrying **that profile only** (backoff 2 s → 5 min) and notifies you once. It never brings up another VPN on its own: profiles reach different networks, even with similar names or equal `priority` ([ADR-0018](adr/0018-failover-only-to-declared-profiles.md)).

To allow a backup, declare it on the profile that may fail, in order of preference:

```yaml
vpns:
  dev-server:
    type: openvpn
    config: ~/.kongtrol/configs/server.ovpn
    failover: [dev-server-b]       # tried once per outage, in order
  dev-server-b:
    type: openvpn
    config: ~/.kongtrol/configs/server-b.ovpn
```

Each entry must be an existing profile, not the profile itself, listed once, and the lists must not form a loop (`a → b → a`). When a backup takes over, Kongtrol stops retrying the original until you connect it again. Health-check failures only notify; they never start a failover.

<!-- feature:wireguard feature:watchdog -->
### WireGuard: does the server answer?

WireGuard has no "connected" state: its interface stays up even when the server is gone. Kongtrol therefore reads each WireGuard tunnel's **latest handshake** every 10 s (through the helper, no admin terminal) and shows one of these under the tunnel in `kongtrol status`, the live view, the tray and the dashboard ([ADR-0030](adr/0030-wireguard-liveness-by-handshake.md)):

| What you see | Meaning |
|---|---|
| nothing extra | The server answered in the last ~3 minutes. |
| "No recent traffic · normal if you are not using it" | Nothing was sent, so nothing needed an answer. WireGuard only handshakes when there is traffic or `PersistentKeepalive`. |
| status `error` + "No answer from the server · …" | Kongtrol kept sending (keepalive or traffic) and the server did not answer: no handshake for 200 s with `PersistentKeepalive`, traffic out with nothing back for 15 s, or no handshake 30 s after connecting. It is treated like a drop: reconnect, your declared `failover`, kill switch and `on_tunnel_down` apply. You get one notice per outage and another when it answers again. |
| "Can't check whether the server answers · …" | The counters could not be read (e.g. the helper is missing or out of date: run `kongtrol service install`). The tunnel interface is up, but Kongtrol can't confirm it works. |

Add `PersistentKeepalive = 25` to the `[Peer]` section if you want a dead server detected even while the tunnel is idle (it sends a small packet every 25 s).
