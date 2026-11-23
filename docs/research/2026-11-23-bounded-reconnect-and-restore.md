# Research: bounded reconnects, restore at start and sign-in start

Version 1.0.0 · 2026-11-23. Sources checked 2026-11-23. Backs [ADR-0098](../adr/0098-bounded-reconnect-and-restore-at-start.md).

## How mature clients retry

- **NetworkManager.** `connection.autoconnect-retries` is "The number of times a connection should
  be tried when autoactivating before giving up. Zero means forever, -1 means the global default
  (4 times if not overridden)… after a timeout, NetworkManager will try to autoconnect again."
  Source: https://networkmanager.dev/docs/api/latest/nm-settings-nmcli.html
  - The timeout is `AUTOCONNECT_RESET_RETRIES_TIMER_SEC 300`:
    https://raw.githubusercontent.com/NetworkManager/NetworkManager/main/src/core/nm-manager.c
  - `nm-policy.c` resets retries when the carrier comes back ("if cable was unplugged and plugged
    in again, we should try to reconnect"), on wake ("Reset retries on all connections so they'll
    checked on wakeup") and after a successful activation:
    https://raw.githubusercontent.com/NetworkManager/NetworkManager/main/src/core/nm-policy.c
  - Kongtrol takes the bound and the network/wake re-arm. It does not take the 300 s timer,
    which would retry forever.
- **OpenVPN 2.6.** `--connect-retry n [max]` waits 1 s by default, doubles after 5 retries and is
  capped at 300 s. `--connect-retry-max` defaults to unlimited. Source:
  https://openvpn.net/community-docs/community-articles/openvpn-2-6-manual.html
- **Windows RAS.** `RASENTRY.dwRedialCount` / `dwRedialPause` set a bounded redial, and
  `RASEO2_ReconnectIfDropped` is the "Redial if line is dropped" option. Source (archived page):
  https://learn.microsoft.com/en-us/previous-versions/windows/desktop/legacy/aa377274(v=vs.85).
  The default count is not documented (no verificado).
- **Mullvad.** It reconnects with `retry_attempt + 1` and stops only on a blocking error.
  `MAX_ATTEMPTS_WITH_SAME_TUN = 5`. Source:
  https://raw.githubusercontent.com/mullvad/mullvadvpn-app/main/talpid-core/src/tunnel_state_machine/connecting_state.rs
- **AWS retry guidance** (already cited by ADR-0095) recommends explicit attempt limits.

Conclusion: a small bound plus a clear re-arm signal is the established pattern (NetworkManager).
Three attempts match ADR-0095.

## Persisted intent and restore at start

- **Tailscale.** `ipn.Prefs.WantRunning` "indicates whether networking should be active on this
  node", and `tailscale down` sets it to false:
  https://raw.githubusercontent.com/tailscale/tailscale/main/ipn/prefs.go and
  https://raw.githubusercontent.com/tailscale/tailscale/main/cmd/tailscale/cli/down.go.
  That it survives a reboot is inferred from the prefs being stored in the state store (no doc
  quote).
- **WireGuard for Windows.** The tunnel service is created with `StartType: mgr.StartAutomatic`,
  and deactivating a tunnel uninstalls the service:
  https://raw.githubusercontent.com/WireGuard/wireguard-windows/master/manager/install.go.
  From this we infer that an active tunnel comes back after a reboot.
- **Mullvad.** `auto_connect: false` is the default ("If the daemon should connect the VPN tunnel
  directly on start"):
  https://raw.githubusercontent.com/mullvad/mullvadvpn-app/main/mullvad-types/src/settings/mod.rs.
  The help page describes "Launch app on start-up" and "Auto-connect":
  https://mullvad.net/en/help/using-mullvad-vpn-app. The launch default is no verificado.
- **Go `os/signal`.** On Windows, CTRL_CLOSE_EVENT, CTRL_LOGOFF_EVENT and CTRL_SHUTDOWN_EVENT
  reach Notify as `syscall.SIGTERM` (https://pkg.go.dev/os/signal). A daemon stopped by sign-out
  or shutdown is therefore distinguishable from `POST /api/v1/shutdown`.

## Sign-in start per OS

- **Windows Run key.** "The Run key makes the program run every time the user logs on":
  https://learn.microsoft.com/en-us/windows/win32/setupapi/run-and-runonce-registry-keys.
  - `Explorer\StartupApproved\Run` is not documented by Microsoft. The best reference is
    https://www.hexacorn.com/blog/?p=5992 ("populated by Task Manager/Startup tab when we
    enable/disable entries").
  - On this machine (Windows 11, 2026-11-23) the disabled entries read `03` + an 8-byte FILETIME,
    and an entry without a value runs.
  - `02` for enabled is no verificado here.
- **XDG Autostart.** A same-name file in the user's autostart directory overrides the system one,
  and `Hidden=true` means the entry "MUST be ignored" (§2.2–2.3):
  https://specifications.freedesktop.org/autostart/latest/
- **macOS.** `~/Library/LaunchAgents` holds per-user agents, and `RunAtLoad` launches the job when
  it is loaded (launchd.plist(5), mirror: https://keith.github.io/xcode-man-pages/launchd.plist.5.html).
  SMAppService (macOS 13+) is the modern API but needs the app bundle and cgo:
  https://developer.apple.com/documentation/servicemanagement/smappservice.
  The LaunchAgent was chosen because it works from the plain Go binary. Not run on a Mac (no
  verificado).

## Detecting "retrying makes sense" cheaply

- **Network change.** Polling `net.Interfaces()` every 5 s, only while a profile has given up,
  needs no admin and no OS-specific notification API. IPv6 addresses are excluded from the
  fingerprint because temporary addresses rotate without a network change (RFC 8981,
  https://www.rfc-editor.org/rfc/rfc8981).
- **Wake from sleep.** Go's monotonic clock reading does not include time suspended on the
  common platforms (it uses `QueryUnbiasedInterruptTime` on Windows, `CLOCK_MONOTONIC` on Linux
  and `mach_absolute_time` on macOS), while the wall clock does. This is reasoned from the Go
  runtime sources and was not observed across a real suspend (no verificado).
