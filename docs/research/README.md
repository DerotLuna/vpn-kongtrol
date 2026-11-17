<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# Research notes

Version 1.2.0 · 2026-11-17

Every investigation behind a Kongtrol decision is recorded here with its sources, so a decision
can be re-checked later. Decisions themselves are ADRs in [`docs/adr/`](../adr/README.md).
Rule (`AGENTS.md`, "Standards first, then improve"): cite sources and say when something is
reasoned but unverified.

## Index

| Note | Topic | Used by |
|---|---|---|
| [2026-11-17-adr-templates](2026-11-17-adr-templates.md) | ADR formats: Nygard vs MADR | ADR-0001 |
| [2026-11-17-kill-switch-mechanisms](2026-11-17-kill-switch-mechanisms.md) | VPN kill switch mechanisms and semantics | ADR-0005, ADR-0009 |
| [2026-11-17-provider-clis](2026-11-17-provider-clis.md) | NordVPN, Proton and Surfshark CLIs and manual configs | ADR-0006 |
| [2026-11-17-multilingual-seo](2026-11-17-multilingual-seo.md) | hreflang, localized URLs, no auto-redirect | ADR-0012 |
| [2026-11-17-go-embedded-spa](2026-11-17-go-embedded-spa.md) | SPAs embedded in Go binaries | ADR-0011 |
| [2026-11-17-openvpn-non-admin](2026-11-17-openvpn-non-admin.md) | OpenVPN interactive service, non-admin options | ADR-0005, ADR-0007 |
| [2026-11-17-windows-network-configuration-operators](2026-11-17-windows-network-configuration-operators.md) | What that Windows group can and cannot do | ADR-0005 |
| [2026-11-17-public-ip-services](2026-11-17-public-ip-services.md) | ipify and alternatives | ADR-0010 |
| [2026-11-17-cloudflare-pages](2026-11-17-cloudflare-pages.md) | Cloudflare Pages build config and headers | ADR-0012, ADR-0015 |
| [2026-11-17-security-txt](2026-11-17-security-txt.md) | RFC 9116 `security.txt` | ADR-0016 |
| [2026-11-17-agent-instructions](2026-11-17-agent-instructions.md) | `AGENTS.md` and Claude Code imports | ADR-0002 |
| [2026-11-17-vpn-failover](2026-11-17-vpn-failover.md) | How VPN clients define failover (OpenVPN, NetworkManager, WireGuard, Cisco) | ADR-0018 |
| [2026-11-17-forticlient-versions](2026-11-17-forticlient-versions.md) | FortiClient versions, command lines, version detection, test installers | ADR-0020 |
| [2026-11-17-go-release-support](2026-11-17-go-release-support.md) | Go release support policy and the go.mod `toolchain` line | ADR-0019 |

## Template

Name: `YYYY-MM-DD-topic-with-dashes.md` (date the research was done). Copy this:

```markdown
<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# Research: <topic>

Version 1.0.0 · YYYY-MM-DD

## Question

What we needed to know, and for which decision.

## Sources

| # | Source | URL | Accessed |
|---|---|---|---|
| 1 | Publisher — page title | https://… | YYYY-MM-DD |

## Findings

- Finding, with a short quote where wording matters [1].
- Anything not confirmed by a primary source: **no verificado** (not verified), and why.

## Confidence

High / Medium / Low, and what it rests on (primary docs, code, third-party, reasoning).

## Open questions

- What is still unknown and how to find out.

## Used by

- ADR-NNNN
```

Rules: prefer primary sources (vendor docs, RFCs, standards, source code); record the access date;
quotes are short; a later re-check gets a new dated note or a version bump, never a silent edit.
