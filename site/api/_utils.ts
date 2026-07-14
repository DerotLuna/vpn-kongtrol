import { createHmac } from 'crypto'

// Token HMAC-SHA256 válido por ventana de 24h (día UTC).
// Aceptamos también la ventana anterior para evitar cortes en el cambio de día.
function tokenForWindow(key: string, win: number): string {
  return createHmac('sha256', key).update(String(win)).digest('hex')
}

export function generateToken(key: string): string {
  return tokenForWindow(key, Math.floor(Date.now() / 86400000))
}

export function validateToken(token: string, key: string): boolean {
  const win = Math.floor(Date.now() / 86400000)
  return token === tokenForWindow(key, win) || token === tokenForWindow(key, win - 1)
}

// Lee el body de la request — maneja tanto el caso en que Vercel lo parsea
// automáticamente como el caso en que llega como stream.
export async function readBody(req: any): Promise<Record<string, unknown>> {
  if (req.body !== undefined && req.body !== null) {
    if (typeof req.body === 'string') {
      try {
        return JSON.parse(req.body)
      } catch {
        return {}
      }
    }
    if (Buffer.isBuffer(req.body) || req.body instanceof Uint8Array) {
      try {
        return JSON.parse(Buffer.from(req.body).toString())
      } catch {
        return {}
      }
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
      try {
        resolve(JSON.parse(Buffer.concat(chunks).toString()))
      } catch {
        resolve({})
      }
    })
    req.on('error', () => resolve({}))
  })
}
