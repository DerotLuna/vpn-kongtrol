# ADR-0121: Startup progress and native profile menu

Status: Accepted · Date: 2026-11-25 · Decision-maker: owner

Research: [startup cancellation and native menus](../research/2026-11-25-startup-and-native-menu.md).
Refines ADR-0098 and ADR-0120.

## Decision

- Register startup connections before opening the API. A client snapshot
  shows a queued or running restart/restore as `connecting` with
  `restoring: true`; the internal adapter status remains unchanged. Ordinary
  startup requests show `connecting` without the restore flag.
- Cancel and wait for an individual queued/running startup connect before
  disconnecting or starting a user-requested replacement. Restore-off
  cancels pending automatic restores, not explicit restart targets. Failures
  leave the transition and are reported. Provider restores use an attach-only
  service path: if the detected tunnel disappears before attachment, the
  adapter is not started and the profile is skipped.
- The native tray orders profiles once when its menu is created: connected,
  then most recently connected if the daemon answers; otherwise by name.
  Fixed items never change the profile they control while a menu is open.
  Known provider/type is shown and, for long lists, an item opens the
  dashboard's profile search. The flyout retains its own search.

## Confirmation and limitations

Automated tests cover queued and running cancellation, state snapshots,
menu ordering and profile labels. The actual native menu and a full
restart with real VPNs require manual verification on each desktop.
