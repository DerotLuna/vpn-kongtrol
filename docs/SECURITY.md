# Security Model

## No telemetry, no external calls

Kongtrol is 100% local. There is no phone-home, no analytics, no update-check ping, no external
API — the daemon, CLI, tray app, and embedded dashboard talk to your VPN clients, your OS network
stack, and `127.0.0.1`, and nothing else. Credentials go straight from your prompt into the OS
keychain (`internal/config`) and are never written to disk in plaintext or logged. The audit log
(`internal/security`) is local, HMAC-signed for tamper-evidence, and never leaves the machine. You
can verify this yourself — the whole codebase is open source; grep for `http://` / `https://`
outside of test fixtures and adapter vendor docs and you'll find nothing that ships data out.

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


| Layer | Windows | Linux | macOS |
|---|---|---|---|
| **Kill Switch** | `netsh advfirewall` block + tunnel allow | `iptables OUTPUT -j DROP` + ACCEPT | `pf` anchor at `/etc/pf.anchors/kongtrol` |
| **DNS Guard** | `netsh interface ip set dns` per interface | `/etc/resolv.conf` rewrite (backup kept) | `networksetup -setdnsservers` per service |
| **Credentials** | Windows Credential Manager | libsecret / D-Bus Secret Service | macOS Keychain |
| **Audit log** | HMAC-SHA256 per entry; signing key in OS keychain | ← same | ← same |
| **Dashboard** | Localhost-only bind (`127.0.0.1`) | ← same | ← same |

Every subprocess the security layer shells out to (`pfctl`, `iptables`, `networksetup`, `netsh`, elevated PowerShell on Windows) runs under a bounded timeout, so a hung system call can't block the daemon's shutdown or leave the kill switch / DNS guard in an indeterminate state.

## Toggling from the dashboard

The dashboard's **Security** page can flip Kill Switch and DNS Guard on/off live (`POST
/api/v1/security/killswitch`, `POST /api/v1/security/dnsguard`) and set a per-profile kill switch
override (`PUT /api/v1/vpns/{name}/killswitch`, `"override": "inherit"|"on"|"off"`). Each toggle
persists to `kongtrol.yaml` *and* immediately re-applies enforcement — for the kill switch that's a
re-run of `KillSwitchService.Apply()` (`internal/app/killswitch_service.go`); for DNS guard it's
`applyDNSGuardState()` (`cmd/kongtrol/main.go`), which mirrors the same enable gate `ProfileService.Connect`
uses. Nothing here is exposed on the network by default — the dashboard only binds to
`127.0.0.1` unless you've explicitly overridden the bind address (see [CONFIGURATION.md](CONFIGURATION.md)).

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
