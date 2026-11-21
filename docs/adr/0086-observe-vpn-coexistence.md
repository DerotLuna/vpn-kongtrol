# ADR-0086: Report observed VPN coexistence signals

Version 1.0.0 · 2026-11-21

Status: Accepted · Date: 2026-11-21

## Context and Problem Statement

Users can have several VPNs installed. A failed connection can result from competing
routes, provider-managed firewall or DNS settings, or a client that fails independently.
Kongtrol needs to guide troubleshooting without taking control of third-party protections.

## Considered Options

1. Disable other clients or rewrite routes automatically.
2. Warn whenever several clients are installed.
3. Compare active interfaces and currently visible routes in the existing diagnostic.

## Decision Outcome

Use option 3. Inspect active interfaces locally when the user runs Diagnostics. Report
each address family where two detected VPN interfaces advertise complete coverage
(`/0` or both complementary `/1`s). Report Cisco's vendor limitation when AnyConnect
and another detected tunnel are both active. Explain how to isolate the failing client.

Preserve the user's routing, DNS, security and corporate policies. Installed clients
alone do not trigger warnings. Provider identification depends on available interface
data and the running Kongtrol session. DNS ownership and firewall conflicts still
need endpoint-aware evidence. The next useful extension is an opt-in local before/after
comparison of route selection and owned DNS and filtering state during a failed attempt.

## Confirmation

[The diagnostic tests](../../internal/doctor/coexistence_test.go) cover dual-stack
defaults, complementary halves, ordinary physical defaults and a split-tunnel
interface. See the [investigation and reproduction plan](../research/2026-11-21-vpn-coexistence.md).
