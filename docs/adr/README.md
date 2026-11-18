<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# Architecture Decision Records

Version 1.5.0 · 2026-11-18

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
| [0012](0012-landing-cloudflare-pages-multilingual.md) | Landing on Cloudflare Pages, `kongtrol.app`, ES + EN | Accepted; hosting superseded by 0022 | 2026-11-17 |
| [0013](0013-brand-single-source.md) | Brand assets: single source in `assets/brand` | Accepted | 2026-11-17 |
| [0014](0014-i18n-language-resolution.md) | i18n: saved choice, then OS; dashboards follow the browser | Accepted | 2026-11-17 |
| [0015](0015-ci-scoped-builds-generated-docs.md) | CI: build only what changed, generated docs, non-blocking doc checks | Accepted | 2026-11-17 |
| [0016](0016-legal-contact-model.md) | Legal and contact: individual open-source project, Apache-2.0 | Accepted | 2026-11-17 |
| [0017](0017-dependabot-and-github-security.md) | Dependabot grouped weekly; GitHub security features | Superseded by 0021 | 2026-11-17 |
| [0018](0018-failover-only-to-declared-profiles.md) | Failover only to explicitly declared profiles | Accepted | 2026-11-17 |
| [0019](0019-track-supported-go-release.md) | Track a supported Go release (1.27) | Accepted | 2026-11-17 |
| [0020](0020-forticlient-multi-version-strategies.md) | FortiClient multi-version support: detect the version, pick a strategy, keep attach fallback | Accepted | 2026-11-17 |
| [0021](0021-renovate-for-dependency-updates.md) | Renovate for dependency updates; Dependabot only for alerts | Accepted | 2026-11-17 |
| [0022](0022-landing-on-workers-static-assets.md) | Landing on Cloudflare Workers static assets (Workers Builds) | Accepted | 2026-11-17 |
| [0023](0023-provider-logos-on-landing.md) | No third-party logos on the landing: our own monograms for every VPN | Accepted | 2026-11-18 |
| [0024](0024-landing-motion-a11y-performance.md) | Landing polish: CSS-only motion without layout shift, WCAG 2.2 AA, Core Web Vitals | Accepted (boot resume superseded by 0026) | 2026-11-18 |
| [0025](0025-desktop-release-channels-first-run.md) | Desktop release, package managers, signing, safe upgrades and first run without a terminal | Accepted | 2026-11-18 |
| [0026](0026-landing-hydrate-prerendered-html.md) | Landing hydrates its prerendered HTML (hydrateRoot) with a hydration check in CI | Accepted | 2026-11-18 |
| [0027](0027-landing-download-ctas.md) | Landing download buttons: one primary per view, recommended OS, phones send the page to a computer | Accepted | 2026-11-18 |
| [0028](0028-landing-hig-material3-review.md) | Landing review with Apple HIG and Material 3: 11px text floor, 44px targets, state layers, readable guide on phones | Accepted | 2026-11-18 |
| [0029](0029-landing-device-class-before-first-paint.md) | Landing marks phones before first paint (boot script `data-device`) so the hero never shifts | Accepted | 2026-11-18 |
| [0030](0030-wireguard-liveness-by-handshake.md) | WireGuard liveness by latest handshake (read-only helper op `wg.stats`); a silent server is a drop, unreadable counters say "unknown" | Accepted | 2026-11-18 |
| [0031](0031-landing-load-performance.md) | Landing load performance: inline CSS, hashed fonts, immutable caching, composited and on-screen-only demo loops, Suspense-split hydration | Accepted | 2026-11-18 |
| [0033](0033-cli-command-restructure.md) | Reestructura de comandos del CLI: nombres unificados (policy list/show, ip, routes --system, config default-group, reload [perfil]), alias viejos ocultos con aviso traducido, preguntas solo en terminal (--yes), init/up ofrecen el ayudante, config killswitch/dns-guard | Accepted | 2026-11-18 |

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
