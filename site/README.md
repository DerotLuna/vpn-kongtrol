# Kongtrol CLI Site (`site/`)

Public marketing/docs site for **Kongtrol CLI**, separate from the embedded dashboard (`web/dashboard`).

## Prerequisites

- Node.js 20+
- pnpm 10+

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

## Quality check

```bash
cd site
pnpm run check
```

## Project structure

```text
site/
├── src/
│   ├── components/   # UI sections and reusable presentational blocks
│   ├── hooks/        # UI/application state hooks
│   └── content/      # static guide/nav content shared by language variants
├── api/              # Vercel serverless endpoints (/api/auth, /api/download)
└── _binaries/        # private release artifacts served by /api/download
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

Set this environment variable in Vercel Project Settings:

- `DOWNLOAD_KEY` (required by `/api/auth` and `/api/download`)

## Troubleshooting

- `401` on `/api/auth` or `/api/download`: verify `DOWNLOAD_KEY` in Vercel Project Settings.
- `404` on `/api/download`: verify file exists in `site/_binaries/` and is listed in `api/download.ts`.
- Missing artifacts in production: run `make build-all-cli && make site-sync-binaries` before deploy.

## API auth quick test

Expected behavior:
- `200` with `{ ok: true, token }` when the key is correct.
- `401` with `{ ok: false, error: "Clave incorrecta" }` when the key is wrong.

Linux/macOS:

```bash
curl -i -X POST https://vpn-kongtrol-site.vercel.app/api/auth \
  -H "Content-Type: application/json" \
  -d '{"key":"test"}'
```

Windows PowerShell (recommended):

```powershell
Invoke-RestMethod -Method Post -Uri "https://vpn-kongtrol-site.vercel.app/api/auth" -ContentType "application/json" -Body (@{ key = "test" } | ConvertTo-Json)
```

Windows with curl binary:

```powershell
curl.exe -i -X POST "https://vpn-kongtrol-site.vercel.app/api/auth" -H "Content-Type: application/json" -d "{\"key\":\"test\"}"
```
