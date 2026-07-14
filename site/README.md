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

Set this environment variable in Vercel Project Settings:

- `DOWNLOAD_KEY` (required by `/api/auth` and `/api/download`)

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
