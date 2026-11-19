# Detecting Proton VPN, NordVPN and Surfshark on Windows

Version 1.0 · 2026-11-19 · cited by ADR-0065

Context: the owner reported that "Conectar una VPN que ya uso" (`GET /api/v1/vpns/detect`,
`internal/vpndetect`) listed FortiClient, OpenVPN and WireGuard but not Proton VPN.

## Observed on the owner's machine (2026-11-19, read-only)

- No Proton VPN install: no `C:\Program Files\Proton`, no `C:\Program Files (x86)\Proton Technologies`,
  no `%LOCALAPPDATA%\Programs\Proton`, no "Apps & features" entry (HKLM 64/32-bit and HKCU
  `...\CurrentVersion\Uninstall` searched for "Proton"), no `ProtonVPN Service` / `ProtonVPNCallout`
  services, no Appx package, no running process.
- Only leftovers: an empty `HKLM\SOFTWARE\Proton AG` key (no values or subkeys) and an empty
  `%LOCALAPPDATA%\Proton` folder (last written 2026-11-16). These match an uninstalled app (or an
  install that never finished), not an installed one.
- NordVPN and Surfshark: not installed (no folders, no uninstall entries).
- Conclusion: on this machine the missing entry is correct; the app needs to be (re)installed.
  The code had real gaps anyway (below), fixed in the same change.

## How the apps install (sources)

- Proton VPN (Inno Setup, `Setup/SetupBase.iss` in github.com/ProtonVPN/win-app):
  `DefaultDirName={autopf}\Proton\VPN`, `DisableDirPage=yes`, no `PrivilegesRequired` override
  (Inno default: admin, so `{autopf}` = `%ProgramFiles%`). `ProtonVPN.Launcher.exe` goes into
  `{app}`; the client (`ProtonVPN.Client.exe`), service and drivers go into `{app}\v<version>\`.
  `UninstallDisplayName=Proton VPN`, `UninstallDisplayIcon={app}\ProtonVPN.Launcher.exe`.
  The README says "installed into `C:\Program Files\Proton\VPN\<version>`"; the legacy main
  executable was `ProtonVPN.exe` (`LegacyMyAppExeName`).
- `src/ProtonVPN.Launcher/Program.cs`: the launcher lists `v*` folders, parses them with
  `System.Version` and starts the newest `ProtonVPN.Client.exe` (numeric, not lexical).
- Per-user Proton install under `%LOCALAPPDATA%\Programs\Proton\VPN`: not offered by that script
  (no verificado on real installers); kept as a cheap candidate.
- NordVPN: third-party listings show `C:\Program Files\NordVPN\NordVPN.exe` with components in a
  `<version>` subfolder (e.g. `7.24.5.0\`) and an Inno `unins000.exe`. No official NordVPN doc
  found (no verificado).
- Surfshark: third-party listings show `C:\Program Files (x86)\Surfshark\Surfshark.exe` (sometimes
  `C:\Program Files\Surfshark`), and a custom `APPDIR` at install. No official doc found
  (no verificado).

## Gaps found and fixed

1. `vpndetect.Detect` picked the match with `sort.Strings`: `v4.9.0` sorted after `v4.10.0`, so
   an old leftover version folder could win. Now `comparePaths` compares version segments as
   numbers.
2. No custom install folder support: now the "Apps & features" entry (`InstallLocation`, else the
   folder of `DisplayIcon`) of "Proton VPN"/"ProtonVPN", "NordVPN" and "Surfshark" is read
   (HKLM 64/32-bit and HKCU, read-only).
3. Surfshark had no Windows probe, so it never appeared in the detect list.
4. The adapters' "open the app" paths were separate hard-coded lists; Proton, NordVPN and
   Surfshark now share `vpndetect.ProtonVPNAppPaths` / `NordVPNAppPaths` / `SurfsharkAppPaths`
   (via `appattach.App.WindowsFind`, worked out only when the app is opened, never in the 5 s
   status poll).

## Sources (accessed 2026-11-19)

- https://github.com/ProtonVPN/win-app (README.md)
- https://raw.githubusercontent.com/ProtonVPN/win-app/master/Setup/SetupBase.iss
- https://raw.githubusercontent.com/ProtonVPN/win-app/master/Setup/Setup.x64.iss
- https://raw.githubusercontent.com/ProtonVPN/win-app/master/src/ProtonVPN.Launcher/Program.cs
- https://www.advanceduninstaller.com/ (Proton VPN, NordVPN, Surfshark application pages; third party)
- https://www.file.net/process/nordvpn.exe.html (third party)
- https://freefixer.com/library/file/Surfshark.exe-307592 (third party)
- Inno Setup docs, `{autopf}` and `PrivilegesRequired`: https://jrsoftware.org/ishelp/ (no verificado in this session)
