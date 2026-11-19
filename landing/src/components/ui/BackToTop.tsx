import { useEffect, useState } from 'react'
import { Lang } from '../../i18n'

// Back to top (2026-11-19, ADR-0060): the common pattern — a round icon
// button fixed bottom-right (above the tmux bar), shown once the visitor has
// scrolled past about one screen (so always on the footer). It scrolls to the
// top (instantly under reduced motion) and moves focus to <main>
// (tabIndex -1), so keyboard and screen-reader users land there too. Hidden =
// visibility:hidden, so it is neither focusable nor announced. Rendered by
// Footer, so every page has it; prerendered hidden (the server has no
// scroll), so hydration matches.
export default function BackToTop({ lang }: { lang: Lang }) {
  const [shown, setShown] = useState(false)
  const label = lang === 'es' ? 'Volver al inicio' : 'Back to top'

  useEffect(() => {
    let frame = 0
    const read = () => { frame = 0; setShown(window.scrollY > window.innerHeight * 0.9) }
    const onScroll = () => { if (!frame) frame = requestAnimationFrame(read) }
    onScroll()
    window.addEventListener('scroll', onScroll, { passive: true })
    return () => { window.removeEventListener('scroll', onScroll); cancelAnimationFrame(frame) }
  }, [])

  const toTop = () => {
    const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    window.scrollTo({ top: 0, behavior: reduced ? 'auto' : 'smooth' })
    document.getElementById('main')?.focus({ preventScroll: true })
  }

  return (
    <button
      type="button"
      className={`to-top tone-ink${shown ? ' is-shown' : ''}`}
      onClick={toTop}
      aria-label={label}
      title={label}
    >
      {/* same stroke style as the hero's down arrow (Hero.tsx ArrowDownIcon) */}
      <svg width="18" height="18" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <path d="M8 13.5v-11M3.5 7 8 2.5 12.5 7" />
      </svg>
    </button>
  )
}
