<img src="../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Security architecture

Version 2.13.0 · 2026-11-20

Technical companion to the [security policy](../SECURITY.md) (how to report a vulnerability,
supported versions, scope). This page describes how Kongtrol protects the machine it runs on and
exactly which network connections it makes.

## No telemetry

Kongtrol sends no telemetry, analytics or crash reports, and no update check unless you turn the
optional new version notice on (off by default, see below). Credentials go straight
from your prompt into the OS credential store (`internal/config`) and are never written to disk in
plaintext or logged. The audit log (`internal/security`) is local, HMAC-signed for
tamper-evidence, and never leaves the machine. Signatures are checked when you read it (`kongtrol logs`,
the dashboard: `verified` per entry) with the key in the OS keychain; the key is never sent to any
client. It keeps `retention_days` days (default 30) in at most `max_total_mb` (default 500 MB), one
file per day split at `max_size_mb`; see "Audit log" below for what it records and how old files go.

<!-- feature:audit_log -->
## Audit log

**Files and limits** ([ADR-0064](adr/0064-audit-log-retention-and-coverage.md)). One file per local
day, `<path>-YYYY-MM-DD.log`; past `max_size_mb` (default 100) the day goes on in `.1`, `.2`…
Old files are deleted automatically when the daemon starts, when a file fills up and every local
midnight (a timer, so it also happens on days nothing is logged):

- files of days older than `retention_days` (1–365, today included; unset or `0` = 30, also for
  configs written before this change, where `0` used to mean "today and yesterday");
- then the oldest files until all of them, with room for the open file to reach `max_size_mb`, fit
  in `max_total_mb` (1–10240; unset or `0` = 500, never less than one file). So the log can never
  use more than `max_total_mb` of disk.

The open file is never deleted. Set them in the dashboard (Actividad → Ajustes del registro),
with `kongtrol config audit-log --days N --max-size-mb N --max-total-mb N`, or in `kongtrol.yaml`.

**What is recorded.** Each entry has time, level, action, profile (when there is one) and a short
message: who / what / when / result. Never request bodies, passwords, tokens, keys, nonces or file
contents — API requests are logged at the equivalent of Kubernetes' `Metadata` audit level
(metadata only, no body). The log belongs to the daemon: only what goes through it is recorded.

