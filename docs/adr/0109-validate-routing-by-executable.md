<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0109: Validate routing by executable and subprocess

Version 1.2.0 · 2026-11-24

Status: Proposed · Date: 2026-11-24 · Decision-makers: project maintainer

## Context and Problem Statement

Users want to assign an executable and the subprocesses it launches to a VPN profile. For example, `Orca.exe` and the programs it starts should use one profile, while an independently launched `claude.exe` uses another. Kongtrol resolves program patterns to profiles in diagnostics. Enforcing that assignment requires the OS to identify the process instance behind the connection.

## Decision Drivers

- Preserve simultaneous destination-based and application-based policies.
- Allow a rule to select an executable alone or include its subprocesses.
- Keep the application in the user session and put privileged network operations behind the helper.
- Demonstrate the observed egress and disconnect behavior before presenting a program as routed in the dashboard or CLI.

## Considered Options

1. Classify applications at launch with Linux cgroup v2 and choose a profile table with policy routing. An isolated WireGuard network namespace can be used to compare behavior.
2. Learn destination rules from programs or from blocked traffic. Shared addresses still use the same route for every caller.
3. Intercept flows through a WFP driver on Windows or a Network Extension on macOS. Each requires a separate compatibility and distribution study.

## Decision Outcome

The current roadmap scope is direct assignment by executable. Use option 3 as the Windows feasibility path for executables opened in the user's normal workflow. Linux needs process-aware classification to distinguish executables inside a cgroup. On macOS, test direct flow identity and outgoing interface selection. Defer the proposed per-rule inheritance switch and its precedence rules. [GitHub issue #64](https://github.com/DerotLuna/vpn-kongtrol/issues/64)

## Consequences

The prototype can reuse the existing matcher for selecting root executables. Flow enforcement needs process instance identity to distinguish an inherited assignment from an independent launch of the same executable. Connections made by an existing shared service require separate attribution.

## Confirmation

Assign an executable and test that traffic from a different child executable follows its own rule or normal routing. Test two clients using the same destination IP, port and transport, first with one VPN and the normal connection, then with two profiles where available. Observe both egress paths. That test exposes accidental use of shared destination routes. Include TCP, UDP, IPv4, IPv6, DNS and profile disconnect. Inheritance tests remain in the research note for the later phase. Linux cgroup membership was observed in WSL 2. Live routing results are **no verificado**.

The Linux prototype also verified executable-only socket marks, distinct IPv4 UDP egress interfaces for the same destination, and blocking after an interface disappears in an isolated network namespace. End-to-end routing through Kongtrol profiles is **no verificado**. [Test details](../research/2026-11-24-per-app-routing.md#linux)

## More Information

- [Platform research and sources](../research/2026-11-24-per-app-routing.md)
- [Roadmap issue #64](https://github.com/DerotLuna/vpn-kongtrol/issues/64)
