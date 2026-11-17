<img src="../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# Security architecture

Version 2.1.0 · 2026-11-17

Technical companion to the [security policy](../SECURITY.md) (how to report a vulnerability,
supported versions, scope). This page describes how Kongtrol protects the machine it runs on and
exactly which network connections it makes.

## No telemetry

Kongtrol sends no telemetry, analytics, crash reports or update checks. Credentials go straight
from your prompt into the OS credential store (`internal/config`) and are never written to disk in
plaintext or logged. The audit log (`internal/security`) is local, HMAC-signed for
tamper-evidence, and never leaves the machine.

## Network connections Kongtrol makes

Besides talking to your VPN clients, your OS network stack and the local dashboard on
`127.0.0.1`, Kongtrol itself opens these connections. None of them carries configuration,
credentials or usage data.

| Connection | When | Why | Code |
|---|---|---|---|
| HTTPS `GET` to the public IP service — `https://api.ipify.org` by default, or the `security.public_ip_check.endpoints` you set (https only) | **Off by default (opt-in).** Only when `security.public_ip_check.enabled: true` (or the legacy `leak_detection.enabled: true`): once after a tunnel connects or disconnects (at most one lookup per 30 s, coalesced) and when you ask (`kongtrol check`, "Revisar" in the dashboard, the tray's public IP line). Never periodic. The kill switch and DNS guard do not use it. | Show your public IP and compare it with `integrity_check.expected_ips` (leak check). That service sees your public IP, as with any website. 5 s timeout; the answer must be a bare IP of at most 64 bytes. | `internal/security/leaktest.go`, `cmd/kongtrol/publicip.go` |
| TCP connect to the profile's `host:port` (default 443), closed immediately | Before each connect (preflight) and on tunnel health checks | Warn early when the VPN server is unreachable; measure latency. | `internal/app/profile_service.go`, `cmd/kongtrol/daemon_runtime.go` |
| DNS lookups of policy domains, through the system resolver and the tunnels' DNS servers (UDP/TCP 53) | When domain rules are resolved to routes | Turn `*.example.com` rules into IP routes. | `internal/monitor/policyresolver.go` |
| DNS lookup of VPN server hostnames | When the per-profile kill switch is applied | Allow-list the VPN server so the tunnel can come up behind the block. | `internal/app/killswitch_service.go` |
| TCP to the tunnel's DNS server on port 53 | Health checks when the VPN server itself cannot be probed | Measure a round trip inside the tunnel. | `cmd/kongtrol/daemon_runtime.go` |

Public IP services: the default is ipify (https://www.ipify.org, plain-text answer). Other well-known
plain-text service you can list in `endpoints` is `https://icanhazip.com` (run by Cloudflare per its
creator: https://major.io/p/a-new-future-for-icanhazip/). Any HTTPS endpoint that answers with just
an IP works. Each one sees your public IP; pick the one you trust. See
`docs/research/2026-11-17-public-ip-services.md`.

Loopback only: the OpenVPN management interface (`internal/vpn/openvpn/management.go`) and the
privileged helper's socket or named pipe (below). Your VPN clients (NordVPN, FortiClient, OpenVPN,
WireGuard) make their own connections; those depend on each provider. The dashboard serves its
fonts and logos from the binary (`/brand/`), so opening it contacts no third party.

## Privileged helper

Routes, DNS, the firewall (kill switch), OpenVPN processes, WireGuard tunnels and the hosts file
need administrator rights. Instead of running the whole daemon elevated, an optional helper
(`internal/privhelper`, installed once with `kongtrol service install`) runs as root / SYSTEM and
exposes a fixed list of operations (`internal/privops/protocol.go`), nothing else — no shell, no
arbitrary command.

- **Transport.** Unix socket `/run/kongtrol-helper.sock` (Linux) or `/var/run/kongtrol-helper.sock`
  (macOS), created root-owned with a restrictive umask; named pipe `\.\pipe\kongtrol-helper` on
  Windows. The Windows client refuses a pipe not owned by SYSTEM or Administrators, so a standard
  user cannot squat the name.
- **Authorization.** Only the user who installed the helper, or an elevated administrator / root,
  may call it (`authorize` in `internal/privhelper/server.go`). The peer is identified by the OS
  (Unix peer credentials; the pipe client's process token on Windows), not by anything the client claims.
- **Bounded work.** One request per connection; each operation runs under its own timeout
  (`privops.OpTimeout`, 15–45 s).
- **Audit.** Every request — caller, operation, arguments, result, duration — is written to the
  helper's JSON-lines audit log (`internal/privhelper/audit.go`, rotated at 5 MB).

### Leases: no orphaned state

Every client picks a random lease id and renews it every 5 s. State the helper creates on a
client's behalf (an OpenVPN process, the kill switch, forced DNS, the hosts block) is owned by that
lease. A lease not renewed for 20 s expires and the helper undoes everything it owns, so a daemon
that crashes or is killed never leaves an unmanaged tunnel, a firewall blocking the network or
resolvers pointing at a dead tunnel (`internal/privops/lease.go`).

Expiry fails **open**: the kill switch is removed rather than left blocking, because a dead daemon
can no longer turn it off and the Windows firewall rules and static DNS survive reboots. The daemon
re-applies the kill switch when it comes back. Not leased, on purpose: WireGuard tunnels (an OS
service a restarted daemon adopts) and routes (they die with the tunnel interface).

## Unsigned binaries and antivirus false positives

Kongtrol does not currently ship with a CA-issued code-signing certificate — that costs money
(EV certs run several hundred dollars a year) and this is an unfunded open-source project. Windows
SmartScreen and Defender lean heavily on binary reputation (hash + publisher trust), and a
freshly-built, unsigned Go binary that manipulates routing tables, DNS, and firewall rules is
exactly the profile their ML heuristics flag — commonly as a generic detection like
`Trojan:Win32/Bearfoos.A!ml`. That `!ml` suffix means it was flagged by a cloud ML model on
low-prevalence/no-reputation files, not a signature match against known malware. It's a known,
common false positive for this class of tool, not evidence of anything malicious.

What to do about it:

- **Verify the download.** Every GitHub release includes a `checksums.txt` (SHA256). It's produced
  automatically — pushing a `v*` tag triggers `.github/workflows/release.yml`, which runs
  `goreleaser release --clean` (config in `.goreleaser.yaml`); goreleaser cross-compiles every
  platform binary, hashes them, and uploads both the binaries and `checksums.txt` as release assets
  in one step. Nothing needs to be uploaded by hand. Compare the binary you downloaded:

  ```powershell
  # Windows
  Get-FileHash kongtrol-windows-amd64.exe -Algorithm SHA256
  ```

  ```bash
  # Linux/macOS
  sha256sum -c checksums.txt
  ```

- **Report the false positive** to Microsoft so the reputation model stops flagging it for
  everyone: https://www.microsoft.com/en-us/wdsi/filesubmission

- **Build it yourself** after reading the source — the whole point of open source. See the
  [README Quick start](../README.md#quick-start) (`make build` / `make build-all-cli`).

- **Sign your own local build**, Windows-only, LOCAL-ONLY: `pwsh scripts/gen-devcert.ps1` generates
  a self-signed cert and trusts it under `Cert:\CurrentUser` on your machine; `make build sign`
  then signs the `.exe` files in `build/dist/`. This stops SmartScreen/Defender from flagging
  binaries **you personally compile**, and only on **the machine you ran the script on** — a
  self-signed cert has no chain to a public CA, so it establishes zero trust anywhere else. It has
  no Linux/macOS equivalent (Authenticode only applies to Windows PE files) and `make sign` no-ops
  on those platforms. Never sign release/landing-page binaries with this cert — it would look
  "signed" without actually meaning anything to whoever downloads it. It exists purely so `make
  sign` is already wired up for a real CA-issued cert if the project ever gets one; swapping in a
  purchased `.pfx` via `KONGTROL_SIGN_PFX`/`KONGTROL_SIGN_PFX_PASSWORD` is the only change needed
  at that point — release automation still would not use it automatically, that'd be a deliberate
  addition to `release.yml`.

## Enforcement per OS

| Layer | Windows | Linux | macOS |
|---|---|---|---|
| **Kill Switch** | `netsh advfirewall` block + tunnel allow | `iptables OUTPUT -j DROP` + ACCEPT | `pf` anchor at `/etc/pf.anchors/kongtrol` |
| **DNS Guard** | `netsh interface ip set dns` per interface | `/etc/resolv.conf` rewrite (backup kept) | `networksetup -setdnsservers` per service |
| **Credentials** | Windows Credential Manager | libsecret / D-Bus Secret Service | macOS Keychain |
| **Audit log** | HMAC-SHA256 per entry; signing key in OS keychain | ← same | ← same |
| **Dashboard** | Localhost-only bind (`127.0.0.1`) + per-user API capability | ← same | ← same |

Every subprocess the security layer shells out to (`pfctl`, `iptables`, `networksetup`, `netsh`, elevated PowerShell on Windows) runs under a bounded timeout, so a hung system call can't block the daemon's shutdown or leave the kill switch / DNS guard in an indeterminate state.

## Toggling from the dashboard

The dashboard's **Security** page can flip Kill Switch and DNS Guard on/off live (`POST
/api/v1/security/killswitch`, `POST /api/v1/security/dnsguard`) and set a per-profile kill switch
override (`PUT /api/v1/vpns/{name}/killswitch`, `"override": "inherit"|"on"|"off"`). Each toggle
persists to `kongtrol.yaml` *and* immediately re-applies enforcement — for the kill switch that's a
re-run of `KillSwitchService.Apply()` (`internal/app/killswitch_service.go`); for DNS guard it's
`applyDNSGuardState()` (`cmd/kongtrol/main.go`), which mirrors the same enable gate `ProfileService.Connect`
uses. Nothing here is exposed on the network by default — the dashboard only binds to
loopback addresses. Non-loopback overrides are rejected. Native clients authenticate with the
mode-`0600` capability in `~/.kongtrol/api-token`; the dashboard receives the same capability in
an HttpOnly, SameSite=Strict cookie and state-changing cross-origin requests are rejected.
The cookie is deliberately not `Secure`: the dashboard is plain HTTP on loopback only, and Safari
has rejected Secure cookies over HTTP even on loopback (https://flaviocopes.com/cookie-not-being-set-in-safari/,
https://bugs.webkit.org/show_bug.cgi?id=281149), which would lock Safari users out. The related
CodeQL alert is dismissed with this justification.

## DNS guard recovery

If Kongtrol crashes without cleanly restoring DNS, run:

```bash
# Windows
netsh interface ip set dns "Ethernet" dhcp

# Linux
sudo cp /etc/resolv.conf.kongtrol.bak /etc/resolv.conf

# macOS
networksetup -setdnsservers Wi-Fi empty
```

## Kill switch recovery

```bash
kongtrol down --all    # disables the kill switch automatically
```

If Kongtrol closed abruptly and traffic is still blocked:

```bash
# Windows (as Administrator)
netsh advfirewall reset

# Linux
sudo iptables -F OUTPUT

# macOS
sudo pfctl -d
```

## Auto-reconnect watchdog

The watchdog polls every 5 seconds per profile. On an unexpected disconnect it waits `2s → 4s → 8s → … → 5min` (exponential backoff) before each retry, and each reconnect attempt is itself bounded by a timeout so a hung adapter driver can't stall the watcher indefinitely. Intentional disconnects via `kongtrol down` suppress reconnect automatically — the watchdog only fires on drops you didn't request.

## Graceful daemon shutdown

`kongtrol down` asks a running `kongtrol up` daemon to shut down through its own API (`POST /api/v1/shutdown`) rather than killing the process directly. This runs the daemon's normal cleanup — DNS restore, kill switch teardown, history flush — before it exits, on every OS (a plain process kill has no cross-platform graceful-termination equivalent, particularly on Windows). A hard kill is only used as a fallback if the daemon doesn't respond.
