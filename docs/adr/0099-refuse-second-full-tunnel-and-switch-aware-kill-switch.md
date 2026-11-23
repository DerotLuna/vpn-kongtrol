# ADR-0099: Refuse a second full tunnel; never block a full-tunnel drop another VPN replaced

Version 1.0.0 · 2026-11-23

Status: Accepted · Date: 2026-11-23 · Decision-makers: Ruben Luna (owner concern), implemented by agent

## Context and Problem Statement

The owner reported that the kill switch "sometimes blocks everything", for example when
Proton VPN is connected in its app and the user then starts a direct WireGuard profile, or
the reverse. Two tunnels that both take the whole default route break each other:
WireGuard for Windows with a `/0` blocks every packet that does not go through its own
tunnel, and consumer apps apply similar firewall rules. Kongtrol made it worse in two ways:

1. Nothing stopped a manual connect of a full-tunnel profile while another full tunnel
   (a Kongtrol profile or a provider app) was up.
2. When a full-tunnel profile then "dropped" because the other VPN took the route, the
   kill switch blocked every destination except VPN endpoints. Windows Firewall block
   rules match remote addresses on every interface, so the other VPN's traffic was blocked
   too: no internet at all.

ADR-0095 already defers traffic-triggered direct WireGuard starts while Proton is active on
Windows. It does not cover manual connects, other providers, or the kill switch.

## Decision Drivers

- Never cut the internet without a clear notice and one action to get it back.
- Vendors agree on "one full tunnel at a time" (sources in the research note).
- Split tunnels next to a full tunnel are the documented way to run two clients.
- Do not change the watchdog retry policy or `MarkIntended`/`OnIntent` (other work).

## Considered Options

1. Allow, only warn (diagnostics already do, ADR-0086).
2. Warn and ask, with an option to switch (disconnect the other, then connect).
3. Refuse with the reason and what to do; let the user disconnect the other VPN.

## Decision Outcome

Chosen: option 3 for connects, plus a switch-aware kill switch.

- `app.ProfileCarriesAllTraffic` predicts a full tunnel before connecting: Proton VPN,
  NordVPN and Surfshark profiles; WireGuard with `default_egress` (otherwise the daemon
  replaces AllowedIPs with rule destinations and the tunnel subnet); OpenVPN with
  `default_egress` or a `redirect-gateway`/`redirect-def1` line in the `.ovpn`.
  A connected profile also counts when its adapter reports `AllTraffic`.
- `ProfileService.CheckFullTunnelConflict` refuses a full-tunnel connect while another
  Kongtrol full tunnel is connected or a Proton/Nord/Surfshark app tunnel is up outside
  Kongtrol (`appattach.ActiveTunnel`, interface with an address; interfaces of Kongtrol's
  own tunnels are ignored so a `proton-ch.conf` WireGuard tunnel is not mistaken for the
  app). Connecting a provider profile while that provider's app is up stays the attach
  case. The error is a `*config.ValidationError` (`api.err.connect_full_conflict` or
  `_app`, with a `.fix` line) wrapping `app.ErrFullTunnelConflict`.
- Every connect path runs it inside `ProfileService.Connect` (watchdog, scheduler, groups,
  policy connects record it as the tunnel's last error). `POST /api/v1/tunnels/{name}/connect`
  runs it before answering 202 and answers 409 with `error`, `hint`, `error_key`.
- Clients: the dashboard shows it as an error toast with its "what to do" line (a 409 with a
  hint is no longer read as "already connecting"); the tray flyout shows the reason under
  the row; the native tray menu shows reason + hint (`daemonctl.Client.LangSource`, so it is
  in the tray's language); `kongtrol up` prints reason + hint (CLI requests now send
  `X-Kongtrol-Lang`).
- Kill switch: when a full-tunnel profile is not connected but another VPN carries all
  traffic (a connected Kongtrol profile captured as full tunnel, or a provider app tunnel),
  `KillSwitchService` installs no full block, forgets the "was up" state (the user switched
  VPNs) and emits `killswitch.event.superseded` once: which VPN carries the traffic now and
  how to go back. Every other full block keeps its existing notice with how to get the
  internet back (`killswitch.event.blocking_full[_lan]`, ADR-0058/0063). Split-tunnel and
  rule blocks are unchanged.

Option 2 is not chosen now: a switch would disconnect a VPN on the user's behalf, and a
provider app without a CLI cannot be disconnected by Kongtrol at all. It can be added later
on top of the same error key without changing this decision.

## Consequences

- Good: two full tunnels are never started by Kongtrol, in either direction; a VPN switch
  never leaves the user offline; one notice says what happened.
- Bad: an advanced user who deliberately chains two full tunnels cannot do it through
  Kongtrol (no override).
- Watchdog hook (not implemented here, owned by the reconnect work): a reconnect that fails
  with `errors.Is(err, app.ErrFullTunnelConflict)` should wait for the user instead of
  counting as a failed attempt; today it is retried with the normal backoff and fails fast.
- `HoldsAllTraffic` (DNS release) still answers from the state before the next `Apply`, so
  DNS of a superseded tunnel may be kept until that `Apply` (at most 5 s).

## Confirmation

- `internal/app/coexist_test.go`: conflict table (split, attach, own interface, managed,
  app), full-tunnel prediction, refused connect never reaches the adapter, and the kill
  switch not blocking a superseded drop (provider app, other Kongtrol tunnel) while still
  blocking an ordinary full-tunnel drop.
- `internal/api/handlers_connect_conflict_test.go`: 409 with translated hint, no async connect.
- Live Windows validation with Proton VPN and a direct WireGuard profile: **no verificado**
  (issue #51).

## More Information

[Research note, section "Two full tunnels and the kill switch"](../research/2026-11-21-vpn-coexistence.md) ·
[ADR-0086](0086-observe-vpn-coexistence.md) · [ADR-0095](0095-save-rules-paused-and-bound-auto-connect.md) ·
code: `internal/app/coexist.go`, `internal/app/killswitch_service.go`,
`cmd/kongtrol/daemon_coexist.go`.
