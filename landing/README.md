# Kongtrol CLI Landing (`landing/`)

Public marketing/docs site for **Kongtrol CLI**, separate from the embedded dashboard (`web/dashboard`).

Two pages (Vite multi-page app):

- `/` — landing: the hero is an SSH session (ASCII motd, auto boot, then an
  interactive demo shell — try `help`, `status`, `map claude.ai`, `sudo rm -rf /`),
  policy switchboard, security, how-it-works, download, tmux-style status bar
- `/guia` — full setup guide (ES/EN, per-OS tabs)

## Prerequisites

- Node.js 20+
- pnpm 10+

## Local development

```bash
cd landing
pnpm install
pnpm run dev
```

## Build

```bash
cd landing
pnpm run build
```

## Quality check

```bash
cd landing
pnpm run check
```

## Project structure

```text
landing/
├── index.html        # landing entry
├── guia.html         # guide entry (/guia via cleanUrls)
└── src/
    ├── App.tsx       # landing page composition
    ├── GuideApp.tsx  # guide page composition
    ├── components/   # UI sections and reusable presentational blocks
    ├── hooks/        # UI/application state hooks (incl. useKonami easter egg)
    └── content/      # static guide/nav content shared by language variants
```

Two-page Vite build (`index.html` + `guia.html`); the hosting layer must rewrite
`/guia` to `guia.html` (clean URLs) since there's no server-side router.

## Downloads

Binaries are **no longer served by the landing site**. They are published as assets on
GitHub Releases:

- https://github.com/vpn-kongtrol/kongtrol/releases

Expected release assets: the 5 platform binaries + `checksums.txt`
(produced by `make build-all-cli`).

## Easter egg

Konami code (`↑↑↓↓←→←→BA`) toggles green-phosphor CRT mode.
