<img src="../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# FortiClient support matrix

Version 1.0.0 · 2026-11-17

<!-- feature:providers -->
Kongtrol works with **any** FortiClient version: if it can't connect automatically, `kongtrol up`
opens FortiClient and waits for you to connect there, then takes over (attach mode,
[ADR-0006](adr/0006-provider-adapters-attach-mode.md)). What changes per version is only the
**automatic connect** step. Kongtrol detects the installed version (file version of
`FortiClient.exe`, folder from the installer's `INSTALLDIR` registry value) and picks a strategy
([ADR-0020](adr/0020-forticlient-multi-version-strategies.md)). `kongtrol doctor` shows the
detected version, the strategy and its status.

Statuses: **verified** = tested end to end by a maintainer on that version · **reported** = a
contributor reported it working, not re-tested by a maintainer · **no verificado** = nobody has
tested it yet.

## Windows

| FortiClient | Auto-connect strategy | Disconnect | Status | Verified by · date |
|---|---|---|---|---|
| 6.4.10 (6.4.10.1821) | `gui-6.4`: fills the login form (tunnel, user, password, Connect) | Disconnect click + `FortiClient.exe /vpndisconnect` | verified | maintainer · layout measured on 6.4.10 (date not recorded); re-test of the stdin-input script (ADR-0020) **pending** |
| other 6.4 patches | `gui-6.4` | same | no verificado | — |
| 6.0, 6.2 | `gui-6.4` | same | no verificado | — |
| 7.x with `FortiVPN.exe` in the install folder | `fortivpn-cli`: `FortiVPN.exe --cli --connect --tunnel <name>` + fills the native "Autenticación / Authentication" dialog | `FortiVPN.exe --cli --disconnect` | reported | contributor (PR #8) · 2026-11-15, exact 7.x build not recorded; the `FortiVPN.exe` options are not in Fortinet's docs |
| 7.0 / 7.2 / 7.4 without `FortiVPN.exe` | `attach`: opens FortiClient, you connect | you disconnect in FortiClient | no verificado (attach works on any version) | — |
| 5.x, newer than 7.x | `attach` | you disconnect in FortiClient | no verificado | — |
| version not detected | `gui-6.4` (previous behavior) | Disconnect click + `/vpndisconnect` | no verificado | — |

The 6.4.10 install has no `FortiVPN.exe` and no `FortiSSLVPNclient.exe` (checked on the
maintainer's machine, 2026-11-17). Fortinet's own command line, `FortiSSLVPNclient.exe` (in the
separate FortiClientTools package), takes the password as a command-line argument (`-u user:pass`),
which other local processes can read, so Kongtrol doesn't use it
([research](research/2026-11-17-forticlient-versions.md)).

## Linux and macOS

| Platform | Auto-connect strategy | Status |
|---|---|---|
| Linux | `cli`: `forticlientsslvpn --vpnconfig <file>` | no verificado; FortiClient 7.4.4 documents a different CLI (`forticlient vpn connect <name>`) |
| macOS | `cli`: `FortiClientAgent --vpnconfig <file>` | no verificado; FortiClient 7.4.4 documents no VPN command line on macOS |

On both, a failed automatic connect falls back to opening FortiClient and waiting.

## How this is tested

- **Unit tests** (every OS): `SelectPlan` has one case per row above
  (`internal/vpn/forticlient/strategy_test.go`); the generated PowerShell scripts are compared with
  golden files in `internal/vpn/forticlient/testdata/` (regenerate after a reviewed change with
  `go test ./internal/vpn/forticlient -run Golden -update`); FortiVPN.exe arguments never carry
  credentials.
- **Windows CI** (`test (windows-latest)` in `.github/workflows/ci.yml`, blocking): every shipped
  script is parsed by the Windows PowerShell 5.1 parser (never run), and the scripts' stdin decoder
  is run alone on quotes and non-ASCII text. In CI a missing PowerShell fails the test.
- **Real FortiClient**: GUI automation can only be confirmed on a machine with that FortiClient
  version. Fortinet publicly offers only the current installers (the free VPN-only installer from
  FortiClient.com); older builds come from the support portal (login required), so a full VM
  matrix needs a Fortinet support account (no verificado whether an older build can be picked
  without one).
- **Community reports**: if you use another version, open a
  [FortiClient test report](../.github/ISSUE_TEMPLATE/forticlient-report.yml) with the output of
  `kongtrol doctor` and what happened. A row becomes **verified** only after a maintainer reproduces
  it, or **reported** with the reporter and date.

## Adding a version

1. Measure the layout (or confirm the CLI) on that version, in a VM.
2. Add a strategy or extend `SelectPlan` in `internal/vpn/forticlient/strategy.go`, with a test case.
3. Update this table (who verified it, date) and, if the decision changes, write a new ADR.
