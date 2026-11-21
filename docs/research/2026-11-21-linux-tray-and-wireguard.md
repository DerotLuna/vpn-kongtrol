# Research: Existing Linux WireGuard links and tray startup

Version 1.0.0 · 2026-11-21

| Source | Accessed |
|---|---|
| [WireGuard quickstart](https://www.wireguard.com/quickstart/) | 2026-11-21 |
| [Linux rtnetlink interface descriptions](https://docs.kernel.org/networking/netlink_spec/rt_link.html) | 2026-11-21 |
| [Kongtrol Ubuntu permission report](https://github.com/DerotLuna/vpn-kongtrol/issues/5) | 2026-11-21 |

The current privileged implementation already uses netlink to check link type.
Identifying an existing kernel WireGuard link by type and administrative flag can
be done outside the helper. Querying peer handshake or modifying a tunnel still
uses the helper. The CLI's current install path already waits for an authenticated
health response. On Linux a root-only WireGuard file is a separate condition from a
missing or unavailable helper.

The already-started interface case can be tested without installing a VPN. The
reported user's helper state, original permission error and live tunnel remain
**no verificado**. The tray command uses the same installed binary lookup and
detached process launch as the existing CLI and tray pairing.
