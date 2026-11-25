# ADR-0110: Integrate five consumer VPN apps

Version 1.0.0 · 2026-11-24

Status: Accepted · Date: 2026-11-24 · Decision-makers: project maintainer

## Context and Problem Statement

Users want more provider apps available from Add VPN, the wizard and CLI. The app handles credentials, protocol preferences and the actual connection. Kongtrol needs an identifiable network interface to apply routing.

## Decision Drivers

- Follow roadmap priority and reuse the existing attach lifecycle.
- Never automate consumer app graphics or ask for an elevated terminal for Kongtrol.
- Use provider-specific interface names and allow an explicit override for ambiguous interfaces.
- Preserve the existing full-tunnel conflict check and local-only behavior.

## Considered Options

1. Official app attach with a bounded user-session CLI where supported.
2. Direct-config imports only.
3. Automate the apps or elevate the provider CLI.

## Decision Outcome

Implement option 1 for Mullvad, Private Internet Access, ExpressVPN, Windscribe and CyberGhost. The provider packages register installation hints, a matcher and, where applicable, a command plan. ExpressVPN on Windows and CyberGhost use manual attach. Region selection uses the vendor CLI and the provider's stored account. Use the existing WireGuard or OpenVPN importer for option 2.

## Consequences

A provider interface must be discoverable or explicitly selected. ExpressVPN Lightway Turbo on Windows does not expose an attachable interface. Provider-owned DNS, account settings, background preferences, firewall and protocol choices stay under the provider's control. The desktop adapters use the shared connection, disconnect, watchdog and conflict paths.

## Confirmation

Automated tests exercise provider registration, configuration, installation probes, coexistence eligibility, matcher specificity, explicit interface selection and CLI argument plans. Platform-specific live checks are recorded in the research note.

## More Information

- [Provider and platform research](../research/2026-11-24-five-provider-apps.md)
- [Shared desktop adapter](../../internal/vpn/appattach/desktop.go)
