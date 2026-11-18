import { useSyncExternalStore } from 'react'
import { detectOS, OS } from './useOS'

// Facts about the visitor's device, read only AFTER hydration. The prerender
// knows nothing about the visitor, so every server snapshot here is the
// "unknown" value; hydrateRoot renders that first (identical to the HTML) and
// then re-renders with the real value (same pattern as useOS, ADR-0026).
// https://react.dev/reference/react/useSyncExternalStore#adding-support-for-server-rendering
const noSubscribe = () => () => {}

type NavigatorUAData = { mobile?: boolean }

// Phone or tablet: Kongtrol is a desktop app, so these visitors get "send it
// to your computer" instead of download buttons. Heuristic (ADR-0027):
//   1. navigator.userAgentData.mobile — User-Agent Client Hints (Chromium).
//      True on phones; false on Android tablets, hence step 2.
//   2. UA contains Android, iPhone, iPad or iPod (Firefox/Safari have no
//      userAgentData; Android tablets omit "Mobile" but keep "Android").
//   3. iPadOS 13+ Safari sends a desktop Mac UA by default: a "Macintosh" UA
//      with more than one touch point is an iPad (no Mac has a touchscreen).
// Misses: a phone in "desktop site" mode (Android Chrome then reports a Linux
// desktop UA and mobile=false); it simply sees the desktop buttons.
// THEME_BOOT in src/seo.ts repeats this test before first paint (data-device
// on <html>, ADR-0029): change both together.
export function detectMobile(): boolean {
  if (typeof navigator === 'undefined') return false
  const uaData = (navigator as Navigator & { userAgentData?: NavigatorUAData }).userAgentData
  if (uaData?.mobile) return true
  const ua = navigator.userAgent
  if (/Android|iPhone|iPad|iPod/i.test(ua)) return true
  return /Macintosh/i.test(ua) && (navigator.maxTouchPoints ?? 0) > 1
}

/** The detected desktop OS, or null before hydration (prerender: unknown). */
export function useDetectedOS(): OS | null {
  return useSyncExternalStore<OS | null>(noSubscribe, detectOS, () => null)
}

/** True on phones and tablets; false before hydration. */
export function useIsMobile(): boolean {
  return useSyncExternalStore(noSubscribe, detectMobile, () => false)
}

/** True when the Web Share API exists; false before hydration. */
export function useCanShare(): boolean {
  return useSyncExternalStore(
    noSubscribe,
    () => typeof navigator !== 'undefined' && typeof navigator.share === 'function',
    () => false,
  )
}
