<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Research notes

Version 1.12.0 · 2026-11-19

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
| [2026-11-17-cloudflare-workers-static-assets](2026-11-17-cloudflare-workers-static-assets.md) | Workers static assets, Workers Builds, redirects | ADR-0023 |
| [2026-11-17-security-txt](2026-11-17-security-txt.md) | RFC 9116 `security.txt` | ADR-0016 |
| [2026-11-17-agent-instructions](2026-11-17-agent-instructions.md) | `AGENTS.md` and Claude Code imports | ADR-0002 |
| [2026-11-17-vpn-failover](2026-11-17-vpn-failover.md) | How VPN clients define failover (OpenVPN, NetworkManager, WireGuard, Cisco) | ADR-0018 |
| [2026-11-17-forticlient-versions](2026-11-17-forticlient-versions.md) | FortiClient versions, command lines, version detection, test installers | ADR-0020 |
| [2026-11-17-go-release-support](2026-11-17-go-release-support.md) | Go release support policy and the go.mod `toolchain` line | ADR-0019 |
| [2026-11-17-provider-logo-guidelines](2026-11-17-provider-logo-guidelines.md) | Brand guidelines of the ten VPN brands for logos on the landing | ADR-0023 |
| [2026-11-17-landing-ux-a11y-perf](2026-11-17-landing-ux-a11y-perf.md) | Motion without CLS, WCAG 2.2 AA gaps, fonts, JSON-LD; Lighthouse before/after | ADR-0024 |
| [2026-11-18-desktop-release-channels](2026-11-18-desktop-release-channels.md) | Tray build per OS, GoReleaser OSS vs Pro, Inno Setup, SignPath, macOS quarantine, npm layout, package managers | ADR-0025 |
| [2026-11-18-react-hydration](2026-11-18-react-hydration.md) | hydrateRoot, useSyncExternalStore server snapshots, jsdom hydration check; Lighthouse before/after | ADR-0026 |
| [2026-11-18-landing-load-performance](2026-11-18-landing-load-performance.md) | Cloudflare `_headers` caching, render-blocking CSS options, forced reflow, compositor-only animations, React 18 selective hydration; Lighthouse before/after | ADR-0031 |
| [2026-11-18-tray-flyout](2026-11-18-tray-flyout.md) | Wails v3, webview_go, go-webview2, fyne.io/systray taps, Shell_NotifyIconGetRect, WebView2 Runtime availability, SNI click position | ADR-0036 |
| [2026-11-18-wireguard-liveness](2026-11-18-wireguard-liveness.md) | wg(8) dump, WireGuard timers, wireguard-windows access rights, Tailscale/NetBird handshake use; issue #33 root cause | ADR-0030 |
| [2026-11-18-native-tray-menu](2026-11-18-native-tray-menu.md) | fyne.io/systray per-OS features, Windows taskbar theme, macOS template icons, SVG mark rasterizing, CLI copies in PATH | ADR-0037 |
| [2026-11-18-landing-download-ctas](2026-11-18-landing-download-ctas.md) | HIG/Material 3 button hierarchy, phone/tablet detection, Web Share + clipboard, Apple Silicon vs Intel check | ADR-0027 |
| [2026-11-18-landing-hig-material3-review](2026-11-18-landing-hig-material3-review.md) | Apple HIG (buttons, typography, accessibility, feedback, motion, layout) and M3 tokens applied to the landing; Lighthouse before/after | ADR-0028 |
| [2026-11-18-cli-command-structure](2026-11-18-cli-command-structure.md) | clig.dev interactivity/deprecation/naming, cobra deprecation text, gh/tailscale/systemctl naming | ADR-0033 |
| [2026-11-18-install-script-delivery](2026-11-18-install-script-delivery.md) | GitHub latest-download alias, GoReleaser release/checksum extra_files, Cloudflare Workers `_redirects` | ADR-0035 |
| [2026-11-18-local-web-ui-auth](2026-11-18-local-web-ui-auth.md) | OWASP session/CSRF guidance, ASVS V3, Jupyter tokens, Syncthing Host check; token-on-page-load root cause | ADR-0039 |
| [2026-11-18-security-hardening-review](2026-11-18-security-hardening-review.md) | wg dump + X25519 key binding, TrustedInstaller SID, Program Files ACL, cmd.exe quoting, GitHub Actions hardening, Renovate digest pinning, OWASP Top 10 / Desktop Top 10 | ADR-0040 |
| [2026-11-18-dashboard-ui-patterns](2026-11-18-dashboard-ui-patterns.md) | Apple HIG, Material 3, Fluent 2, WCAG 2.2 AA, Nielsen, live-data patterns (SWR, optimistic, backoff+jitter), router / data library choice for `web/ui` | ADR-0038 |
| [2026-11-18-dashboard-api-gaps](2026-11-18-dashboard-api-gaps.md) | Size-based log rotation (logrotate, lumberjack), OpenVPN/WireGuard directives that run code, HTTP content coding (RFC 9110), stable sort | ADR-0042 |
| [2026-11-18-panel-at-root](2026-11-18-panel-at-root.md) | Fragments across redirects (RFC 9110), 308, CSP without `unsafe-inline` (MDN, CSP3 hash-source), backslash open redirect (WHATWG URL) | ADR-0043 |
| [2026-11-18-unsaved-changes-guard](2026-11-18-unsaved-changes-guard.md) | Leaving a form with unsaved edits: `beforeunload`, React Router `useBlocker` (data mode only), prompt wording | ADR-0041 |
| [2026-11-19-explorer-double-click](2026-11-19-explorer-double-click.md) | Console program opened from Explorer: mousetrap `StartedByExplorer`, cobra v1.8.0 mousetrap hook (English-only wait line) | ADR-0045 |
| [2026-11-19-installer-upgrade-detection](2026-11-19-installer-upgrade-detection.md) | Detecting an existing install: Inno Setup uninstall key, `UsePrevious*`, `ShouldSkipPage`, SemVer precedence | ADR-0046 |
| [2026-11-19-product-name-and-logo-sizes](2026-11-19-product-name-and-logo-sizes.md) | Dropping "VPN" from the name (site name, alternateName, what must not be renamed), SVG favicon, logo sizes | ADR-0047 |

## Template

Name: `YYYY-MM-DD-topic-with-dashes.md` (date the research was done). Copy this:

```markdown
<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

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
