<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# Research: SPAs embedded in Go binaries

Version 1.0.0 · 2026-11-17

## Question

How do mature Go projects ship a modern web UI inside a single binary, and can `go build` stay
usable without Node?

## Sources

| # | Source | URL | Accessed |
|---|---|---|---|
| 1 | tailscale/tailscale — `client/web` | https://github.com/tailscale/tailscale/tree/main/client/web | 2026-11-17 |
| 2 | pocketbase/pocketbase — `ui/embed.go` | https://github.com/pocketbase/pocketbase/blob/master/ui/embed.go | 2026-11-17 |
| 3 | Gitea docs — Installation from source | https://docs.gitea.com/installation/install-from-source | 2026-11-17 |

Quotes were captured with an automated fetch tool; re-open the URL before quoting them elsewhere.

## Findings

- Tailscale [1]: `client/web` has `package.json`, `vite.config.ts`, `tsconfig.json`,
  `tailwind.config.js`, `index.html`, `src/`; `package.json` lists `react ^18.2.0`,
  `typescript ^5.3.3`, `vite ^6.4.6`. `assets.go` has no `//go:embed`: it imports
  `github.com/tailscale/web-client-prebuilt`, and in dev mode proxies to the Vite server
  (`http://127.0.0.1:4000`). So: Vite + React + TS, assets prebuilt in a separate module.
- PocketBase [2]: `//go:embed all:dist` and `var distDir embed.FS` — built UI committed/embedded
  from `ui/dist`.
- Gitea [3]: "`bindata`: Build a single monolithic binary, with all assets included. Required for
  production build." — assets embedded only under a build tag.

## Conclusions (applied in `web/ui`)

- Vite + React + TS like Tailscale; dev server proxies `/api` to the daemon (`make ui-dev`).
- Embed `web/ui/dist` like PocketBase, but track only `dist/.gitkeep` so `go build` works without
  Node and serves a "not built" page; release builds always build the UI (Gitea-like split
  between dev and production builds, without a build tag).

## Confidence

High (source repositories and official docs).

## Open questions

- None.

## Used by

- [ADR-0011](../adr/0011-dashboard-vite-react-ts.md)
