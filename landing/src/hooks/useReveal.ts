import { useEffect } from 'react'

// Scroll reveals for browsers without CSS scroll-driven animations
// (animation-timeline: view()). Where that CSS is supported this does nothing:
// index.css animates .reveal on its own. Elsewhere it arms .reveal-io on <html>
// and adds .is-in to each .reveal as it enters the viewport.
//
// Never hides what the visitor can already see: elements on screen when the
// hook runs are marked .is-in before the class that hides the rest is set.
// Skipped entirely under prefers-reduced-motion, and without JS nothing is
// ever hidden (the hiding rule needs .reveal-io).
export function useReveal() {
  useEffect(() => {
    const css = typeof CSS !== 'undefined' && CSS.supports
    if (css && CSS.supports('animation-timeline: view()')) return
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return
    if (!('IntersectionObserver' in window)) return

    const els = Array.from(document.querySelectorAll<HTMLElement>('.reveal'))
    // read every position first, then write: interleaving setProperty with
    // getBoundingClientRect forced one layout per element (ADR-0031)
    const vh = window.innerHeight
    const onScreen = els.map(el => el.getBoundingClientRect().top < vh)
    els.forEach((el, n) => {
      // stagger siblings: 2nd card waits 70ms, 3rd 140ms… (index.css --i)
      const sibs = el.parentElement ? Array.from(el.parentElement.children).filter(c => c.classList.contains('reveal')) : []
      el.style.setProperty('--i', String(Math.min(sibs.indexOf(el), 5)))
      if (onScreen[n]) el.classList.add('is-in')
    })

    const io = new IntersectionObserver(entries => {
      entries.forEach(e => {
        if (!e.isIntersecting) return
        e.target.classList.add('is-in')
        io.unobserve(e.target)
      })
    }, { rootMargin: '0px 0px -8% 0px' })
    els.forEach(el => { if (!el.classList.contains('is-in')) io.observe(el) })
    document.documentElement.classList.add('reveal-io')

    return () => {
      io.disconnect()
      document.documentElement.classList.remove('reveal-io')
    }
  }, [])
}
