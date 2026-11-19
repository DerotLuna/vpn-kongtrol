<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Research: FortiClient versions, command lines and version detection

Version 1.0.0 · 2026-11-17

## Question

Which FortiClient versions are in use, what official command line each offers (and whether any
takes credentials safely), how the GUI differs, how to detect the installed version, and how to
test Kongtrol's automation across versions. Trigger: PR #8 (contributor on FortiClient 7.x) vs
the maintainer's FortiClient 6.4.10.

## Sources

| # | Source | URL | Accessed |
|---|---|---|---|
| 1 | Fortinet KB 192581 — How to use FortiClient SSL VPN from the CLI | https://community.fortinet.com/t5/FortiClient/Technical-Tip-How-to-use-FortiClient-SSL-VPN-from-the-CLI/ta-p/192581 | 2026-11-17 |
| 2 | FortiClient 7.4.7 Windows release notes (PDF) | https://fortinetweb.s3.amazonaws.com/docs.fortinet.com/v2/attachments/c350818e-4014-11f1-b31d-02356ffb40d9/forticlient-7.4.7-windows-release-notes.pdf | 2026-11-17 |
| 3 | FortiClient 7.2.14 Windows release notes (PDF) | https://fortinetweb.s3.amazonaws.com/docs.fortinet.com/v2/attachments/f050d765-111e-11f1-8b43-d2943efe5b2f/forticlient-7.2.14-windows-release-notes.pdf | 2026-11-17 |
| 4 | FortiClient 7.4.4 admin guide — Linux CLI | https://docs.fortinet.com/document/forticlient/7.4.4/administration-guide/41299 | 2026-11-17 |
| 5 | FortiClient 7.4.4 admin guide — Windows CLI commands | https://docs.fortinet.com/document/forticlient/7.4.4/administration-guide/95591/forticlient-windows-cli-commands | 2026-11-17 |
| 6 | FortiClient 7.4.4 admin guide — macOS CLI commands | https://docs.fortinet.com/document/forticlient/7.4.4/administration-guide/952200/forticlient-macos-cli-commands | 2026-11-17 |
| 7 | FortiClient 7.4.4 admin guide — Appendix A, VPN automation API | https://docs.fortinet.com/document/forticlient/7.4.4/administration-guide/933408/appendix-a-api | 2026-11-17 |
| 8 | Fortinet community forum (not official) — FortiClient 7.2 from the Windows command line | https://community.fortinet.com/t5/Support-Forum/How-to-connect-disconnect-FortiClient-7-2-from-Windows-Command/m-p/315445 | 2026-11-17 |
| 9 | Fortinet community forum (not official) — release channels / lifecycle | https://community.fortinet.com/t5/Support-Forum/Forticlient-Release-Channels/m-p/314034 | 2026-11-17 |
| 10 | Local inspection of the maintainer's FortiClient 6.4.10.1821 install (files, registry), read-only | — | 2026-11-17 |
| 11 | Local PowerShell 5.1 tests (bare `true`, typographic quote, stdin decoding), scratch scripts only | — | 2026-11-17 |

Fortinet pages were read with an automated fetch (curl + pdftotext); re-open the URL before
quoting them elsewhere.

## Findings

**Versions.** 7.0, 7.2 and 7.4 are current Windows branches; newest builds 7.4.7 and 7.2.14 [2, 3].
No FortiClient 7.6 endpoint client was found. Version strings are `major.minor.patch.build`
(e.g. 7.4.7.2003) [2]. Official end-of-support dates were not found (lifecycle page needs login);
a forum post quotes "54 months after GA" [9] — **no verificado**. FortiOS 7.6.3+ drops SSL VPN
tunnel mode on the server side [2].

**Editions.** A free VPN-only installer (`FortiClientVPNSetup_<ver>_x64.exe`) exists alongside
the full, EMS-managed client [2, 3]; for 7.4.4–7.4.7 Fortinet says to keep the 7.4.3 VPN-only
agent [2]. GUI differences between editions and versions (e.g. a separate native
"Authentication" dialog in 7.x) — **no verificado** from Fortinet docs; PR #8 reports such a
dialog on 7.x.

**Command lines (Windows).**
- The full install "cannot be used for command line VPN tunnel access"; the CLI is
  `FortiSSLVPNclient.exe` in the separate FortiClientTools package (`SSLVPNcmdline`) from the
  support portal [1]. Syntax: `FortiSSLVPNclient.exe connect|disconnect -s <profile>
  -h <host>[:port] -u <user>[:<password>] -i -q -m` [1]. The password is a command-line argument,
  readable by other local processes: unsafe. Without credentials a pop-up asks for them [1].
  FortiClientTools still lists it for 7.0.8, 7.2.0, 7.2.14, 7.4.7 [2, 3]; one forum user found it
  missing from a 7.2.4 zip [8] (not official).
- The official Windows CLI page lists only `FortiESNAC.exe` EMS commands, no VPN commands [5].
- The VPN automation COM API is IPsec only, not SSL VPN [7].
- `FortiClient.exe /vpnconnect` / `/vpndisconnect` and `FortiVPN.exe --cli --connect --tunnel` /
  `--disconnect`: no Fortinet source found — **no verificado**. PR #8 reports the FortiVPN.exe
  options working on 7.x.
- The 6.4.10.1821 install has no `FortiVPN.exe` and no `FortiSSLVPNclient.exe` (it has
  `FortiClient.exe`, `FortiTray.exe`, `FortiSSLVPNdaemon.exe`, `FortiVPNSt.exe`, `ipsec.exe`) [10].

**Linux / macOS.** Linux 7.4.4: `forticlient vpn connect <name> --user=<u> --password
[--save-password] [--always-up]`, plus `disconnect`, `list`, `status` [4]; whether `--password`
prompts or takes a value is not stated. macOS 7.4.4 documents only `epctrl` (EMS), no VPN CLI [6].
Kongtrol's current `forticlientsslvpn --vpnconfig` path matches neither — **no verificado**.

**Version detection.** Registry `HKLM\SOFTWARE\Fortinet\FortiClient` holds `INSTALLDIR` (no
version value) on 6.4.10; tunnels under `...\Sslvpn\Tunnels\<name>` [10]. `FortiClient.exe`
ProductVersion is `6.4.10.1821` [10]. MSI product codes change per release [2, 3], so an Uninstall
lookup must match DisplayName, not a fixed GUID. Kongtrol reads the file version resource with the
Win32 version API (no process launched).

**PowerShell pitfalls.** In Windows PowerShell 5.1, `$x = true` runs `true` as a command and
leaves `$x` empty, so the generated `$allowInsecureCert = true` was always false [11]. A
single-quoted string also ends at a typographic quote (U+2019 tested) [11], so escaping only `'`
is not enough for embedded values; values passed as base64 lines on stdin decode intact [11].

**Installers for testing.** Public downloads give the current build only (FortiClient.com
VPN-only installer); older builds and FortiClientTools come from support.fortinet.com Firmware
Download, login required [1]. Whether a past version can be chosen without a support contract —
**no verificado** (forum users say a licence or TAC ticket is needed [9]).

## Used by

ADR-0020.
