import { createHmac } from 'crypto'

const WINDOW_MS = 60 * 60 * 1000 // 1 hora

function tokenForWindow(key: string, win: number): string {
  return createHmac('sha256', key).update(String(win)).digest('hex')
}

export function generateToken(key: string): string {
  return tokenForWindow(key, Math.floor(Date.now() / WINDOW_MS))
}

export function validateToken(token: string, key: string): boolean {
  const win = Math.floor(Date.now() / WINDOW_MS)
  return token === tokenForWindow(key, win) || token === tokenForWindow(key, win - 1)
}

// Lee el body de la request — maneja tanto el caso en que Vercel lo parsea
// automáticamente como el caso en que llega como stream.
export async function readBody(req: any): Promise<Record<string, string>> {
  if (typeof req.body === 'string') {
    try {
      return JSON.parse(req.body)
    } catch {
      return {}
    }
  }
  if (req.body && typeof req.body === 'object' && !Buffer.isBuffer(req.body)) {
    return req.body
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
