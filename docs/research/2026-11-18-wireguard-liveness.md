<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# Research: detecting a dead WireGuard tunnel without an admin terminal

Version 1.0.0 · 2026-11-18

## Question

Issue #33: the WireGuard profile `brainbox` stopped working while `kongtrol status` and the
dashboard kept showing "connected". How do mature tools decide that a WireGuard tunnel is alive,
and how can Kongtrol read that signal from the unprivileged daemon on Windows, Linux and macOS?

## Root cause (code, verified)

`internal/vpn/wireguard/adapter.go` `Status()` ran `wg show <iface>`; it needs Administrators /
SYSTEM on Windows (and root on Unix), so for the unprivileged daemon it always failed and
`Status()` fell back to `ifaceExists(iface)`. The WireGuard interface stays up while the peer is
unreachable, so the tunnel was always "connected". `TunnelInfo()` had the same problem with
`wg show <iface> transfer`.

## Sources

| # | Source | URL | Accessed |
|---|---|---|---|
| 1 | wg(8) man page (`show … dump` format) | https://git.zx2c4.com/wireguard-tools/about/src/man/wg.8 | 2026-11-18 |
| 2 | wireguard-go `device/constants.go` (timers) | https://git.zx2c4.com/wireguard-go/tree/device/constants.go | 2026-11-18 |
| 3 | WireGuard protocol & cryptography page (timers, keepalive) | https://www.wireguard.com/protocol/ | 2026-11-18 |
| 4 | WireGuard whitepaper, §6 Timers | https://www.wireguard.com/papers/wireguard.pdf | 2026-11-18 |
| 5 | wireguard-windows `docs/enterprise.md` (who may run `wg(8)` against a tunnel) | https://git.zx2c4.com/wireguard-windows/about/docs/enterprise.md | 2026-11-18 |
| 6 | Tailscale `ipnstate.PeerStatus.LastHandshake` | https://pkg.go.dev/tailscale.com/ipn/ipnstate | 2026-11-18 |
| 7 | NetBird — Troubleshooting client issues (`netbird status -d`, "Last WireGuard handshake") | https://docs.netbird.io/how-to/troubleshooting-client | 2026-11-18 |
| 8 | WireGuard mailing list, "Dead peer detection" (2018) | https://lists.zx2c4.com/pipermail/wireguard/2018-May/002926.html | 2026-11-18 |
| 9 | Monitoring vendors' handshake-age alerts (xitoring, tunnelhq) | https://xitoring.com/integrations/wireguard-monitoring · https://app.tunnelhq.com/blog/why-wireguard-port-monitoring-fails/ | 2026-11-18 |

## Findings

- WireGuard has no connection state; liveness is the peer's latest handshake. `wg show <iface>
  dump` prints, per peer, tab separated: public-key, preshared-key, endpoint, allowed-ips,
  latest-handshake, transfer-rx, transfer-tx, persistent-keepalive; the first line is the
  interface (private-key, public-key, listen-port, fwmark) [1]. So the dump holds secrets: only
  selected peer fields may leave the helper.
- Timers: `RekeyAfterTime = 120 s`, `RejectAfterTime = 180 s`, `RekeyAttemptTime = 90 s`,
  `RekeyTimeout = 5 s`, `KeepaliveTimeout = 10 s` [2]. After REJECT_AFTER_TIME a session's keys
  are no longer accepted; handshake initiations without answer are retried every REKEY_TIMEOUT
  (+ jitter) for REKEY_ATTEMPT_TIME [3]. Handshakes happen only when there is something to send
  (data or keepalive) [3]. Whitepaper §6 has the same table [4]; the PDF could not be read by the
  fetch tool, so the section number is **no verificado** from this session (values match [2]).
- Windows: the tunnel service "may be queried … using the standard `wg(8)` command line
  utility", and Local System or Administrator rights are required to do so [5]. So the daemon
  cannot read it unprivileged; the SYSTEM helper can, with the admin-installed `wg.exe` that
  Kongtrol already trusts for `wg.allowed_ips`.
- Tailscale exposes `LastHandshake` per peer; zero means no handshake since the peer was added,
  and idle peers are removed from WireGuard, so an old/zero value on an idle peer is not a fault
  [6]. NetBird shows "Last WireGuard handshake" per peer, refreshed about every 2 minutes on an
  active connection; empty means no handshake completed, and a stale value suggests a broken
  connection [7].
- A common rule is "handshake older than ~3 minutes = peer gone" (handshakes renew every ~2 min
  with traffic; keepalives count as traffic) [8]. Monitoring products warn that 180 s can flap
  around a healthy keepalive tunnel and default to larger thresholds (5 min) [9] (vendor blogs,
  lower weight).
- Not checked in this session (**no verificado**): NetworkManager and systemd-networkd liveness
  handling, the wireguard-windows UI code that renders "latest handshake", wg-quick (a setup
  script; reasoned to do no monitoring), and whether `golang.zx2c4.com/wireguard/wgctrl` reads
  WireGuardNT tunnels on Windows. Also reasoned, not verified here: transfer-tx counts handshake
  initiations (wireguard-go adds every buffer sent to a peer to its tx counter), which is what
  makes "never handshaked but tx > 0" mean "initiations without answer".

## Conclusions for Kongtrol

1. Use the handshake + counters, never the interface alone ([6], [7], [8]).
2. Read them through a new read-only helper op running `wg show <iface> dump` [1], [5]: no new
   dependency, same trusted binary as `wg.allowed_ips`; parse in the helper and return only
   public key, handshake, rx, tx, keepalive.
3. Do not flag an unused tunnel without keepalive (no traffic → no handshake [3], [6]); flag
   when Kongtrol's side keeps sending without answers: stale handshake (> 180 s + 20 s margin)
   with PersistentKeepalive, tx growing with rx flat for 3 × REKEY_TIMEOUT, or no handshake
   30 s after connect while sending.
4. When the counters can't be read, say so ("unknown" + what to do) instead of "connected".
