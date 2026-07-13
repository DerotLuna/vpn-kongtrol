# Download Security Notes (current + next)

## Estado actual (vigente)

- Sitio en Vercel.
- Descargas protegidas por `DOWNLOAD_KEY` + token temporal.
- API `site/api/download.ts` sirve binarios desde `site/_binaries/`.
- Acceso recomendado del sitio: Password/IP allowlist/Access (Vercel o Cloudflare).

## Riesgos conocidos (aceptables para interno)

- Si alguien comparte URL + clave, puede descargar.
- No hay identidad por usuario en la descarga (es clave compartida).
- Binarios no firmados criptográficamente.

## Endurecimiento mínimo recomendado

1. Rotar `DOWNLOAD_KEY` periódicamente.
2. Mantener token corto (actualmente 1h).
3. Publicar y verificar `checksums.txt`.
4. Limitar acceso al landing (IP allowlist o Access).

## Próxima evolución sugerida

- Mover binarios a storage privado (S3/R2/B2) con URLs firmadas de 5–10 min.
- Mantener Vercel solo como gate/API.
- Opcional: firma de artefactos (cosign o GPG).
