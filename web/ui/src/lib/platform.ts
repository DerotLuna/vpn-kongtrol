// Which OS shows the permission prompt? · v0.1.0 · 2026-11-18
//
// The panel is opened on the same computer the daemon runs on (127.0.0.1),
// so the browser's platform names the OS that will ask for permission.
// Used only to word instructions; nothing depends on it.
export type Platform = 'windows' | 'mac' | 'linux' | 'other'

interface NavigatorUAData {
  platform?: string
}

export function detectPlatform(nav: Navigator = navigator): Platform {
  const hint = (nav as Navigator & { userAgentData?: NavigatorUAData }).userAgentData?.platform
  const raw = (hint || nav.platform || nav.userAgent || '').toLowerCase()
  if (raw.includes('win')) return 'windows'
  if (raw.includes('mac')) return 'mac'
  if (raw.includes('linux') || raw.includes('x11')) return 'linux'
  return 'other'
}
