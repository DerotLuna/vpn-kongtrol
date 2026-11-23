# ADR-0096: Repair the Windows network helper during upgrades

Status: Accepted · Date: 2026-11-22

Windows Setup checks the network helper before resuming VPNs on updates.
With the helper task selected, a compatible helper is reused. An absent,
unreachable or incompatible helper uses the existing installation and
verification steps. The Windows permission prompt remains part of that path.
The setup task respects the saved choice and explicit installer flags.

[Research](../research/2026-11-22-rule-activation-and-vpn-coexistence.md)
