<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0065: Provider apps on Windows found by versioned folders and the uninstall entry, one lookup for detect and launch

Version 1.0.0 · 2026-11-19

Status: Accepted · Date: 2026-11-19 · Decision-makers: project maintainer (owner)

## Context and Problem Statement

The owner reported Proton VPN missing from "Conectar una VPN que ya uso"
(`GET /api/v1/vpns/detect`). On that machine Proton was not installed (only an empty
`HKLM\SOFTWARE\Proton AG` key and an empty `%LOCALAPPDATA%\Proton` folder: leftovers), but the
code had real gaps: version folders were compared as text (`v4.9` beat `v4.10`), a custom
install folder was never found, Surfshark had no Windows probe, and each adapter kept its own
hard-coded launch paths.

## Considered Options

1. Add more hard-coded paths.
2. **Read the app's "Apps & features" (Uninstall) entry plus the installer's default folders,
   pick the newest `v<version>` subfolder numerically, and share one lookup between detection
   and the adapters' "open the app".**

## Decision Outcome

Option 2. `internal/vpndetect` `ProtonVPNAppPaths`/`NordVPNAppPaths`/`SurfsharkAppPaths`
(read-only registry: HKLM 64/32-bit and HKCU `…\CurrentVersion\Uninstall`, `InstallLocation`,
else the folder of `DisplayIcon`); Proton's layout follows its Inno Setup script and launcher
(`ProtonVPN.Launcher.exe` in `%ProgramFiles%\Proton\VPN`, client in `v<version>\`, newest by
`System.Version`). Adapters reach it through `appattach.App.WindowsFind`, evaluated only when
the app is opened, never in the 5 s status poll.

### Consequences

- Good: custom folders and new versions are found; Surfshark appears on Windows.
- Bad: NordVPN and Surfshark layouts come from third-party listings (**no verificado**); a
  per-user Proton folder is a guess kept as a cheap candidate.

## More Information

- Code: `internal/vpndetect/apps*.go`, `internal/vpndetect/probes_windows.go`,
  `internal/vpn/{appattach,protonvpn,nordvpn,surfshark}`.
- Research: [2026-11-19-windows-vpn-app-detection](../research/2026-11-19-windows-vpn-app-detection.md).
