# ADR-0119: Dashboard search and visual states

Status: Accepted · Date: 2026-11-25 · Decision-maker: owner

Refines ADR-0051 for the dashboard header only and ADR-0114 for the Inicio
latency cell. Research: [dashboard search and cards](../research/2026-11-25-dashboard-search-and-cards.md).

## Decision

- The panel header uses the requested 48 px line-art mark from the existing SVG master.
  Other logo surfaces keep their existing rules.
- Search is local to the panel and offers sections, add actions, VPN profiles,
  rules (including their destinations) and groups. Ctrl/Cmd+K opens it; Escape
  closes it. React Aria provides list navigation. Result navigation preserves
  the existing unsaved-changes guard.
- Dashboard status pills use a word and color without a leading dot. Working
  protections are green, enabled protections waiting for a tunnel are neutral.
  Tray and CLI indicators are unchanged.
- Traffic tunnel route details start closed. Their cards reflow independently.
- Inicio stops saying it is measuring when a connected tunnel has no latency
  after 45 seconds; a single reading is shown without a fictitious trend.

## Confirmation and limitations

Tests cover search, route disclosure and latency states. The 48 px mark,
card reflow and keyboard behavior were checked in a local browser with
simulated profiles. OpenVPN latency has not been verified on a real tunnel.
