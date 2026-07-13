import { readFileSync } from 'fs'
import { join } from 'path'
import { validateToken } from './_utils'

const ALLOWED: Record<string, string> = {
  'kongtrol-darwin-amd64':      'application/octet-stream',
  'kongtrol-darwin-arm64':      'application/octet-stream',
  'kongtrol-linux-amd64':       'application/octet-stream',
  'kongtrol-linux-arm64':       'application/octet-stream',
  'kongtrol-windows-amd64.exe': 'application/octet-stream',
  'checksums.txt':              'text/plain; charset=utf-8',
}

export default function handler(req: any, res: any) {
  if (req.method !== 'GET') {
    return res.status(405).json({ ok: false, error: 'Method not allowed' })
  }

  try {
    const { file, token } = req.query as Record<string, string>

    if (!process.env.DOWNLOAD_KEY || !token || !validateToken(token, process.env.DOWNLOAD_KEY)) {
      return res.status(401).json({ ok: false, error: 'Token inválido o expirado' })
    }

    const mime = file && ALLOWED[file]
    if (!mime) {
      return res.status(404).json({ ok: false, error: 'Archivo no encontrado' })
    }

    const data = readFileSync(join(process.cwd(), '_binaries', file))
    res.setHeader('Content-Type', mime)
    res.setHeader('Content-Disposition', `attachment; filename="${file}"`)
    res.setHeader('Content-Length', data.length)
    res.status(200).send(data)
  } catch (err) {
    console.error('[download] error:', err)
    res.status(500).json({ ok: false, error: 'Error interno' })
  }
}
