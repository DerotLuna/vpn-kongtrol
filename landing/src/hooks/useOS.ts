import { useState, useSyncExternalStore } from 'react'

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

// The visitor's OS, or the tab they picked. The prerender cannot know the OS,
// so the server snapshot is what detectOS() returns without a navigator;
// hydrateRoot renders that first (identical to the HTML) and then re-renders
// with the detected OS. Reading navigator in a useState initializer instead
// would make the first client render differ from the HTML (a hydration
// mismatch). The user agent never changes, so there is nothing to subscribe to.
// https://react.dev/reference/react/useSyncExternalStore#adding-support-for-server-rendering
const noSubscribe = () => () => {}
const serverOS = (): OS => 'linux'

export function useOS(): [OS, (os: OS) => void] {
  const detected = useSyncExternalStore(noSubscribe, detectOS, serverOS)
  const [picked, setPicked] = useState<OS | null>(null)
  return [picked ?? detected, setPicked]
}
