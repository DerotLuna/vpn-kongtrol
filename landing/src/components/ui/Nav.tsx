import { MouseEvent, useEffect, useRef, useState } from 'react'
import { Lang } from '../../i18n'
import { GITHUB_REPO } from '../../links'
import { PageId, pathFor } from '../../site'
import { chooseLang } from '../../hooks/usePreferences'
import { useIsMobile } from '../../hooks/useVisitor'
import { CTA_COPY, DOWNLOAD_SECTION_ID, downloadHref, DownloadIcon, FOOTER_ID, GitHubIcon } from './downloadCta'
import SendToComputer from './SendToComputer'
import './download-cta.css'

interface Props {
  page: PageId
  lang: Lang
}

// The section the visitor is reading: the last page anchor (home sections,
// guide and legal sections — the same ids in both languages) whose top is
// above a reading line a third of the way down the screen. Null near the
// top of the page.
function currentAnchor(navH: number): string | null {
  if (window.scrollY < 40) return null
  let id: string | null = null
  for (const el of document.querySelectorAll<HTMLElement>('main section.section[id], main .guide-section[id]')) {
    if (el.getBoundingClientRect().top <= navH + (window.innerHeight - navH) / 3) id = el.id
  }
  return id
}

export default function Nav({ page, lang }: Props) {
  // the line-art original, ≥52 px (DESIGN.md "Logo", ADR-0060)
  const logoSrc = `${import.meta.env.BASE_URL}logo-kong.svg`
  const [langPulse, setLangPulse] = useState<string | null>(null)
  const [scrolled, setScrolled] = useState(false)
  const [menuOpen, setMenuOpen] = useState(false)
  // the header steps aside while the full-height footer is on screen (it
  // carries its own logo, download button and links) and comes back on leaving
  const [footerInView, setFooterInView] = useState(false)
  const langPulseTimer = useRef<number | null>(null)
  const navRef = useRef<HTMLElement>(null)
  const menuButton = useRef<HTMLButtonElement>(null)
  const mobile = useIsMobile()
  const cta = CTA_COPY[lang]
  // On the home page the header's download button hides while the hero's own
  // download button (#hero-cta) or the download section (#install) is in view,
  // so it never competes with them. It keeps its space while hidden (no layout
  // shift) and is visibility:hidden, so it is neither focusable nor announced.
  // Other pages show it from the start. Starts hidden on home in the
  // prerender too.
  const [ownCtaInView, setOwnCtaInView] = useState(page === 'home')

  const labels = lang === 'es'
    ? { langTitle: 'Idioma', toggleLang: 'Cambiar idioma', menu: 'Menú' }
    : { langTitle: 'Language', toggleLang: 'Toggle language', menu: 'Menu' }

  // Nav links styled as CLI flags. The top nav only navigates ACROSS pages —
  // in-page sections belong to the tmux bar at the bottom on both pages.
  const links = [
    { flag: lang === 'es' ? '--inicio' : '--home', href: pathFor('home', lang), current: page === 'home' },
    { flag: lang === 'es' ? '--guia' : '--guide', href: pathFor('guia', lang), current: page === 'guia' },
  ]

  // smooth in-page scrolling only from here on (index.css html[data-ready])
  useEffect(() => { document.documentElement.dataset.ready = '' }, [])

  useEffect(() => {
    // one read per frame, at its start (rAF), never in the middle of DOM writes
    let frame = 0
    const read = () => { frame = 0; setScrolled(window.scrollY > 16) }
    const onScroll = () => { if (!frame) frame = requestAnimationFrame(read) }
    onScroll()
    window.addEventListener('scroll', onScroll, { passive: true })
    return () => {
      window.removeEventListener('scroll', onScroll)
      cancelAnimationFrame(frame)
    }
  }, [])

  useEffect(() => {
    if (page !== 'home') return
    const targets = ['hero-cta', DOWNLOAD_SECTION_ID]
      .map(id => document.getElementById(id))
      .filter((el): el is HTMLElement => el !== null)
    if (targets.length === 0 || typeof IntersectionObserver === 'undefined') { setOwnCtaInView(false); return }
    // The sticky header covers the top --nav-h px: a button under it is not
    // "in view". Its height comes from a ResizeObserver, delivered after layout,
    // instead of offsetHeight, which forced a layout during hydration (ADR-0031).
    const nav = navRef.current
    let io: IntersectionObserver | null = null
    let navH = -1
    const watch = (h: number) => {
      if (h === navH) return
      navH = h
      io?.disconnect()
      const inView = new Map<Element, boolean>()
      io = new IntersectionObserver(
        entries => {
          for (const e of entries) inView.set(e.target, e.isIntersecting)
          setOwnCtaInView([...inView.values()].some(Boolean))
        },
        { rootMargin: `-${h}px 0px 0px 0px` },
      )
      for (const t of targets) io.observe(t)
    }
    if (!nav || typeof ResizeObserver === 'undefined') {
      watch(0)
      return () => io?.disconnect()
    }
    const ro = new ResizeObserver(([e]) => watch(Math.round(e.borderBoxSize?.[0]?.blockSize ?? e.contentRect.height)))
    ro.observe(nav)
    return () => { ro.disconnect(); io?.disconnect() }
  }, [page])

  useEffect(() => {
    const footer = document.getElementById(FOOTER_ID)
    if (!footer || typeof IntersectionObserver === 'undefined') return
    // "in view" once the footer's top passes the middle of the viewport, so
    // the header does not vanish while the last section is still being read
    const io = new IntersectionObserver(([e]) => setFooterInView(e.isIntersecting), { rootMargin: '0px 0px -50% 0px' })
    io.observe(footer)
    return () => io.disconnect()
  }, [])

  useEffect(() => {
    return () => {
      if (langPulseTimer.current != null) {
        window.clearTimeout(langPulseTimer.current)
      }
    }
  }, [])

  // disclosure menu: Escape closes and gives focus back to the button; a tap
  // or click outside the header closes it too
  useEffect(() => {
    if (!menuOpen) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== 'Escape') return
      setMenuOpen(false)
      menuButton.current?.focus()
    }
    const onPointer = (e: PointerEvent) => {
      if (navRef.current && !navRef.current.contains(e.target as Node)) setMenuOpen(false)
    }
    document.addEventListener('keydown', onKey)
    document.addEventListener('pointerdown', onPointer)
    return () => {
      document.removeEventListener('keydown', onKey)
      document.removeEventListener('pointerdown', onPointer)
    }
  }, [menuOpen])

  // ES ↔ EN is a navigation to the other prerendered page (separate URLs and
  // hreflang stay). It keeps the visitor where they were: the link gets the
  // current section as its fragment just before the browser follows it, and
  // the page change cross-fades (index.css @view-transition, ADR-0060).
  const handleLangToggle = (e: MouseEvent<HTMLAnchorElement>) => {
    const next: Lang = lang === 'es' ? 'en' : 'es'
    chooseLang(next)
    const anchor = currentAnchor(navRef.current?.getBoundingClientRect().height ?? 0)
    e.currentTarget.href = pathFor(page, next) + (anchor ? '#' + anchor : '')
    setLangPulse(next.toUpperCase())
    if (langPulseTimer.current != null) {
      window.clearTimeout(langPulseTimer.current)
    }
    langPulseTimer.current = window.setTimeout(() => {
      setLangPulse(null)
      langPulseTimer.current = null
    }, 700)
  }

  return (
    <nav ref={navRef} className={`nav tone-ink${scrolled ? ' is-sticky' : ''}${footerInView && !menuOpen ? ' is-away' : ''}`} aria-label={lang === 'es' ? 'Principal' : 'Main'}>
      <div className="nav-inner">
        <a href={page === 'home' ? '#motd' : pathFor('home', lang)} className="nav-logo">
          <img className="logo-mark" src={logoSrc} alt="Kongtrol" width={54} height={54} />
          <span className="nav-logo-name">
            <strong aria-hidden="true">K O N G T R O L</strong>
            <small className="beta">BETA</small>
          </span>
        </a>

        {/* Links sit next to the logo; the flexible middle column absorbs the
            reserved slot of the hidden download CTA, so no hole shows (ADR-0027). */}
        <ul className="nav-links">
          {links.map(l => (
            <li key={l.flag}>
              <a href={l.href} className={l.current ? 'current' : ''} aria-current={l.current ? 'page' : undefined}>
                <span className="nav-flag-dashes">--</span>{l.flag.slice(2)}
              </a>
            </li>
          ))}
        </ul>

        <div className="nav-right">
          <span className={`nav-dl-slot${page === 'home' && ownCtaInView ? ' is-hidden' : ''}`}>
            {mobile ? (
              <SendToComputer lang={lang} variant="compact" />
            ) : (
              <a href={downloadHref(page, lang)} className="nav-dl">
                <DownloadIcon size={14} />
                <span className="nav-dl-label">{cta.navDownload}</span>
              </a>
            )}
          </span>

          <a href={GITHUB_REPO} className="nav-gh" target="_blank" rel="noreferrer" aria-label={cta.githubLabel} title={cta.githubLabel}>
            <GitHubIcon size={18} />
          </a>

          <div className="nav-controls nav-utils">
            <div className="nav-lang-switcher" title={labels.langTitle}>
              <a
                className="nav-lang-toggle"
                href={pathFor(page, lang === 'es' ? 'en' : 'es')}
                hrefLang={lang === 'es' ? 'en' : 'es'}
                onClick={handleLangToggle}
                aria-label={`${labels.toggleLang}: ${lang.toUpperCase()}`}
                title={`${labels.langTitle}: ${lang.toUpperCase()}`}
              >
                <span className={`lang-icon ${langPulse ? 'hide' : ''}`}>
                  <svg viewBox="0 0 24 24" aria-hidden="true">
                    <path fill="currentColor" d="M12 2a10 10 0 1 0 10 10A10 10 0 0 0 12 2Zm7.9 9h-3.2a15.9 15.9 0 0 0-1.1-5.1A8.04 8.04 0 0 1 19.9 11ZM12 4.1A14 14 0 0 1 14.6 11H9.4A14 14 0 0 1 12 4.1ZM8.4 5.9A15.9 15.9 0 0 0 7.3 11H4.1a8.04 8.04 0 0 1 4.3-5.1ZM4.1 13h3.2a15.9 15.9 0 0 0 1.1 5.1A8.04 8.04 0 0 1 4.1 13Zm7.9 6.9A14 14 0 0 1 9.4 13h5.2A14 14 0 0 1 12 19.9Zm3.6-1.8a15.9 15.9 0 0 0 1.1-5.1h3.2a8.04 8.04 0 0 1-4.3 5.1Z"/>
                  </svg>
                </span>
                <span className={`lang-flash ${langPulse ? 'show' : ''}`}>{langPulse ?? ''}</span>
              </a>
            </div>
          </div>

          <button
            type="button"
            ref={menuButton}
            className={`nav-hamburger${menuOpen ? ' open' : ''}`}
            aria-label={labels.menu}
            aria-expanded={menuOpen}
            aria-controls="nav-mobile-menu"
            onClick={() => setMenuOpen(o => !o)}
          >
            <span /><span /><span />
          </button>
        </div>
      </div>

      <div id="nav-mobile-menu" className={`nav-mobile-menu${menuOpen ? ' open' : ''}`}>
        <ul>
          {links.map(l => (
            <li key={l.flag}>
              <a href={l.href} className={l.current ? 'current' : ''} aria-current={l.current ? 'page' : undefined} onClick={() => setMenuOpen(false)}>
                <span className="nav-flag-dashes">--</span>{l.flag.slice(2)}
              </a>
            </li>
          ))}
        </ul>
      </div>
    </nav>
  )
}
