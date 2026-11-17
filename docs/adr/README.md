<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# Architecture Decision Records

Version 1.0.0 · 2026-11-17

Every product or architecture decision in Kongtrol is recorded here as an ADR, using the
[MADR 4.0.0](https://adr.github.io/madr/) template (why MADR: [ADR-0001](0001-record-decisions-with-madr.md),
research: [ADR templates](../research/2026-11-17-adr-templates.md)). Investigations that back a
decision live in [`docs/research/`](../research/README.md).

## Index

| ADR | Title | Status | Date |
|---|---|---|---|
| [0001](0001-record-decisions-with-madr.md) | Record decisions as MADR ADRs, research as dated notes | Accepted | 2026-11-17 |
| [0002](0002-agents-md-cross-vendor-instructions.md) | `AGENTS.md` as the cross-vendor agent instructions | Accepted | 2026-11-17 |
| [0003](0003-product-model-gui-first-cli-complete.md) | One product, GUI-first, CLI-complete | Accepted | 2026-11-17 |
| [0004](0004-detached-daemon-lifecycle.md) | Detached daemon: views never stop tunnels, only `down` | Accepted | 2026-11-17 |
| [0005](0005-privileged-helper-service.md) | Privileged helper service: closed op list, IPC auth, lease, fail-open | Accepted | 2026-11-17 |
| [0006](0006-provider-adapters-attach-mode.md) | Provider adapters with attach mode; direct WireGuard/OpenVPN way | Accepted | 2026-11-17 |
| [0007](0007-openvpn-pushed-routes-binary-cd.md) | OpenVPN: accept pushed routes, binary resolution, `--cd` | Accepted | 2026-11-17 |
| [0008](0008-policy-routing-on-tunnel-down.md) | Policy routing for all adapters, `on_tunnel_down`, notices | Accepted | 2026-11-17 |
| [0009](0009-kill-switch-split-tunneling.md) | Kill switch for split tunneling: block-only, per destination | Accepted | 2026-11-17 |
| [0010](0010-public-ip-check-opt-in.md) | Public IP check is opt-in (Local only) | Accepted | 2026-11-17 |
| [0011](0011-dashboard-vite-react-ts.md) | Dashboard migration to Vite + React + TS by phases, at `/app/` | Accepted | 2026-11-17 |
| [0012](0012-landing-cloudflare-pages-multilingual.md) | Landing on Cloudflare Pages, `kongtrol.app`, ES + EN | Accepted | 2026-11-17 |
| [0013](0013-brand-single-source.md) | Brand assets: single source in `assets/brand` | Accepted | 2026-11-17 |
| [0014](0014-i18n-language-resolution.md) | i18n: saved choice, then OS; dashboards follow the browser | Accepted | 2026-11-17 |
| [0015](0015-ci-scoped-builds-generated-docs.md) | CI: build only what changed, generated docs, non-blocking doc checks | Accepted | 2026-11-17 |
| [0016](0016-legal-contact-model.md) | Legal and contact: individual open-source project, Apache-2.0 | Accepted | 2026-11-17 |
| [0017](0017-dependabot-and-github-security.md) | Dependabot grouped weekly; GitHub security features | Accepted | 2026-11-17 |

## Statuses

From MADR 4.0.0 and Nygard (see [research](../research/2026-11-17-adr-templates.md)):

| Status | Meaning |
|---|---|
| Proposed | Under discussion, not yet in force. |
| Accepted | In force. A suffix such as "(implementation in progress)" says the code does not fully match yet. |
| Rejected | Considered and not adopted; kept for the record. |
| Deprecated | No longer applies; nothing replaces it. |
| Superseded by ADR-NNNN | Replaced; the new ADR links back with "Supersedes ADR-NNNN". |

ADRs are immutable once accepted: to change a decision, write a new ADR that supersedes the old
one and only edit the old one's status line. Numbers are never reused (Nygard).

## How to add an ADR

1. Copy the structure of an existing ADR (MADR 4.0.0 sections: Context and Problem Statement,
   Decision Drivers, Considered Options, Decision Outcome, Consequences, Confirmation, More
   Information). The full template: https://github.com/adr/madr/blob/4.0.0/template/adr-template.md
2. Name it `NNNN-title-with-dashes.md` with the next free number.
3. Start with the logo line and `Version X.Y.Z · YYYY-MM-DD`, then a
   `Status · Date · Decision-makers` line.
4. Link the code that implements it and the research note(s) that back it. Every external fact
   needs a URL; if it could not be verified, write **no verificado** (not verified).
5. Add a row to the index above in the same change.
