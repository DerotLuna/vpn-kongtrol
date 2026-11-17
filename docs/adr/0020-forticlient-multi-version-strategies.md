<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0020: FortiClient multi-version support — detect the version, pick a strategy, keep attach fallback

Version 1.0.0 · 2026-11-17

Status: Accepted · Date: 2026-11-17 · Decision-makers: project maintainer

## Context and Problem Statement

FortiClient's automatic connect on Windows was GUI automation with coordinates measured on
6.4.10. A contributor (PR #8, FortiClient 7.x) reported that 7.x looks different and contributed a
`FortiVPN.exe --cli` path plus filling a native "Authentication" dialog. FortiClient versions in
use span 6.0–7.4 and two editions; Fortinet documents no VPN command line in the full Windows
client, and its standalone CLI takes the password as an argument
([research](../research/2026-11-17-forticlient-versions.md)). How does Kongtrol support several
versions, and how is that tested?

## Decision Drivers

- Attach mode (ADR-0006) must keep working on any version: automation is best effort.
- Never type credentials into a layout nobody measured; never put credentials on a command line.
- Every claim about a version is verified, reported or marked "no verificado".
- Tests that run without FortiClient catch script and selection regressions.

## Considered Options

1. One GUI script for all versions (status quo).
2. Ask the user for the version (wizard `version:` field) and branch on it.
3. Detect the installed version and pick a strategy per version, with attach fallback.
4. Use Fortinet's `FortiSSLVPNclient.exe` where available.

## Decision Outcome

Chosen: **option 3**.

- `forticlient.DetectInstall`: folder from `app_path`, else the installer's `INSTALLDIR` registry
  value, else the default; version from `FortiClient.exe`'s ProductVersion via the Win32 version
  API; `FortiVPN.exe` presence.
- `forticlient.SelectPlan` (pure, unit-tested per matrix row): `FortiVPN.exe` present →
  `fortivpn-cli` (reported); 6.x or unknown → `gui-6.4` (verified only on 6.4.10); other versions
  → `attach` (no automation: open FortiClient, wait). Linux/macOS → `cli` (no verificado).
- Script inputs (FortiClient path, tunnel, username, password) travel on stdin as base64 lines,
  never in the script text or a command line; PowerShell booleans render as `$true`/`$false`.
- `kongtrol doctor` shows version · strategy · status; anything not verified is a warning.
- [docs/forticlient-support.md](../forticlient-support.md) is the support matrix; the
  `forticlient-report` issue form collects community results.
- The `version:` config field stays informational (detection replaces it).

Option 2 was rejected: users rarely know the exact version and it goes stale after upgrades.
Option 4 was rejected: the password is a command-line argument, and the tool is a separate
download behind a support login.

### Consequences

- Good: 7.x users with `FortiVPN.exe` get automation; versions without a known layout no longer
  get 6.4 clicks; every row has a test.
- Good: a password with quotes (`'`, `’`) no longer breaks or escapes the script.
- Bad: versions without a strategy fall back to manual connect until someone measures them.
- Neutral: `FortiVPN.exe --cli` options are not documented by Fortinet; support rests on the
  contributor's report.

## Confirmation

`internal/vpn/forticlient/strategy_test.go` (matrix rows), `script_test.go` (golden scripts,
stdin encoding, PowerShell booleans), `gui_windows_test.go` (Windows PowerShell 5.1 parser on
every script, Read-Input round trip; blocking in the Windows CI job). A real run on 6.4.10 after
this change is pending (matrix says so).

## More Information

- Code: `internal/vpn/forticlient/{strategy.go,install_windows.go,script.go,gui_windows.go,adapter.go}`,
  `cmd/kongtrol/doctor_forti.go`.
- PR #8 by Jorge Ramírez (UTF-8 BOM, `$true`/`$false`, `FortiVPN.exe --cli`, native auth dialog).
- Builds on ADR-0006 (attach mode).
