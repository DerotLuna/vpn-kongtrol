<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0097 · A user disconnect is never a kill switch drop

Version 1.0.0 · 2026-11-23

Status: Accepted · Date: 2026-11-23 · Decision-makers: project maintainer

## Context and Problem Statement

Issue #46: on Windows the user disconnected the OpenVPN profile `ekiipago-ovpn` from the panel and
Inicio › Avisos showed `killswitch.event.blocking` ("se cayó · el kill switch bloquea solo sus
destinos … Kongtrol la reconecta solo"). A disconnect the user asked for must never block, notify
or reconnect.

`ProfileService.Disconnect` records the intent (`OnIntent(name, true)` → `KillSwitchService.NoteIntent`)
before `adapter.Disconnect`. But `KillSwitchService.Apply` re-armed every tunnel it saw connected
(`st.wasUp, st.intended = true, false`) unless an ADR-0084 `disconnecting` counter was set, and that
counter exists only for profiles with rules that block when the tunnel is down. Two orderings wiped
the intent:

- An `Apply` while the adapter is still going down. The 5 s ticker (`killSwitchSvc.Run`) or a config
  save lands while OpenVPN waits for the helper to stop openvpn, or while a provider app waits for
  the user to disconnect (`vpn.ErrManualDisconnect`, minutes).
- An `Apply` that read the statuses (unlocked, before `s.mu`) just before `NoteIntent` and applied
  them after it.

The next `Apply` (the one in `afterDisconnect`) then saw a disconnected tunnel with `wasUp` and no
intent: a drop. It blocked the tunnel's destinations and emitted the notice. The watchdog was not
affected: `MarkIntended` runs before `adapter.Disconnect` and `paused` is checked after the status
read. Every surface uses this path: panel, tray, flyout, `kongtrol down <profile>`, group disconnect
and the scheduler call `ProfileService.Disconnect` through the daemon. OpenVPN, WireGuard and
provider (attach) profiles all go through it.

## Considered Options

- Set the `disconnecting` counter for every user disconnect. This closes the in-progress window,
  but a status read taken before the counter and applied after `finish` still re-arms the tunnel.
- Keep a per-profile intent sequence and drop observations older than it. This covers stale reads
  but not an `Apply` during the disconnect.
- After a user disconnect, re-arm only once the tunnel was seen down and then up again (`downSeen`).

## Decision Outcome

Chosen: the third option, the smallest change in `internal/app/killswitch_service.go`.
`NoteIntent` clears `downSeen`. `Apply` sets it when it sees an intended tunnel down. A connected
reading re-arms the tunnel (`wasUp`) only when there is no intent or `downSeen` is true. Both
orderings above are covered. A tunnel brought up again outside Kongtrol after being seen down (for
example in the provider app) is protected again as before. `Connect` still resets the intent
through `OnIntent(name, false)`.

## Consequences

- A user disconnect never blocks, never shows `killswitch.event.blocking` and is never reconnected.
- A provider tunnel left connected after `down` (the user never closed it in the app) is not
  protected by the kill switch until the next `up` or until it is seen down and up again.
- ADR-0084 rule protection during a disconnect is unchanged. Its blocks are reported as
  `PolicyBlocking`, never through `SetOnBlock`.

## Confirmation

`internal/app/user_disconnect_test.go`:

- `TestKillSwitch_ApplyDuringUserDisconnectIsNotADrop` covers the tick during the disconnect and
  the re-arm after an external reconnect.
- `TestKillSwitch_StatusReadBeforeIntentIsNotADrop` covers the stale read.
- `TestDisconnect_UserDisconnectNeverDropBlockOrReconnect` wires `ProfileService`, the kill switch
  and the real watchdog. Its adapter runs `Apply` inside `Disconnect` and stays connected until it
  ends.

All three failed before the change with the exact notice and block set. Reproduction on Windows
with a real OpenVPN and WireGuard profile is **no verificado**.

## More Information

[Investigation](../research/2026-11-23-user-disconnect-kill-switch.md) · issue #46 · ADR-0018,
ADR-0058, ADR-0084.
