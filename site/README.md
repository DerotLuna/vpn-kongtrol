# Kongtrol Landing (`site/`)

Landing público independiente del dashboard embebido (`web/dashboard`).

## Desarrollo local

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

## Descargas

- El sitio usa binarios locales en `site/_binaries/`.
- `GET /api/download` valida token temporal y sirve el archivo desde esa carpeta.
- Archivos esperados:
  - `kongtrol-windows-amd64.exe`
  - `kongtrol-darwin-amd64`
  - `kongtrol-darwin-arm64`
  - `kongtrol-linux-amd64`
  - `kongtrol-linux-arm64`
  - `checksums.txt`

Para refrescarlos desde este repo:

```bash
make build-all-cli
make site-sync-binaries
```

## Deploy (Vercel)

```bash
cd site
vercel --prod
```
