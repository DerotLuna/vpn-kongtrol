import { generateToken, readBody } from './_utils'

export default async function handler(req: any, res: any) {
  if (req.method !== 'POST') {
    return res.status(405).json({ ok: false, error: 'Method not allowed' })
  }
  try {
    const body = await readBody(req)
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
