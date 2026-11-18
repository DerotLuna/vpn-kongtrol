// Kongtrol UI — profile file import: pure helpers · v1.0.0 · 2026-11-18
//
// Same limits as internal/profileimport, so a file the daemon would refuse
// for its name or size never travels (tested in importModel.test.ts).

/** The helper's limit (privops.MaxConfigSize, 512 KiB); the API refuses more. */
export const MAX_IMPORT_BYTES = 512 * 1024

export const IMPORT_EXT = { wireguard: '.conf', openvpn: '.ovpn' } as const

/**
 * The stored file name (profileimport nameRE: letters, digits, dot, dash,
 * underscore, starting with a letter or digit, ≤ 64): the profile name when
 * typed, else the file's own name; accents dropped, anything else → "-".
 */
export function importName(preferred: string, fileName: string): string {
  const stem = fileName.replace(/\.[^.]*$/, '')
  for (const raw of [preferred, stem]) {
    const name = raw
      .trim()
      .normalize('NFD')
      .replace(/[̀-ͯ]/g, '')
      .replace(/[^A-Za-z0-9._-]+/g, '-')
      .replace(/^[^A-Za-z0-9]+/, '')
      .replace(/-+$/, '')
      .slice(0, 64)
    if (name) return name
  }
  return 'vpn'
}

/** base64 of the file, in chunks (one huge spread into fromCharCode overflows the stack). */
export function toBase64(bytes: Uint8Array): string {
  let bin = ''
  for (let i = 0; i < bytes.length; i += 0x8000) bin += String.fromCharCode(...bytes.subarray(i, i + 0x8000))
  return btoa(bin)
}
