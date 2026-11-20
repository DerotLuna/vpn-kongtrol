<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0071: Panel navigation polish, Tráfico in order of importance, tmux bar without other pages, and the release tag pinned

Version 1.0.0 · 2026-11-20

Status: Accepted · Date: 2026-11-20 · Decision-makers: project maintainer (owner)

## Context and Problem Statement

Owner feedback while testing v0.7.0-rc.2, 2026-11-20:

- The panel's tab rows show a scrollbar although the tabs fit (issue #44).
- "Contraer menú" jumps without animation, and its icon does not say where the menu goes (#44).
- Tráfico lists the lookup tool and recent destinations before what goes through each VPN.
- The landing's tmux bar ends with `8:guía`, a link to another page, mixed with the page's sections.
- The beta notice on the landing's download page is narrower than the cards around it.
- The stable tag v0.7.0, on the same commit as v0.7.0-rc.2, failed to publish: GoReleaser picked
  the rc tag and tried to upload into the rc's release (422 `already_exists`, run 44754426073).

## Decision Outcome

- **Tabs** (`web/ui/src/theme/kit.css`): the list's hairline is an inset shadow and the selected
  mark sits at `bottom: 0`, inside the list's box; `overflow-y: hidden`. A list that fits never
  scrolls; too many tabs scroll sideways.
- **Side menu** (`theme/shell.css`, `components/layout/AppShell.tsx`, `components/ui/icons.tsx`):
  the grid column animates (`grid-template-columns`, `--dur-med`, `--ease-standard`; off with
  reduced motion), icons keep the same x in both states (11px item padding), labels fade and the
  column clips them (`overflow: clip`, which keeps the menu sticky). The toggle icon is an arrow
  against the menu's edge pointing left; collapsed, it turns to point right. Same names and tooltips.
- **Tráfico** (`pages/TrafficPage.tsx`): "Qué pasa por cada VPN" first (live state), then "¿Por
  dónde sale?" (a tool), "Recientes" (history) and "Rutas del sistema" (technical, folded).
- **Landing tmux bar** (`components/ui/TmuxBar.tsx`): it lists only the sections of the page it is
  on; other pages are reached from the header (`--guia`). The `link` window is removed.
- **Beta notice** (`download.css`): full width, like the cards around it.
- **Release** (`.github/workflows/release.yml`): GoReleaser gets `GORELEASER_CURRENT_TAG` = the tag
  that started the run, and, for a stable tag, `GORELEASER_PREVIOUS_TAG` = the previous stable tag
  (its changelog covers the whole version, not just the last rc). Pre-release tags keep
  GoReleaser's default. Documented for tags that share a commit:
  https://goreleaser.com/cookbooks/set-a-custom-git-tag (accessed 2026-11-20). The failed v0.7.0
  tag, which never had a release, is recreated on the commit carrying this fix.

### Consequences

- Good: the panel reads calmer and the release pipeline works for "stable on the rc's commit",
  the normal way we promote an rc.
- Neutral: the column animation is a layout animation (not transform/opacity), the standard way
  navigation drawers resize; it lasts 250 ms and is off with reduced motion.

## More Information

- Issue: https://github.com/DerotLuna/vpn-kongtrol/issues/44
- Refines: [ADR-0068](0068-dashboard-kit-on-react-aria-components.md) (collapsible side menu),
  [ADR-0070](0070-landing-instrument-terminal-style-contour-edges-and-short-gold-band.md) (landing),
  [ADR-0025](0025-desktop-release-channels-first-run.md) (release pipeline).
- Docs: `DESIGN.md` (tmux bar), landing guide (Tráfico).
