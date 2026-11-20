<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# Install scripts: CLI and tray by default

Version 1.0.0 · 2026-11-20 · Backs [ADR-0075](../adr/0075-install-scripts-install-cli-and-tray.md)

All sources accessed 2026-11-20.

## Question

`install.ps1` installed only `kongtrol.exe`, while `install.sh` and every other channel install
the tray too. The owner decided (2026-11-20) that every path gives the same product: CLI + tray
by default, CLI only on request, no question asked. How do others do it, which Windows assets
carry the tray, how to tell "no desktop", and how to coexist with the Windows installer?

## Install everything by default, opt out with a flag

| Fact | Source |
|---|---|
| rustup installs the `default` profile unless told otherwise: "The default profile includes all of components in the minimal profile, and adds rust-docs, rustfmt, and clippy"; `--profile minimal` opts out. | https://rust-lang.github.io/rustup/concepts/profiles.html |
| `rustup-init` arguments pass through the piped script with `sh -s -- <flags>` (e.g. `--no-modify-path`). Kongtrol uses the same form for `--cli-only`. | https://rust-lang.github.io/rustup/installation/other.html |
| Docker Desktop's Windows installer flags (`--user`, `--quiet`, `--accept-license`, `--backend`, `--no-windows-containers`, …) have none that leaves the app out: one product per install. That no such flag exists is read from the documented list (reasoned). | https://docs.docker.com/desktop/setup/install/windows-install/ |
| Tailscale's Linux one-liner installs "the Tailscale client" through the distribution's package manager; that the package carries both daemon and CLI is **no verificado** from that page. | https://tailscale.com/kb/1031/install-linux |
| A piped `iwr \| iex` / `curl \| sh` often has no terminal to answer a question (stdin is the script); install.sh already asks only on `/dev/tty` and only for a downgrade. | `install.sh` `has_tty`, ADR-0046 |

## Which release asset carries the Windows tray

| Fact | Source |
|---|---|
| v0.7.0-rc.3 assets (`gh release view v0.7.0-rc.3 --repo DerotLuna/vpn-kongtrol --json assets`): no raw tray `.exe`; `kongtrol_windows_amd64.zip` and `kongtrol_windows_arm64.zip` both contain `kongtrol.exe` and `kongtrol-tray.exe` (`unzip -l`, measured), and both zips are in `checksums.txt`. | release v0.7.0-rc.3; `.goreleaser.yaml` archive `cli-archive` (`ids: [kongtrol-cli, kongtrol-tray]`) |
| The raw `kongtrol-windows-<arch>.exe` (archive `cli-binary`) is the CLI only. | `.goreleaser.yaml` |
| Using the existing zip works for every release already published; a new raw tray asset would need a `.goreleaser.yaml` change and would not exist in older releases. | reasoned |
| `System.IO.Compression.ZipFile` + `ZipFileExtensions.ExtractToFile` extract the two entries by name in Windows PowerShell 5.1 and PowerShell 7.6 (measured with the real rc.3 zip); on Linux PowerShell (.NET 8) the Unix mode bits stored by `zip` survive the extraction (measured in the `mcr.microsoft.com/powershell` container). | measured here |

## Telling "no desktop"

| Fact | Source |
|---|---|
| Windows Server 2012 and later: Microsoft's recommended check is `HKLM\Software\Microsoft\Windows NT\CurrentVersion\Server\ServerLevels`: `ServerCore=1` alone = Server Core; `Server-Gui-Shell=1` = full desktop; `NanoServer=1` = Nano Server. The page is archived (no longer updated). | https://learn.microsoft.com/en-us/previous-versions/windows/desktop/legacy/hh846315(v=vs.85) |
| A Microsoft answer also points at `InstallationType` ("Server Core" / "Server") and warns the old document may not match future releases. Kongtrol uses ServerLevels (the documented recommendation); not tried on a real Server Core (**no verificado**). | https://learn.microsoft.com/en-gb/answers/questions/1855601/how-can-i-distinguish-between-windows-server-windo |
| Linux: the desktop entry spec defines `$XDG_CURRENT_DESKTOP` ("a colon-separated list of strings", matched against `OnlyShowIn`/`NotShowIn`), which the autostart spec reuses for autostart entries. A graphical session also sets `DISPLAY` (X11) or `WAYLAND_DISPLAY` (Wayland); that an SSH login or a server has none of the three is common practice (reasoned, not a spec). | https://specifications.freedesktop.org/desktop-entry/latest/recognized-keys.html , https://specifications.freedesktop.org/autostart/latest/ |
| An SSH session to a desktop machine also has none of them, so "no desktop" is decided again on every run and never stored as the user's choice; an update keeps a tray that was installed. | reasoned |

## Coexisting with the Windows installer

| Fact | Source |
|---|---|
| Inno Setup names the uninstall key `<AppId>_is1`; with `PrivilegesRequired=lowest` it lives in HKCU. `kongtrol.iss` reads `DisplayVersion` / `InstallLocation` from it in `InitializeSetup`. On this machine `install.ps1` found the real key of an installed 0.7.0-rc.3 (measured). | `packaging/windows/kongtrol.iss`; https://jrsoftware.org/ishelp/topic_setup_appid.htm |
| `/SILENT` hides the wizard but shows progress; `/VERYSILENT` hides both. Running the new setup over an install is the documented update path of `kongtrol.iss` (ADR-0046). | https://jrsoftware.org/ishelp/topic_setupcmdline.htm |
| The Inno copy writes the Start menu shortcut `{autoprograms}\Kongtrol` and HKCU Run value `Kongtrol`; a script copy with the same names would overwrite them and leave two `kongtrol.exe` (two versions talking to one daemon). | `kongtrol.iss` `[Icons]`, `[Registry]` |

## Probing the tray (ADR-0073)

| Fact | Source |
|---|---|
| `kongtrol-tray` had no flags except `--open`; any other argument starts the tray (and a daemon). | `cmd/kongtrol-tray/main.go`, `open_request.go` |
| A `-H=windowsgui` Go binary writes to a redirected stdout: `& kongtrol-tray.exe --version 2>$null \| Select-Object -First 1` printed `9.9.9`, exit 0, in PowerShell 7.6 and Windows PowerShell 5.1 (measured with a local build). | measured here |
| Reading a file back makes on-access antivirus scanning look at it; whether Defender's ML detections (`!ml`) fire on read as they do on start is **no verificado**. | reasoned |
| `& $exe version 2>$null \| Select-Object -First 1` left `$LASTEXITCODE` unset in PowerShell 7 on Linux (the pipeline stopped the program early, measured in the container); reading the whole output first keeps the exit code and the pipeline's Win32 error shape. | measured here |