| Action | Recorded | Entry (`action`) | Level |
|---|---|---|---|
| Connect / disconnect / connect failed / preflight failed | Yes | `vpn.connect`, `vpn.disconnect`, `vpn.connect_failed`, `vpn.preflight_failed` | INFO / ERROR |
| Attach phases (provider apps), manual disconnect timeout, re-login needed | Yes | `vpn.attach.*`, `vpn.disconnect_timeout`, `vpn.reauth_required` | INFO / WARN |
| Watchdog: reconnect attempt / reconnected / failed, no response, responding, manual reconnect, health degraded | Yes | `vpn.reconnect_*`, `vpn.reconnected`, `vpn.no_response`, `vpn.responding`, `vpn.manual_reconnect_required`, `vpn.health_degraded` | INFO / WARN / ERROR |
| Watchdog failover to a declared backup (started / activated / failed) | Yes | `vpn.failover_started`, `vpn.failover_activated`, `vpn.failover_failed` | WARN / ERROR |
| Policy `on_tunnel_down` taking effect | Yes | `policy.tunnel_down.*` | per rule |
| Leak detected (public IP not the expected one) | Yes | `security.leak` | SECURITY |
| Kill switch state change (off / armed / active and what it blocks) and failure to apply the block | **Added** | `security.killswitch.state`, `security.killswitch.error` | SECURITY when blocking, else INFO / ERROR |
| Kill switch, DNS guard, public IP lookup, per-profile kill switch toggled | **Added** | `security.killswitch.set`, `security.dnsguard.set`, `security.publicip.set`, `security.profile_killswitch.set` | SECURITY |
| Public IP checked on demand | **Added** | `security.publicip.check` | INFO |
| Profile create / update (incl. rename) / delete / import of a `.conf`/`.ovpn` (incl. the elevated read of a root-only file) | **Added** (name and result; never the file or the password) | `config.profile.create`, `.update`, `.delete`, `.import` | SECURITY |
| Policy, group and scheduler rule create / update / delete; policy reload | **Added** | `config.policy.*`, `config.group.*`, `config.schedule.*` | INFO |
| Settings saved (incl. audit log, kill switch mode, public IP endpoints) | **Added** | `config.settings.update` | SECURITY |
| Preferences, update notice on/off | **Added** | `config.preferences.update`, `config.update_check.set` | INFO |
| Connect / disconnect / reload requested (tunnel or group), connect cancelled | **Added** (who asked; the result is the `vpn.*` entry) | `api.tunnel.*`, `api.group.*` | INFO |
| Helper install / update from the tray or dashboard | **Added** | `helper.install` | SECURITY |
| Restart / shutdown requested; daemon started / stopped | **Added** (`daemon.restart` already existed) | `daemon.restart_request`, `daemon.restart`, `daemon.shutdown`, `daemon.start`, `daemon.stop` | INFO |
| Delete all my data | **Added**, written when the request arrives; the purge then deletes the log itself (by design), and the daemon log keeps the result | `data.purge` | SECURITY |
| Dashboard sign-in: nonce issued (app), session created or refused (dashboard), sign-out | **Added** | `auth.nonce`, `auth.session_create`, `auth.session_end` | INFO / SECURITY; WARN if refused |
| API request refused: no valid token or session, foreign origin, non-local host | **Added**, at most one entry per 10 s; the next says how many were skipped | `auth.denied` | SECURITY |
| Any other change through the API (a route added later without its own action) | **Added** as a fallback | `api.request` | INFO |
| Reads (status, lists, logs, doctor, export) | No: nothing changes | — | — |
| Automatic public IP lookup after connect/disconnect, update check request | No: no state changes; a found leak is `security.leak`, a new version is a notice | — | — |
| Token rotation | Not applicable: `~/.kongtrol/api-token` is created once (0600) and never rotated; deleting it makes a new one at the next start | — | — |
| Edits made with the CLI while no daemon runs (`kongtrol config …`, `profile add`), `kongtrol service install` from a terminal, hand edits of `kongtrol.yaml` | No (no daemon to write it). The privileged helper keeps its own audit of every privileged operation (see "Privileged helper") | — | — |

Code: `internal/api/audit_requests.go` (route → action, a test fails if a mutating route has none),
`cmd/kongtrol/audit_events.go`, `internal/app/profile_service.go`, `cmd/kongtrol/daemon_watchdog.go`.

## Network connections Kongtrol makes

Besides talking to your VPN clients, your OS network stack and the local dashboard on
`127.0.0.1`, Kongtrol itself opens these connections. None of them carries configuration,
credentials or usage data.

