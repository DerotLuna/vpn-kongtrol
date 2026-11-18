<img src="../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# Security architecture

Version 2.4.1 · 2026-11-18

Technical companion to the [security policy](../SECURITY.md) (how to report a vulnerability,
supported versions, scope). This page describes how Kongtrol protects the machine it runs on and
exactly which network connections it makes.

## No telemetry

Kongtrol sends no telemetry, analytics or crash reports, and no update check unless you turn the
optional new version notice on (off by default, see below). Credentials go straight
from your prompt into the OS credential store (`internal/config`) and are never written to disk in
plaintext or logged. The audit log (`internal/security`) is local, HMAC-signed for
tamper-evidence, and never leaves the machine.

## Network connections Kongtrol makes

Besides talking to your VPN clients, your OS network stack and the local dashboard on
`127.0.0.1`, Kongtrol itself opens these connections. None of them carries configuration,
credentials or usage data.

| Connection | When | Why | Code |
|---|---|---|---|
| HTTPS `GET` to the public IP service — `https://api.ipify.org` by default, or the `security.public_ip_check.endpoints` you set (https only) | **Off by default (opt-in).** Only when `security.public_ip_check.enabled: true` (or the legacy `leak_detection.enabled: true`): once after a tunnel connects or disconnects (at most one lookup per 30 s, coalesced) and when you ask (`kongtrol ip`, "Revisar" in the dashboard, the tray's public IP line). Never periodic. The kill switch and DNS guard do not use it. | Show your public IP and compare it with `integrity_check.expected_ips` (leak check). That service sees your public IP, as with any website. 5 s timeout; the answer must be a bare IP of at most 64 bytes. | `internal/security/leaktest.go`, `cmd/kongtrol/publicip.go` |
| HTTPS `GET https://api.github.com/repos/DerotLuna/vpn-kongtrol/releases/latest` with only `User-Agent: kongtrol/<version>` and `Accept: application/vnd.github+json` (no token, cookie or user data) | **Off by default (opt-in).** Only when `updates.check_enabled: true` (`kongtrol config update-check on`, the dashboard's Ajustes, or "Sí" in `kongtrol init`): at most once every 24 h (the last attempt is remembered in `~/.kongtrol/update-check.json`, failures included), plus once right after you turn it on if 24 h have passed. | Tell you a newer stable version exists (pre-releases ignored), once per version, with how to update. It never downloads or installs anything. GitHub sees your public IP, the time and that User-Agent. 5 s timeout; at most 1 MiB of the answer is read. [ADR-0034](adr/0034-uninstall-and-update-notice.md) | `internal/update/update.go`, `cmd/kongtrol/daemon_update.go` |
| TCP connect to the profile's `host:port` (default 443), closed immediately | Before each connect (preflight) and on tunnel health checks | Warn early when the VPN server is unreachable; measure latency. | `internal/app/profile_service.go`, `cmd/kongtrol/daemon_runtime.go` |
| DNS lookups of policy domains, through the system resolver and the tunnels' DNS servers (UDP/TCP 53) | When domain rules are resolved to routes | Turn `*.example.com` rules into IP routes. | `internal/monitor/policyresolver.go` |
| DNS lookup of VPN server hostnames | When the per-profile kill switch is applied | Allow-list the VPN server so the tunnel can come up behind the block. | `internal/app/killswitch_service.go` |
| TCP to the tunnel's DNS server on port 53 | Health checks when the VPN server itself cannot be probed | Measure a round trip inside the tunnel. | `cmd/kongtrol/daemon_runtime.go` |

<!-- feature:update_check -->
Updates come from the channel you installed with (winget, Scoop, Homebrew, AUR, apt/dnf, npm, the
Windows installer or `install.sh`); see [ADR-0025](adr/0025-desktop-release-channels-first-run.md).
Kongtrol asks GitHub whether a new version exists only when you turned the optional notice on (row
above); it never updates itself.

<!-- feature:uninstall -->
Uninstalling keeps your data unless you ask: `kongtrol uninstall --purge`, "Borrar todos mis datos"
in the dashboard, or "Sí" in the Windows uninstaller delete `~/.kongtrol` and exactly Kongtrol's
keychain entries (`vpn-kongtrol` / `<profile>.username`, `<profile>.password` for every profile
named in `kongtrol.yaml` and its backups, and `_system.audit_hmac_key`). Files owned by a package
manager are never deleted. See [ADR-0034](adr/0034-uninstall-and-update-notice.md).

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
  helper's JSON-lines audit log (`internal/privhelper/audit.go`, rotated at 5 MB). The one
  exception is a *successful* read-only request (`wg.stats`, polled every 10 s per WireGuard
  tunnel); its failures are audited.
- **Read-only WireGuard counters.** `wg.stats` runs `wg show <iface> dump` only for a tunnel the
  helper manages (or the caller's own profile tunnel, checked like `wg.remove`, nothing adopted)
  and returns only peer public keys, latest handshake, rx/tx bytes and keepalive. The interface
  private key and preshared keys in the dump never leave the helper, and a failed read never
  echoes the command output ([ADR-0030](adr/0030-wireguard-liveness-by-handshake.md)).

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

- **Free signing for open source** ([ADR-0025](adr/0025-desktop-release-channels-first-run.md)):
  `release.yml` already submits `kongtrol.exe`, `kongtrol-tray.exe` and the Windows installer to
  [SignPath Foundation](https://signpath.org) (free Authenticode certificate for OSI-licensed
  projects). The steps are skipped until the project is approved and the `SIGNPATH_API_TOKEN`
  secret exists. macOS has no free path: Developer ID signing and notarization need the paid Apple
  Developer Program (the fee waiver excludes individuals), so the app is ad-hoc signed and the
  install script, Homebrew formula and npm package are used because they do not set the
  quarantine flag.

<!-- feature:distribution -->
## Verifying releases

Every release (`.github/workflows/release.yml`) publishes, for every file:

- `checksums.txt` — SHA-256 of every asset, including the macOS app, the Windows installer and
  the `install.sh` / `install.ps1` scripts attached to the release (ADR-0035).
- `checksums.txt.sigstore.json` — keyless [Sigstore](https://www.sigstore.dev) signature of
  `checksums.txt`, made with the workflow's GitHub OIDC identity (no key to leak):

  ```bash
  cosign verify-blob checksums.txt --bundle checksums.txt.sigstore.json \
    --certificate-identity-regexp '^https://github.com/DerotLuna/vpn-kongtrol/\.github/workflows/release\.yml@refs/tags/v' \
    --certificate-oidc-issuer https://token.actions.githubusercontent.com
  sha256sum --ignore-missing -c checksums.txt
  ```

- GitHub [build provenance attestations](https://docs.github.com/en/actions/security-for-github-actions/using-artifact-attestations)
  (SLSA provenance) for every archive, package, installer and binary:
  `gh attestation verify <file> --repo DerotLuna/vpn-kongtrol`.
- An SBOM (SPDX JSON, `*.sbom.json`) for every archive.
- npm packages are published with `--provenance`, shown on npmjs.com.

Package managers (winget, Scoop, Homebrew, AUR, apt/dnf) check the SHA-256 written in their
manifest. None of these steps sends anything from the user's machine.

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
