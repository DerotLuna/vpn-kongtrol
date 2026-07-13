# Kongtrol Landing (`site/`)

Public landing site, separate from the embedded dashboard (`web/dashboard`).

## Local development

```bash
cd site
pnpm install
pnpm run dev
```

## Build

```bash
cd site
pnpm run build
```

## Downloads

- The site serves local binaries from `site/_binaries/`.
- `GET /api/download` validates a temporary token and serves the file from that folder.
- Expected files:
  - `kongtrol-windows-amd64.exe`
  - `kongtrol-darwin-amd64`
  - `kongtrol-darwin-arm64`
  - `kongtrol-linux-amd64`
  - `kongtrol-linux-arm64`
  - `checksums.txt`

To refresh them from this repo:

```bash
make build-all-cli
make site-sync-binaries
```

## Deploy (Vercel)

```bash
cd site
vercel --prod
```
