export type OS = 'windows' | 'macos' | 'linux'

export function detectOS(): OS {
  if (typeof navigator === 'undefined') return 'linux'
  const ua = navigator.userAgent.toLowerCase()
  if (ua.includes('win')) return 'windows'
  if (ua.includes('mac')) return 'macos'
  return 'linux'
}

export const OS_LABELS: Record<OS, string> = {
  windows: 'Windows',
  macos:   'macOS',
  linux:   'Linux',
}

export const OS_ICON: Record<OS, string> = {
  windows: '⊞',
  macos:   '',
  linux:   '🐧',
}
