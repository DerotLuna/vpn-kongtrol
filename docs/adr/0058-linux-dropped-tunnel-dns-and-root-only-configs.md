<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0058: Linux: DNS of a dropped tunnel, kill switch notice, root-only profile files

Version 1.0.0 · 2026-11-19

Status: Accepted · Date: 2026-11-19 · Decision-makers: project maintainer (owner)

## Context and Problem Statement

Four Ubuntu 26.04.1 LTS reports (issues #37, #6, #5, #7, filed 2026-11-17/18, on v0.4.0–v0.6.0-rc.2):

- **#6** — while the `kongtrol up` live view was open, traffic outside the VPN was blocked; closing
  it restored it. In v0.4.0 `up` ran the tunnels in-process and the kill switch blocked **all**
  non-tunnel traffic while any protected tunnel was **connected** (iptables `OUTPUT … -j DROP`,
  wizard default on); closing the view disconnected and lifted it. Both are gone since ADR-0004
  (detached daemon, `c559b56`, v0.5.0) and ADR-0009 (block only after an unexpected drop, only that
  tunnel's destinations). No current code path changes DNS, routes or firewall for the live view.
- **#37** — when the VPN itself fails, sites that never used it stop working too. Three causes:
  1. The DNS guard kept a dropped tunnel's resolvers (reachable only through the dead tunnel):
     `DNSManager.OnDisconnect` ran only on `down`, so every lookup failed.
  2. On Linux the helper rewrote `/etc/resolv.conf`, which on Ubuntu is systemd-resolved's stub
     link; a crash before restore left the resolvers behind (lease expiry is the only backstop).
  3. By design, a full-tunnel profile (WireGuard `AllowedIPs = 0.0.0.0/0`) with the kill switch on
     blocks everything but the VPN servers and the LAN after a drop; the user was never told.
  Plus a limit outside Kongtrol: `wg-quick` with `DNS =` sends **all** lookups to the tunnel
  (`resolvconf -x` = routing domain `~.`) while the interface exists.
- **#5 / #7** — a WireGuard config in `/etc/wireguard` (root:root 0600 in a 0700 folder) could not
  be used: the user-run daemon cannot read it, the wizard said "file not found", and the helper
  refuses on purpose to reveal a root-only file to a non-root caller (`readTrustedFile`).

## Decision Drivers

- Security first: no new silent path from a user process to root-only secrets; no DNS leak while
  the kill switch holds a full tunnel.
- Standards first: systemd-resolved per-link DNS is how wg-quick (via resolvconf), NetworkManager
  and systemd-networkd hand a VPN its resolvers.
- User guidance: every block and every refusal says what happened and what to do.
- GUI-first, CLI-complete: the copy works from the dashboard, the CLI and the wizard.

## Considered Options

For root-only files:
1. A new helper operation that reads `/etc/wireguard/*.conf` for the caller.
2. **One OS permission prompt (sudo / pkexec) for a short-lived root `kongtrol` that releases the
   file only after the helper's own validators**, then the normal import.
3. Guidance only.

For DNS: keep the guard on drops (status quo) / **release it on drop unless the kill switch holds
the full tunnel** / always release.

## Decision Outcome

1. **Dropped tunnel releases its DNS.** The watchdog's `reconnect_attempt` and
   `manual_reconnect_required` call `DNSManager.Release(profile)` unless
   `KillSwitchService.HoldsAllTraffic(profile)` (full tunnel, kill switch on, not turned off by the
   user): then lookups stay on the tunnel's servers, so none leaves outside it while everything is
   blocked anyway. `ProfileService.Connect` applies them again after the reconnect.
2. **systemd-resolved per link.** When `/etc/resolv.conf` resolves into `/run/systemd/resolve/` and a
   root-owned `resolvectl` exists, the helper runs `resolvectl dns <if> …`, `domain <if> ~.`,
   `default-route <if> yes` (best effort, systemd < 240 lacks it) and undoes with `revert <if>`
   (a link already gone counts as undone). resolved drops a link's settings when it disappears, so
   a crashed tunnel never leaves a dead resolver. Otherwise the old resolv.conf write stays. No wire
   change: `ProtocolVersion` stays 4; `service status` reports an older helper build and
   `kongtrol service install` updates it.
3. **Kill switch notice.** `KillSwitchService.SetOnBlock` fires once per outage when a dropped
   tunnel starts being blocked: `killswitch.event.blocking` ("only its destinations; the rest works")
   or `killswitch.event.blocking_full` ("carried all your traffic … disconnect it to browse now, or
   add rules so the rest does not depend on it"), kind `tunnel_watchdog`, so tray, dashboard and live
   view show it with no client change beyond the dashboard's key map.
4. **Root-only profile files: option 2.** `config.CheckFileReadable` turns EACCES into
   `config.err.file_root_only` (+ "what to do") for profile save, preflight and doctor. The CLI
   (`profile import`, `init`) asks, then runs `sudo -- kongtrol profile read-protected -- <path>`;
   the dashboard sends `{"path","elevate":true}` to `POST /api/v1/vpns/import` and the daemon runs it
   through `pkexec` (Linux; macOS needs a terminal, Windows has no such files). The root child
   (`profileimport.ReadProtected`) accepts only an absolute `.conf`/`.ovpn`, refuses a link or
   non-regular file (Lstat + same-file check after open), caps the size and runs the helper's
   validators; a refusal names a key or line, never the content. The daemon validates again and
   stores 0600 in `~/.kongtrol/configs/`.
   Option 1 was rejected: the helper serves the installing user without a prompt, so it would let
   any process of that user read root-only private keys silently — exactly what root-only
   permissions are for. A directory allowlist adds little on top of content validation (only a
   WireGuard/OpenVPN profile passes) and would block `/etc/openvpn/client` or `/root/…`.

### Consequences

- Good: split-tunnel users keep working DNS when a VPN fails; full-tunnel users learn why they are
  offline and how to get back; `/etc/wireguard` configs work in one step.
- Bad: the WireGuard `DNS =` case still routes all lookups to a dead tunnel until it is replaced
  (wg-quick owns that DNS); documented, with the workaround (drop `DNS =` when internal names are
  not needed). A real split-DNS for WireGuard profiles with policies (routing domains from the
  policies' domains instead of `~.`) is a follow-up: it needs a new helper field and a protocol bump.
- Not verified on a real Ubuntu 26.04 machine (developed on Windows; Linux code cross-compiled and
  vetted, pure parts unit-tested). pkexec from the detached daemon needs the session's polkit agent
  (**no verificado**).

## More Information

- Research: [Linux DNS restore and root-only WireGuard files](../research/2026-11-19-linux-dns-and-root-only-configs.md).
- Code: `internal/privops/resolved.go`, `internal/privops/sys_linux.go`,
  `internal/monitor/dnsmanager.go` (`Release`), `internal/app/killswitch_service.go`
  (`SetOnBlock`, `HoldsAllTraffic`), `cmd/kongtrol/daemon_watchdog.go`,
  `internal/config/files.go`, `internal/profileimport/protected.go`,
  `cmd/kongtrol/protected_read*.go`, `internal/api/handlers_vpn_import.go`,
  `web/ui/src/components/profiles/ImportFile.tsx`.
- Related: ADR-0004, ADR-0005, ADR-0009, ADR-0030, ADR-0042.
