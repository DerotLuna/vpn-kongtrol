# ADR-0095: Save rules paused and bound traffic-triggered connections

Status: Accepted · Date: 2026-11-22

The Dashboard accepts a list of destinations in one field and saves new rules
paused. A separate action activates the existing routing, blocking and
connection behavior. `disabled: true` preserves the pause in YAML and through
API updates. Existing configurations and CLI automations preserve their
activation defaults. CLI users can opt into a pause at creation and change it
with `policy state`.

The TCP observer shares three attempts per profile, waiting 3 and then 6 minutes
after failures. Each attempt still requires observed traffic. A successful
connection re-arms the observer. With the budget exhausted, the firewall keeps
protecting known destinations and the observer points to manual connection.

On Windows, an active Proton VPN holds traffic-triggered direct WireGuard
starts. Proton's ProTUN interface is recognized. The hold applies before any
tunnel or route changes and keeps the rule's destination protection. The guard
covers the reported pairing. Simultaneous use with other providers still
depends on their routing and filtering.

[Research](../research/2026-11-22-rule-activation-and-vpn-coexistence.md)
