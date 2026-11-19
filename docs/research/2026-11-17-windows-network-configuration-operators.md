<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Research: Windows "Network Configuration Operators" group

Version 1.0.0 · 2026-11-17

## Question

Could adding the user to the built-in Network Configuration Operators group replace a privileged
helper for Kongtrol's network operations?

## Sources

| # | Source | URL | Accessed |
|---|---|---|---|
| 1 | Microsoft Learn — Understand security groups (Active Directory) | https://learn.microsoft.com/en-us/windows-server/identity/ad-ds/manage/understand-security-groups | 2026-11-17 |

Quotes were captured with an automated fetch tool; re-open the URL before quoting them elsewhere.

## Findings

- [1]: SID `S-1-5-32-556`. Members can "Modify the… (TCP/IP) properties for a… LAN connection,
  which includes the IP address, the subnet mask, the default gateway, and the name servers",
  and "Issue `ipconfig`, `ipconfig /release`, and `ipconfig /renew` commands"; also rename and
  enable/disable LAN connections and manage remote access connections.
- [1] lists no explicit exclusions. That the group cannot manage Windows Firewall rules,
  install the WireGuard tunnel service or start OpenVPN with driver access is **no verificado**
  — not stated on the page.

## Conclusions (reasoned)

- The documented rights cover TCP/IP and DNS settings only; Kongtrol also needs firewall rules
  (kill switch, blocked destinations), tunnel services and hosts-file writes, which the page
  does not grant. Joining an admin-adjacent group is also a broader, standing privilege than a
  closed list of operations. Not chosen.

## Confidence

Medium: the positive list is documented; the limits are inferred from absence.

## Open questions

- Test on a real machine which of Kongtrol's operations succeed for a group member.

## Used by

- [ADR-0005](../adr/0005-privileged-helper-service.md)
