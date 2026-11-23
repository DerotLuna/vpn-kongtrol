# Research: knowing the network is ready at sign-in

Version 1.0.0 · 2026-11-23. Sources checked 2026-11-23. Backs [ADR-0101](../adr/0101-guided-restore-at-start-and-network-wait.md).

## Problem

The tray starts at sign-in and starts the daemon. At that moment Wi-Fi or DHCP is often not
ready. A restore connect made then fails and spends the watchdog's three attempts (ADR-0098) on
nothing. Kongtrol runs as the user, without admin, on Windows, Linux and macOS.

## What each OS offers

- **Linux (systemd).** `network-online.target` is the usual "wait for the network" target, but
  user units cannot use it. The Arch Wiki line quoted in the forum thread: "User units can not
  reference or depend on system units or units of other users." So `Wants=`/`After=
  network-online.target` in a user unit does nothing. Source:
  https://bbs.archlinux.org/viewtopic.php?id=278852
- **Linux (NetworkManager).** `nm-online` waits until NetworkManager reports a connection, with
  a timeout. It only works where NetworkManager runs (not systemd-networkd, not every distro).
  Source: https://developer.gnome.org/NetworkManager/1.24/nm-online.html. **No verificado:** the
  URL now answers 301 to the general NetworkManager API index (networkmanager.dev), so the page
  itself was not read.
- **Windows.** `INetworkListManager` has `get_IsConnectedToInternet` and `GetConnectivity`
  ("returns the overall connectivity state of the machine"). It is a COM interface.
  `NotifyRouteChange2` "registers to be notified for changes to IP route entries on a local
  computer". It needs a callback in the same process. Sources:
  https://learn.microsoft.com/en-us/windows/win32/api/netlistmgr/nn-netlistmgr-inetworklistmanager,
  https://learn.microsoft.com/en-us/windows/win32/api/netioapi/nf-netioapi-notifyroutechange2
  (both read).
- **macOS.** `NWPathMonitor` (Network framework) reports path changes and whether the path is
  satisfied. It needs cgo or Objective-C. **No verificado:** the Apple page was not opened.

## Conclusion

Each OS has its own signal: one COM API on Windows, NetworkManager only on some Linux systems,
and the Network framework only through cgo on macOS. The CLI and the daemon are built with
`CGO_ENABLED=0`, and three code paths would each need their own tests.

A pure-Go check works the same on every OS, needs no admin and sends nothing:

1. `net.Dial("udp", "192.0.2.1:9")` (RFC 5737 TEST-NET-1; also `[2001:db8::1]`, RFC 3849).
   A UDP "connect" only asks the routing table. It fails when there is no route, and otherwise
   gives the local address that would be used.
2. That address must not be loopback, link-local (169.254/16 or fe80::/10, which is what an
   interface gets with no DHCP answer) or unspecified.
3. The interface with that address must be up and must not be one of Kongtrol's own tunnels.

The daemon polls this every 2 s for up to 90 s. Without a network it then waits for the next
change of `monitor.NetworkFingerprint`, the same signal the watchdog uses to re-arm (ADR-0098).
Polling every 2 s costs one socket and one interface listing.

Verified here: `NetworkReady` returns true on this Windows 11 machine with a network
(`go test ./internal/monitor -run NetworkReadySmoke -v`). **No verificado:** the offline case on
each OS, and Linux and macOS in general (this machine runs Windows).
