import { createHmac } from 'crypto'

function generateToken(key: string): string {
  const win = Math.floor(Date.now() / 86400000)
  return createHmac('sha256', key).update(String(win)).digest('hex')
}

async function parseBody(req: any): Promise<Record<string, unknown>> {
  if (req.body !== undefined && req.body !== null) {
    if (typeof req.body === 'string') {
      try { return JSON.parse(req.body) } catch { return {} }
    }
    if (Buffer.isBuffer(req.body) || req.body instanceof Uint8Array) {
      try { return JSON.parse(Buffer.from(req.body).toString()) } catch { return {} }
    }
    if (typeof req.body === 'object' && !Buffer.isBuffer(req.body)) {
      return req.body as Record<string, unknown>
    }
  }
  if (!req || typeof req.on !== 'function') {
    return {}
  }
  return new Promise(resolve => {
    const chunks: Buffer[] = []
    req.on('data', (c: Buffer) => chunks.push(c))
    req.on('end', () => {
      try { resolve(JSON.parse(Buffer.concat(chunks).toString())) }
      catch { resolve({}) }
    })
    req.on('error', () => resolve({}))
  })
}

export default async function handler(req: any, res: any) {
  if (req.method !== 'POST') {
    return res.status(405).json({ ok: false, error: 'Method not allowed' })
  }

  try {
    const body = await parseBody(req)
    const key = typeof body.key === 'string' ? body.key : undefined

    if (!process.env.DOWNLOAD_KEY || key !== process.env.DOWNLOAD_KEY) {
      return res.status(401).json({ ok: false, error: 'Clave incorrecta' })
    }

    const token = generateToken(process.env.DOWNLOAD_KEY)
    return res.status(200).json({ ok: true, token })
  } catch (err) {
    console.error('[auth] error:', err)
    return res.status(500).json({ ok: false, error: 'Error interno' })
  }
}