| Connection | When | Why | Code |
|---|---|---|---|
| HTTPS `GET` to the public IP service — `https://api.ipify.org` by default, or the `security.public_ip_check.endpoints` you set (https only) | **Off by default (opt-in).** Only when `security.public_ip_check.enabled: true` (or the legacy `leak_detection.enabled: true`): once after a tunnel connects or disconnects (at most one lookup per 30 s, coalesced) and when you ask (`kongtrol ip`, "Revisar" in the dashboard, the tray's public IP line). Never periodic. The kill switch and DNS guard do not use it. | Show your public IP and compare it with `integrity_check.expected_ips` (leak check). That service sees your public IP, as with any website. 5 s timeout; the answer must be a bare IP of at most 64 bytes; https only, no userinfo (config validation and again at request time); redirects only over https to the same host (max 3). | `internal/security/leaktest.go`, `cmd/kongtrol/publicip.go` |
| HTTPS `GET https://api.github.com/repos/DerotLuna/vpn-kongtrol/releases/latest` with only `User-Agent: kongtrol/<version>` and `Accept: application/vnd.github+json` (no token, cookie or user data) | **Off by default (opt-in).** Only when `updates.check_enabled: true` (`kongtrol config update-check on`, the dashboard's Ajustes, or "Sí" in `kongtrol init`): at most once every 24 h (the last attempt is remembered in `~/.kongtrol/update-check.json`, failures included), plus once right after you turn it on if 24 h have passed. | Tell you a newer stable version exists (pre-releases ignored), once per version, with how to update. It never downloads or installs anything. GitHub sees your public IP, the time and that User-Agent. 5 s timeout; at most 1 MiB of the answer is read; TLS 1.2+ with default verification; redirects only over https to `api.github.com` (max 3). Only `tag_name` is used and it must be a strict `vX.Y.Z`; the release link is built as `https://github.com/DerotLuna/vpn-kongtrol/releases/tag/<tag>`, never taken from the answer (`html_url` is ignored), and a remembered link is re-validated on load. [ADR-0034](adr/0034-uninstall-and-update-notice.md) | `internal/update/update.go`, `cmd/kongtrol/daemon_update.go` |
| TCP connect to the profile's `host:port` (default 443), closed immediately | Before each connect (preflight) and on tunnel health checks | Warn early when the VPN server is unreachable; measure latency. | `internal/app/profile_service.go`, `cmd/kongtrol/daemon_runtime.go` |
| DNS lookups of policy domains, through the system resolver and the tunnels' DNS servers (UDP/TCP 53) | When domain rules are resolved to routes | Turn `*.example.com` rules into IP routes. For a wildcard, the base name and `www.`, `api.`, `cdn.`, `app.`, `docs.`, `console.` under it are looked up. | `internal/monitor/policyresolver.go`, `internal/app/profile_service.go` |
| DNS lookup of VPN server hostnames | When the per-profile kill switch is applied | Allow-list the VPN server so the tunnel can come up behind the block. | `internal/app/killswitch_service.go` |
| TCP to the tunnel's DNS server on port 53 | Health checks when the VPN server itself cannot be probed | Measure a round trip inside the tunnel. | `cmd/kongtrol/daemon_runtime.go` |
| DNS query for `connectivitycheck.gstatic.com` sent only to the tunnel's own DNS servers (UDP 53) | Tunnel health checks of a connected profile that pushed DNS servers | Check that DNS works inside the tunnel. Only the VPN's DNS server sees the query; nothing is sent to Google. | `cmd/kongtrol/daemon_runtime.go` |

Nothing else leaves the machine. Local-only traffic: the daemon API on `127.0.0.1` (CLI, tray and
the browser panel, see "Local panel sign-in"), the OpenVPN management port on loopback, and the
privileged helper's unix socket / named pipe. Opening the panel launches the default browser on
a `http://127.0.0.1:<port>/…` address only. Inventory checked on 2026-11-18 by searching the code
for HTTP clients, dialers, resolvers and external tools (`curl`, `wget`, `Invoke-WebRequest`):
none other found.

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
manager are never deleted. Outside `~/.kongtrol` only single regular files are removed (a
`--config` file and the daily `audit-YYYY-MM-DD.log` files of a custom `security.audit_log.path`),
never a folder, link or anything recursively; `audit_log.path` must name a file, not a folder
(config validation). See [ADR-0034](adr/0034-uninstall-and-update-notice.md) and
[ADR-0040](adr/0040-local-hardening-review-2026-11.md).

Public IP services: the default is ipify (https://www.ipify.org, plain-text answer). Other well-known
plain-text service you can list in `endpoints` is `https://icanhazip.com` (run by Cloudflare per its
creator: https://major.io/p/a-new-future-for-icanhazip/). Any HTTPS endpoint that answers with just
an IP works. Each one sees your public IP; pick the one you trust. See
`docs/research/2026-11-17-public-ip-services.md`.

<!-- feature:recent_connections -->
## Data Kongtrol keeps about your traffic (local, not outbound)

The recent destinations list ([ADR-0054](adr/0054-recent-destinations-memory-only.md),
`internal/monitor/recent.go`) is **not** a network connection: it reads the same OS table
`netstat`/`ss` read (Windows `GetExtendedTcpTable`, Linux `/proc/net/tcp`; macOS not supported),
without admin, every 15 s. Per destination it keeps host (only a name a policy domain already
resolved to; no reverse DNS), remote IP, port, program name when the OS gives it to a standard
user, the tunnel or "direct" and the local interface, first/last seen and a count.

| Property | Value |
|---|---|
| Where | Only in the daemon's memory. Never written to disk, logs or the audit log; never sent anywhere. |
| How much | At most 500 destinations; each forgotten 1 h after it was last seen. |
| Who can read it | The authenticated loopback API (`GET /api/v1/connections/recent`: token or browser session, like every route): the dashboard and `kongtrol traffic recent`. |
| Off | `monitor.recent_connections.enabled: false` (default on), `kongtrol config recent-connections off` or the dashboard settings: the list is emptied at once and nothing is read. It is also emptied when Kongtrol stops. |

On by default because nothing leaves the machine and the same user can already read this table
with `netstat -ano` / `ss -tnp`; the opt-in rule above is for data that leaves the machine.

Loopback only: the OpenVPN management interface (`internal/vpn/openvpn/management.go`) and the
privileged helper's socket or named pipe (below). Your VPN clients (NordVPN, FortiClient, OpenVPN,
WireGuard) make their own connections; those depend on each provider. The dashboard bundles its
fonts and logos into the binary (`web/ui`), so opening it contacts no third party.

## Local panel sign-in

The daemon's local API (`127.0.0.1:9741` by default; non-loopback binds are refused and
`localhost` is pinned to `127.0.0.1`) has two kinds of callers
([ADR-0039](adr/0039-browser-sessions-one-time-nonce.md)):

- **Native clients** (CLI, tray) send the control token from `~/.kongtrol/api-token` (`0600` in
  a `0700` directory) in `X-Kongtrol-Token`. It is compared in constant time and is never
  accepted from a cookie or sent to a browser. The daemon writes a **new token on every start**,
  before it listens (atomic temp file + rename); clients read the file on every request, so a
  token caught earlier stops working once Kongtrol starts again
  ([ADR-0062](adr/0062-origin-bound-sessions-token-rotation-listener-owner.md)).
- **Listener-owner check.** Before the first byte of a request that carries the token, the CLI
  and the tray check who owns the other end of the loopback connection: Windows reads the owning
  process from `GetExtendedTcpTable` (IPv4 and IPv6 tables) and compares its user SID with
  yours (a process whose owner you may not read passes only if the process itself opens, it is the
  PID in your `~/.kongtrol/run/kongtrol.pid` and it started before that file was written: your own
  elevated daemon, not a PID reused after a crash); Linux compares
  the uid in `/proc/net/tcp`. Another user's program gets nothing and you see "Otro usuario de
  este equipo tiene un programa en la dirección de Kongtrol … Qué hacer: …". macOS has no
  verified no-root check: the token is sent and relies on the rotation above. If the OS table
  cannot be read, the request proceeds (logged at debug level).
- **Browsers** get a session, never the token. Kongtrol (tray "Abrir panel", the tray flyout,
  `kongtrol dashboard`) asks the daemon for a one-time nonce (`POST /api/v1/session/nonce`, token
  required; 256 bits, single use, valid 60 s) and opens `http://127.0.0.1:9741/#k=<nonce>`. The
  page removes the nonce from the address bar and trades it (`POST /api/v1/session`: loopback
  Host, the daemon's own `Origin`, JSON only, failures rate-limited to 10 per minute) for two
  things: `kongtrol_session`, a separate random 256-bit id in an `HttpOnly; SameSite=Strict;
  Path=/` cookie, and a 256-bit **page token** in the answer, which the page keeps in
  `sessionStorage` and sends as `X-Kongtrol-Session`. Every request needs both. Browsers send
  cookies to every port of 127.0.0.1 (RFC 6265 §8.5), so a program of another local user on
  another port could receive the cookie; it cannot read `sessionStorage`, which belongs to the
  daemon's origin (scheme, host and port) only. The live feed's WebSocket uses a ticket from
  `POST /api/v1/ws-ticket` (cookie + header; valid 30 s, single use, bound to the session) plus
  the daemon's `Origin`. Sessions live only in the daemon's memory (a restart ends them), expire
  after 12 h without use and after 7 days in any case, and only SHA-256 digests of ids, page
  tokens, nonces and tickets are stored. A second tab or the tray flyout gets its own page token
  on the same cookie (up to 16). `DELETE /api/v1/session` signs out.
- After a daemon restart the hidden tray flyout's session is gone (401): the page hides its last
  states and posts `reauth` (closed message list); the tray reloads it with a fresh one-time nonce,
  at most once per 5 s, through the same loopback-origin check as the first load. The tray also
  reloads it when the api-token changed; it compares a truncated SHA-256 of the token and never
  stores the token ([ADR-0072](adr/0072-live-settings-changes-and-flyout-reauth.md),
  [ADR-0074](adr/0074-tray-flyout-prewarm-and-reauth.md)).
- With `?events=1` the WebSocket also sends `{"type":"changed","rev":N}` after a successful
  mutating request; the frame carries no data (clients refetch through the authenticated API).

Pages (`/` and every dashboard route; `/app/*` only redirects there) never set credentials. Every route, pages included, checks that the Host
header is a loopback name with the daemon's port (DNS rebinding); API requests with an `Origin`
must come from the daemon's own origin, and mutations must be `application/json` (so a
cross-site form or `text/plain` request cannot reach them). The page CSP allows connections only
to its own origin and its one WebSocket origin (`connect-src 'self' ws://<host>:<port>`). A page
opened another way (a bookmark, a new tab without `#k=`, another program) gets 401 and shows:
"Abre el panel desde Kongtrol: el ícono de la bandeja o `kongtrol dashboard`." Reloading the same
tab keeps the session.

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
  tunnel) of a tunnel the helper started; its failures are audited, and a success authorized by
  the caller's profile config (a tunnel Kongtrol did not start) is audited once per tunnel per hour.
- **Tunnels Kongtrol did not start are bound by key.** `wg.stats`, `wg.remove` and
  `wg.allowed_ips` accept such a tunnel only when the caller owns a profile `.conf` with the same
  name **and** the public key derived from its `[Interface] PrivateKey` (Curve25519,
  `golang.org/x/crypto/curve25519`) equals the running interface's public key (`wg show <if>
  dump`). A same-named tunnel of someone else (IT-managed, another app) is denied
  (`wg_not_managed`). The names `all` and `interfaces` (wg(8) keywords) are refused.
- **Trusted WireGuard tools.** Windows: the helper runs `wg.exe` / `wireguard.exe` only from
  `%ProgramFiles%\WireGuard` when the file and its folder are owned by SYSTEM, Administrators or
  TrustedInstaller and no one else has write rights (`adminOnlyWritable`); Unix: root-owned, not
  group/world-writable, in a trusted directory (`trustedBinary`).
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
`Trojan:Win32/Bearfoos.A!ml`. Microsoft only documents that a `!` suffix is an internal
indicator; its own blog uses `!ml` on machine-learning detections, so it is usually a
machine-learning verdict on a file with no reputation yet, not a signature match against known
malware (no official definition: see
[research](research/2026-11-20-antivirus-blocked-binary-in-installers.md)). It's a known, common
false positive for this class of tool, not evidence of anything malicious.

**What it looks like.** Windows Security says it found a threat and quarantined `kongtrol.exe`;
starting it fails with "Operation did not complete successfully because the file contains a virus
or potentially unwanted software" (Windows error 225). The install scripts and the Windows
installer check that the new binary runs before calling the install done
([ADR-0073](adr/0073-installers-stop-when-the-new-binary-cannot-run.md)): `install.ps1` /
`install.sh` stop before touching the installed copy (or put it back) and print what to do; the
Windows installer runs nothing more and says so on its last page. None of them ever turns
Defender off or adds an exclusion.

What to do about it:

- **Restore it only after checking it** (next point): Windows Security > Virus & threat
  protection > Protection history, open the Kongtrol entry and choose **Allow on device** (or
  **Restore**, then Allow on device), then run the installer again. Microsoft's own advice is to
  quarantine when unsure, and every exclusion lowers your protection, so prefer reporting the
  false positive over excluding folders.

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
    --certificate-identity-regexp '^https://github\.com/DerotLuna/vpn-kongtrol/\.github/workflows/release\.yml@refs/tags/v' \
    --certificate-oidc-issuer https://token.actions.githubusercontent.com
  sha256sum --ignore-missing -c checksums.txt
  ```

  The bundle needs cosign 2.4 or newer; on cosign 2.4–2.x add `--new-bundle-format` (3.0 reads
  bundles by default). Releases before v0.6.0 were not signed.

- GitHub [build provenance attestations](https://docs.github.com/en/actions/security-for-github-actions/using-artifact-attestations)
  (SLSA provenance) for every archive, package, installer and binary:
  `gh attestation verify <file> --repo DerotLuna/vpn-kongtrol`.
- An SBOM (SPDX JSON, `*.sbom.json`) for every archive.
- npm packages are published with `--provenance`, shown on npmjs.com.

**The install scripts do this for you** ([ADR-0057](adr/0057-installers-verify-release-signature.md)):
`install.sh` and `install.ps1` always check the download against `checksums.txt`. When `cosign`
2.4+ is installed they first verify `checksums.txt.sigstore.json` with the identity above; a bad
or missing signature stops the install and nothing is installed (`KONGTROL_SKIP_SIGNATURE=1`
only for releases before v0.6.0). Without cosign they print one line saying the signature was not
checked and the command to verify by hand. They never download cosign. They run from a function
called on their last line, so a download cut short runs nothing. `~/.kongtrol/install.manifest`
is writable by any program of the user, so on an update the folder it names is reused only when
it is a default one (`/usr/local/bin`, `~/.local/bin`) or the user can already write it without
`sudo` / administrator rights (as `%LOCALAPPDATA%\kongtrol\bin`); a folder that would need `sudo`
is skipped with a warning and needs `KONGTROL_INSTALL_DIR`. `sudo` therefore never writes where
that file says ([ADR-0063](adr/0063-kill-switch-lan-protected-read-and-release-hardening.md)).

The release jobs restore no build cache (a cache written by a `main` run never feeds a signed
build), use exact tool versions, keep no git credentials after checkout, and the publishing job
(`release`, with its secrets) runs in the protected GitHub environment `release`; the Windows
signing job joins it when the SignPath secrets exist
([ADR-0063](adr/0063-kill-switch-lan-protected-read-and-release-hardening.md)).

Package managers (winget, Scoop, Homebrew, AUR, apt/dnf) check the SHA-256 written in their
manifest. None of these steps sends anything from the user's machine (cosign, when you run it,
contacts the public Sigstore services to fetch its trust root).

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
mode-`0600` token in `~/.kongtrol/api-token` (new on every daemon start); the dashboard never
receives it: it gets its own session, an HttpOnly, SameSite=Strict cookie that only works
together with the page token in `X-Kongtrol-Session` (see "Local panel sign-in"), and
state-changing cross-origin requests are rejected.
The cookie is deliberately not `Secure`: the dashboard is plain HTTP on loopback only, and Safari
has rejected Secure cookies over HTTP even on loopback (https://flaviocopes.com/cookie-not-being-set-in-safari/,
https://bugs.webkit.org/show_bug.cgi?id=281149), which would lock Safari users out. The related
CodeQL alerts are dismissed with this justification; a captured cookie is useless without the
page token.

CodeQL `go/path-injection` on `POST /api/v1/vpns/import` with `"path"`
(`internal/api/handlers_vpn_import.go`) is dismissed as a false positive (2026-11-20): the signed-in
local user picks one of their own `.conf`/`.ovpn` files and the daemon runs as that same user, so no
trust boundary is crossed; the path must still be absolute and clean (no `..`, no UNC, NT object,
`/dev`, `/proc` or `/sys`), a regular file opened without following links and checked to be the
file inspected (`os.SameFile`), within the size cap and with a `.conf`/`.ovpn` name. A root-only
file is never read through the helper (ADR-0058, ADR-0063).

Repository protections: CodeQL (GitHub default setup) on every push and pull request, Renovate for dependency updates, secret scanning
with push protection (enabled 2026-11-20).

## DNS guard recovery

If Kongtrol crashes without cleanly restoring DNS, run:

```bash
# Windows
netsh interface ip set dns "Ethernet" dhcp

# Linux with systemd-resolved: DNS is set on the tunnel link only (resolvectl dns/domain ~.),
# dropped by resolved when the link goes away; to undo by hand:
resolvectl revert <tunnel-interface>

# Linux without systemd-resolved
sudo cp /etc/resolv.conf.kongtrol.bak /etc/resolv.conf

# macOS
networksetup -setdnsservers Wi-Fi empty
```

A tunnel that drops on its own gives its DNS servers back at once (they are only reachable
through it), unless the kill switch holds all traffic for that full-tunnel profile: then lookups
stay on the tunnel's servers so none leaks outside it ([ADR-0058](adr/0058-linux-dropped-tunnel-dns-and-root-only-configs.md)).

With systemd-resolved the tunnel's DNS lives on the tunnel link, and resolved forgets it when the
interface disappears: lookups then go to the physical link's resolver, usually the router in your
local network. So, while a full-tunnel profile is dropped, the kill switch blocks the directly
connected subnets too unless `allow_lan: true` ("Allow the local network"): the lookup to the router
fails instead of leaving outside the tunnel. With `allow_lan: false` a drop therefore also cuts
printers, NAS and other LAN devices until the VPN returns; with `allow_lan: true` they stay
reachable, and so does a LAN resolver (lookups can leave through it during the outage). Always
open: broadcast and multicast (DHCP, mDNS), link-local (IPv6 neighbor discovery) and the VPN
servers. Known limit: a resolver announced at an IPv6 link-local address (RFC 8106 RDNSS) stays
reachable ([ADR-0063](adr/0063-kill-switch-lan-protected-read-and-release-hardening.md)).

## Root-only profile files

A profile file only root can read (`/etc/wireguard/wg0.conf`) is never read through the helper:
that would hand root-only keys to any process of the user without asking. Instead the user gives
one OS permission prompt (`sudo` in a terminal, `pkexec` from the dashboard) to a short-lived
`kongtrol profile read-protected <path>` that runs as root and prints the file only if it is an
absolute `.conf`/`.ovpn`, a regular file with a single link (not a symlink or a hard link), at
most 512 KiB, passes the helper's own WireGuard/OpenVPN checks and has a profile's shape (an
`.ovpn` needs a `remote` and `client` or `tls-client`). Every refusal is the same generic message
with the path you typed: it never says which check failed nor quotes a section, key or line. The
copy is stored 0600 in `~/.kongtrol/configs/` ([ADR-0058](adr/0058-linux-dropped-tunnel-dns-and-root-only-configs.md),
[ADR-0063](adr/0063-kill-switch-lan-protected-read-and-release-hardening.md)).

## Kill switch recovery

```bash
kongtrol down --all    # disables the kill switch automatically
```

If Kongtrol closed abruptly, the helper's lease lifts its blocks within about 20 seconds
(fail-open), and the helper also clears them every time it starts (for example after a reboot).
If traffic is still blocked, remove **only** Kongtrol's rules. Never use `netsh advfirewall reset`,
`pfctl -d` or a chain-less `iptables -F`: they wipe or disable the whole machine's firewall.

```bash
# Windows (PowerShell as Administrator): only the rules named Kongtrol*
netsh advfirewall firewall delete rule name=KongtrolBlock
netsh advfirewall firewall delete rule name=KongtrolKillSwitchBlock
netsh advfirewall firewall delete rule name=KongtrolKillSwitchAllow
netsh advfirewall firewall delete rule name=KongtrolKillSwitchEndpoint
netsh advfirewall firewall delete rule name=KongtrolKillSwitchLAN
netsh advfirewall firewall delete rule name=KongtrolKillSwitchLoopback

# Linux: only Kongtrol's chains (if -X fails, repeat the -D: another jump is left)
sudo iptables -D OUTPUT -j KONGTROL_BLOCK; sudo iptables -F KONGTROL_BLOCK; sudo iptables -X KONGTROL_BLOCK
sudo ip6tables -D OUTPUT -j KONGTROL_BLOCK; sudo ip6tables -F KONGTROL_BLOCK; sudo ip6tables -X KONGTROL_BLOCK
sudo iptables -D OUTPUT -j KONGTROL; sudo iptables -F KONGTROL; sudo iptables -X KONGTROL

# macOS: flush only Kongtrol's anchor; pf stays on for everything else
sudo pfctl -a com.kongtrol -F all
```

## Auto-reconnect watchdog

The watchdog polls every 5 seconds per profile. On an unexpected disconnect it waits `2s → 4s → 8s → … → 5min` (exponential backoff) before each retry, and each reconnect attempt is itself bounded by a timeout so a hung adapter driver can't stall the watcher indefinitely. Intentional disconnects via `kongtrol down` suppress reconnect automatically — the watchdog only fires on drops you didn't request.

## Graceful daemon shutdown

`kongtrol down` asks a running `kongtrol up` daemon to shut down through its own API (`POST /api/v1/shutdown`) rather than killing the process directly. This runs the daemon's normal cleanup — DNS restore, kill switch teardown, history flush — before it exits, on every OS (a plain process kill has no cross-platform graceful-termination equivalent, particularly on Windows). A hard kill is only used as a fallback if the daemon doesn't respond.

## OWASP mapping

What the design covers and what is still open, mapped to
[OWASP Top 10 2021](https://owasp.org/Top10/), [ASVS 4.0.3](https://github.com/OWASP/ASVS/tree/v4.0.3)
and the [Desktop App Security Top 10](https://owasp.org/www-project-desktop-app-security-top-10/).
This is a self-assessment, not an audit or certification.

| Area | Covered | Known gaps |
|---|---|---|
| Access control — A01, DA5, ASVS V4 | Local API needs the file token (native, rotated on every daemon start, sent only after the listener-owner check) or a browser session (cookie + page token); pages give no credentials; Host checked on every route; the helper serves only the installing user or an admin, runs a closed list of operations, binds foreign WireGuard tunnels by key ([ADR-0039](adr/0039-browser-sessions-one-time-nonce.md), [ADR-0040](adr/0040-local-hardening-review-2026-11.md), [ADR-0062](adr/0062-origin-bound-sessions-token-rotation-listener-owner.md)). | macOS clients cannot check who owns the port (rotation only). Any process running as the same user can read `~/.kongtrol/api-token`; this is the OS user boundary, not defended against. |
| Session management — DA2, ASVS V3 | 256-bit random session ids distinct from the token, HttpOnly + SameSite=Strict, bound to a per-page token the daemon's origin keeps in `sessionStorage` (`X-Kongtrol-Session`), one-time 30 s WebSocket tickets, server-side idle (12 h) and absolute (7 d) timeouts, logout, single-use 60 s nonce, in-memory store. | No `Secure` flag (loopback HTTP by design, Safari). |
| CSRF — A01, ASVS V4.2/V13 | SameSite=Strict, `Origin` must be the daemon's own when present (required for the nonce exchange), mutations only as `application/json`. | — |
| Injection / XSS — A03, DA1 | Names restricted to a safe charset; the dashboard is React (escaped by default); the page CSP allows scripts and styles only from the daemon, no `'unsafe-inline'` (ADR-0043). | — |
| Security misconfiguration — A05, DA6 | Loopback-only bind (non-loopback refused); security headers on every response; update and public IP checks off by default; restrictive file modes for state. | — |
| Sensitive data / crypto — A02, DA3, DA4 | Credentials only in the OS keychain; HMAC-signed audit log; TLS 1.2+ with default verification for the two opt-in outbound calls. | — |
| Insecure communication — DA7 | Only the opt-in outbound calls listed above, https only, constrained redirects; everything else is loopback or local IPC. | — |
| Vulnerable components / supply chain — A06, A08, SLSA | `govulncheck` in CI and weekly Renovate updates; release artifacts with checksums, Sigstore signature and build provenance attestations; third-party Actions pinned to commit SHAs; secrets only on the steps that use them. | Install scripts verify the cosign signature of `checksums.txt` only when cosign is installed (ADR-0057); Windows binaries are unsigned until SignPath is configured. |
| Logging — A09, DA10 | The helper audits every privileged request (read-only successes of managed tunnels excepted); HMAC-signed audit log in the daemon. | Browser session creation is not audited. |
| Privileged helper / installer — Desktop Top 10 | Closed operation list, caller identified by the OS, lease-based fail-open cleanup, trusted-binary checks (Unix root-owned; Windows admin-owned and admin-writable only); uninstall deletes only Kongtrol's own files and refuses sudo. | Inno Setup uninstaller working directory not verified ([#39](https://github.com/DerotLuna/vpn-kongtrol/issues/39)). |
