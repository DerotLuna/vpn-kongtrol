# ADR-0111: Activate executable egress on Linux

Status: Accepted · Date: 2026-11-25

## Context

An executable policy and a destination policy can both apply to a flow. The destination rule must still be available to other programs, such as a browser.

## Decision

For direct WireGuard on supported Linux systems, opt in per profile with `app_route_mode` and enable separate app-only rules with absolute executable paths. `prefer_destinations` keeps a more-specific host routing decision where present. `all` moves the app table lookup ahead of ordinary destination routing. A profile can use either this option or native whole-device default egress. The helper owns the scoped classifier, route tables and lease. The original WireGuard file and machine-wide DNS settings remain unchanged.

The policy preview applies the same precedence for configured candidates. IP rules keep longest-prefix precedence. A rule that contains a destination and application still requires both in flow diagnostics. Live app-only egress uses separate rules. Profile and executable changes take effect when the tunnel is reconnected. Linux clears the app routing state on an intentional disconnect or expired daemon lease.

## Validation

The isolated Linux integration tests exercise independent app and destination selection and IPv4 fallback after the other destination route is removed. Native backend Windows signing and interface control remain open. Production testing across Linux installers, distribution kernels and provider accounts remains open.

[Platform and policy evidence](../research/2026-11-25-executable-egress-precedence.md)
