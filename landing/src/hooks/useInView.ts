import { RefObject, useEffect, useState } from 'react'

// Whether an element is on screen, via IntersectionObserver — version 1.0.0 · 2026-11-18.
// Looping demos use it to animate only while visible (ADR-0031): off screen
// they cost nothing, and nothing runs during page load unless it is in view.
// Starts false on the server and the client alike, so hydration matches;
// the observer reports the real state right after.
export function useInView(ref: RefObject<Element>): boolean {
  const [inView, setInView] = useState(false)
  useEffect(() => {
    const el = ref.current
    if (!el) return
    if (typeof IntersectionObserver === 'undefined') { setInView(true); return }
    const io = new IntersectionObserver(([e]) => setInView(e.isIntersecting))
    io.observe(el)
    return () => io.disconnect()
  }, [ref])
  return inView
}
