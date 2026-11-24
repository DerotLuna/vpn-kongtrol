<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# ADR-0106: Live lists use "load more" + filters and hold new entries, never pages

Version 1.0.0 · 2026-11-24

Status: Accepted · Date: 2026-11-24 · Decision-makers: Ruben Luna (owner), implemented by agent

Refines [ADR-0054](0054-recent-destinations-memory-only.md). Research:
[2026-11-24-live-list-load-more](../research/2026-11-24-live-list-load-more.md).

## Context and Problem Statement

Tráfico › Recientes rendered every recent destination (up to 500), refreshed every 15 s,
newest first. The list was very long, and each refresh moved rows: an entry seen again jumps to
the top, so what the user was reading shifted. Pages would shift the same way (page 2 changes on
every tick).

## Decision Drivers

- Never move what the user is reading; let them choose when to see new entries.
- Find one destination fast (site, IP or program; which way it went out).
- No GUI-only feature (CLI parity); no API change when the client can do it (500 entries max).
- Accessible: keyboard and screen reader.

## Considered Options

1. Show everything (as before).
2. Pagination.
3. Load more + filters + held live updates.

## Decision Outcome

Chosen: option 3. NN/g presents "load more" as the middle ground between pages and infinite
scroll; pages give landmarks that a live list cannot keep (research note).

- `web/ui/src/components/traffic/recentView.ts` + `RecentCard.tsx`: 10 rows, then
  "Ver 20 más (quedan N)" (hidden when none remain; its row has `overflow-anchor: none` so
  appended rows never scroll the page). Search "Buscar sitio, IP o programa" (host, IP, program,
  case-insensitive) and "Salió por" (Todas / Conexión normal / each VPN in the list, kit
  `Select`) filter the whole list client-side; a new filter starts again at 10 rows; no match →
  empty state with "Quitar filtros".
- The rows on screen are a snapshot (`useStableList`): each row's data (Visto) updates in place,
  nothing is added, removed or reordered until "N nuevas · Mostrar" (a `role="status"` polite
  region, always mounted) merges the current list, new entries on top; focus moves to the list
  without scrolling.
- Long hosts and program names are clipped with the full text in `title`; the table still turns
  into cards by container query.
- The panel asks for all entries (`?limit=500`, `monitor.RecentMax`); the API is unchanged.
- CLI parity: `kongtrol traffic recent --search <text> --via <profile|direct>` with the same
  matching (`filterRecent`), then `--limit`.

## Consequences

- Good: a short list that stays still while read; a destination is one search away.
- Good: the same rules in panel and CLI.
- Bad: rows that left the daemon's list (forgotten after an hour) stay on screen until the next
  "Mostrar".
- Neutral: the same pattern applies to any future live list in the panel.
