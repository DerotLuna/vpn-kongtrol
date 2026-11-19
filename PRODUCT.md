<img src="assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# Product

Versión 1.1.0 · 2026-11-19

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

Two audiences with equal weight. The project started for technical users; it now serves both.

- **Non-technical people** who use more than one VPN on the same computer — typically a work client (FortiClient, Cisco AnyConnect, GlobalProtect) and/or a consumer VPN (NordVPN, Proton VPN, Surfshark) — and want them to work together without fighting over routes and DNS, or to stay protected when one drops. They install and use the app from the GUI (installer, tray, dashboard).
- **Developers and technical professionals** (devs, SRE, consultants) who reach several networks at once (client networks, WireGuard, OpenVPN, Tailscale, Cloudflare WARP) and want per-destination routing, automation and a CLI with full parity.

## Product Purpose

Kongtrol puts all your VPNs in one place. You connect the way you always do (in the provider's official app, or directly with a WireGuard/OpenVPN file); Kongtrol detects the tunnel and takes care of the rest: status, automatic reconnection, DNS leak protection, kill switch and routing rules per destination. Success: several VPNs running at once on one machine, each site going through the right one, and the user knowing at a glance what is up, what dropped and what to do.

## Positioning

- Works **with** the VPN apps people already use (attach mode) instead of replacing them; 10 supported VPNs.
- **Local only:** never sends or processes user data off the machine; any outbound call is opt-in.
- **Free and open source** (Apache-2.0), no account.
- One product, GUI-first and CLI-complete: no feature exists only in the terminal or only in the GUI.

## Operating Context

- Surfaces covered by this record: the public site `landing/` and the web dashboard `web/ui`. The product itself is a desktop app (tray, dashboard, CLI) for Windows, macOS and Linux.
- Runs as a background daemon per user; tray, dashboard and CLI are clients of its local API. Closing a window never disconnects.
- Network changes go through a privileged helper installed once (one admin prompt), so daily use needs no admin terminal.
- Distribution: GitHub Releases, installers and package managers (winget, Scoop, Homebrew tap, .deb/.rpm, AUR, npm/pnpm); macOS binaries are not notarized (no paid Apple account).
- Visitors reach the site from search and shared links, on desktop (to download) and on phones (to read and send the link to their computer).

## Capabilities and Constraints

- Supported VPNs and per-OS support differ; the site states only what works today (see `docs/` and the landing VPN matrix). The site never advertises a feature before it ships.
- Languages: Spanish and English with equal weight and full parity; market is global, no priority region.
- Product status: **public beta**.
- **Name (decided 2026-11-19, ADR-0047):** the product is just **Kongtrol**. "VPN" lives only in the tagline ("Todas tus VPN bajo Kongtrol" / "All your VPNs under Kongtrol", a play on "control", ADR-0048); "VPN Kongtrol" is only a former name (search alias). Technical identifiers keep `vpn-kongtrol` (GitHub repo, Go module path, keychain service, package names).
- **Undecided — terminal/hacker voice:** the current site speaks in terminal idiom (tmux bar, `$ make install`, `man kongtrol`). The owner and developer friends like it; whether non-technical visitors understand or like it is unvalidated. Treat it as an open question to test, not a fixed rule nor something to remove silently.

## Brand Commitments

- The monkey logo (`assets/brand/`, single source of logos) and the name Kongtrol are fixed identity.
- Third-party VPN logos are not used; providers appear as our own monograms with a no-affiliation note (ADR-0023).
- User-facing copy: short lines, plain words, every action states what happens next and every error adds one "what to do" line.

## Evidence on Hand

- Real: the open-source repository, releases, the setup guide, legal pages (terms, privacy, security policy), Lighthouse/PageSpeed results.
- **None yet:** testimonials, customer logos, usage numbers, press, case studies. Never fabricate them.

## Product Principles

1. **Truth over promise:** say only what the product does today, per OS and per VPN.
2. **Guide, don't assume:** non-technical people must succeed without a terminal; developers must never be blocked by the GUI.
3. **Local and private by default:** nothing leaves the machine unless the user opts in.
4. **Works with what you have:** complement the user's existing VPN apps, never ask them to switch.
5. **Free and sustainable:** prefer free-for-open-source tooling; no paid dependencies in the user path.

## Accessibility & Inclusion

WCAG 2.2 AA on the public site and dashboards (contrast, keyboard, focus, target size ≥ 24 px, 44 px for primary touch targets, reduced motion respected); ES/EN parity for every string.
