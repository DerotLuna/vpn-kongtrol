# ADR-0120: Profile lists and Windows picker cancellation

Status: Accepted · Date: 2026-11-25 · Decision-maker: owner

Research: [profile list and picker](../research/2026-11-25-profile-list-and-picker.md).
Refines ADR-0117 for Windows cancellation.

## Decision

- Inicio and the tray flyout call the entries VPN profiles, show their known
  provider and connection type, and never guess an unknown provider.
- For long lists, search by profile name or known type and show ten profiles,
  with twenty more on request. Connected profiles remain visible. At opening,
  connected profiles come first, then the existing local connection history;
  order remains stable while the user interacts with the view.
- When a Windows picker request is cancelled, post `WM_CLOSE` to the window
  identified by its title and owning process. Continue waiting for the window
  if cancellation preceded its creation, until the picker returns.

## Confirmation and limitations

The opt-in Windows desktop tests checked that the picker stays visible, closes
on cancellation, and returns a selected temporary file. They did not exercise
the whole browser-to-daemon path. The native tray menu does not yet provide
search or the new recent ordering.
