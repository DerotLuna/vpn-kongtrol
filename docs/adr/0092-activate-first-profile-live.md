# ADR-0092: Apply the first profile in the current panel and tray session

Status: Accepted · Date: 2026-11-22

The first valid profile is persisted and applied through the adapter sync used
by subsequent edits. The API leaves setup mode once the live sync succeeds.
The listener, browser session and live feed stay active. The user can continue
to the connection or rule step in the same tab.

The tray removes its first-run items individually and builds the full menu
using the existing API client. Removing a menu item closes its click channel.
The exit action runs only on a received click. The helper menu item can remain
usable during the transition.

This supersedes the first-profile restart in
[ADR-0082](0082-resume-first-rule-after-panel-sign-in.md). Draft recovery still
applies if a restart interrupts setup.

[Source notes](../research/2026-11-22-first-profile-live.md)
