# ADR-0088: Open the tray on demand and reuse active Linux WireGuard links

Version 1.0.0 · 2026-11-21

Status: Accepted · Date: 2026-11-21

## Decision

Expose `kongtrol tray` as an explicit desktop action independent of `up`. Find the
installed companion beside the CLI or on PATH. Both clients still use the existing
local daemon and closing a view leaves the tunnels running.

On Linux, identify an already up kernel WireGuard interface by link type and flags
through the existing rtnetlink dependency. Skip redundant startup for this case.
New links and deliberate restart still use the existing helper. Keep handshake
health separate from presence of the link. The dual-stack default-route opt-in
requires its native setup lifecycle.

See the [supporting research](../research/2026-11-21-linux-tray-and-wireguard.md).
