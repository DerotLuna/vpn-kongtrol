<img src="../../assets/brand/logo-kong-64.png" alt="Kongtrol" width="48" align="left" />

# ADR-0061: Setup guide rewritten GUI-first, with "you'll see" steps, in the single-theme landing

Version 1.0.0 · 2026-11-19

Status: Accepted · Date: 2026-11-19 · Decision-makers: project maintainer (owner)

## Context and Problem Statement

The guide (`/guia`, `/en/guide`) still described a terminal-first product: unzip a binary and
edit PATH by hand on Windows, `curl | tar` on macOS/Linux, a wizard that asks for the language,
numbered prompts and sample outputs that no longer match. It also kept the old look (a hairline
over the content, a light quickstart band) after the landing moved to one theme (ADR-0060). The
owner asked to make it accurate, clear for non-technical users and consistent with the new design.

## Decision Drivers

- AGENTS.md: GUI-first and CLI-complete; every step says what to do and what to expect; one
  "what to do" line per problem; the landing only advertises what works; file names only from
  `landing/src/links.ts`.
- DESIGN.md "Sesión en la Torre" and ADR-0060: Bone page, `.tone-ink` only for emphasis, no new
  colours, 11 px floor, 44 px targets, no horizontal scroll at 390 px.

## Considered Options

1. Patch the stale lines and keep the structure.
2. **Rewrite the content GUI-first with a fixed step shape, keep the frame and the links.**

## Decision Outcome

Option 2 (owner request, 2026-11-19).

- **Order:** quickstart → how it works → before you start → install → your first VPN → connect →
  rules per VPN → groups → diagnostics → the panel → day to day (with the generated command
  reference, `#commands`) → if something fails → update → uninstall. Anchors other pages use
  (`#quickstart`, `#install`, `#commands`, `#update`, `#uninstall`) stay.
- **Step shape:** numbered steps with one instruction each and a "Verás →" / "You'll see →" line
  (Microsoft Writing Style Guide, research note). Problems: symptom → "Qué hacer →" line →
  optional command per OS (only Kongtrol's own firewall rules, never a full firewall reset).
- **Install per OS:** Windows = the installer (`WINDOWS_INSTALLER_FILE`) with SmartScreen,
  recommended options and the helper prompt; macOS/Linux = the one-line script (`INSTALL_CMD`),
  `.deb`/`.rpm` mentioned without file names; the PowerShell script only for the command.
- **Panel first, command second:** every task names the panel place, then the command.
  Removed: hand-written wizard transcripts, sample `doctor`/`status` output, the raw API endpoint
  list.
- **Visual:** the quickstart and the "how it works" diagram are the only `.tone-ink` blocks;
  everything else stays Bone. Guide-only CSS moved from `index.css` to
  `landing/src/components/sections/guide.css`; the shared layout stays because the legal pages use it.
- **SEO and tmux bar** follow the new titles and sections.

### Consequences

- Good: the guide matches the shipped installers, wizard and panel labels; shorter pages
  (desktop about 18 000 px instead of 27 000).
- Neutral: fewer sample outputs; the CLI reference (from `docs/cli.json`) remains the full list.
- Bad: `docs/SETUP.md` and the WARP adapter message still carry the stale commands listed in the
  research note (follow-up).

### Confirmation

Landing `lint`, `build`, `check:hydration`, `check:i18n`; Playwright screenshots at 1366×657
and 390×844, no horizontal scroll, no text below 11 px.

## More Information

- Code: `landing/src/content/guide.{es,en}.tsx`, `landing/src/content/guideSections.ts`,
  `landing/src/components/sections/{Guide.tsx,guide.css}`, `landing/src/pages/guia/App.tsx`,
  `landing/src/seo.ts`, `landing/src/index.css`.
- Research: [2026-11-19-landing-guide-accuracy](../research/2026-11-19-landing-guide-accuracy.md).
- Builds on: [ADR-0060](0060-landing-single-theme-bands-and-line-art-marks.md).
