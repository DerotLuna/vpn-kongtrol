import React, { useEffect, useRef, useState } from 'react'
import { Lang } from '../../i18n'
import { Theme } from '../../theme'
import { GITHUB_REPO } from '../../links'
import { PageId, pathFor } from '../../site'
import { chooseLang } from '../../hooks/usePreferences'
import { useIsMobile } from '../../hooks/useVisitor'
import { CTA_COPY, downloadHref, DownloadIcon, GitHubIcon } from './downloadCta'
import SendToComputer from './SendToComputer'
import './download-cta.css'

interface Props {
  page: PageId
  lang: Lang
  theme: Theme
  setTheme: (theme: Theme) => void
}

type DocumentWithViewTransition = Document & {
  startViewTransition?: (callback: () => void) => { finished: Promise<void> }
}

export default function Nav({ page, lang, theme, setTheme }: Props) {
  const logoSrc = `${import.meta.env.BASE_URL}logo-kong.svg`
  const [langPulse, setLangPulse] = useState<string | null>(null)
  const [scrolled, setScrolled] = useState(false)
  const [menuOpen, setMenuOpen] = useState(false)
  const langPulseTimer = useRef<number | null>(null)
  const themeTransitioning = useRef(false)
  const navRef = useRef<HTMLElement>(null)
  const menuButton = useRef<HTMLButtonElement>(null)
  const mobile = useIsMobile()
  const cta = CTA_COPY[lang]
  // On the home page the header's download button waits until the hero's own
  // download button (#hero-cta) has left the viewport, so the two never
  // compete. It keeps its space while hidden (no layout shift) and is
  // visibility:hidden, so it is neither focusable nor announced. Other pages
  // show it from the start. Starts hidden on home in the prerender too.
  const [heroCtaInView, setHeroCtaInView] = useState(page === 'home')

  const toggleTheme = (e: React.MouseEvent<HTMLButtonElement>) => {
    if (themeTransitioning.current) return
    const next = theme === 'dark' ? 'light' : 'dark'
    const switchTheme = () => setTheme(next)
    const doc = document as DocumentWithViewTransition
    const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    if (!doc.startViewTransition || reduced) { switchTheme(); return }
    // the new theme grows as a circle from the button (index.css themeWipe)
    const r = e.currentTarget.getBoundingClientRect()
    const x = r.left + r.width / 2
    const y = r.top + r.height / 2
    const root = document.documentElement.style
    root.setProperty('--vt-x', `${x}px`)
    root.setProperty('--vt-y', `${y}px`)
    root.setProperty('--vt-r', `${Math.hypot(Math.max(x, innerWidth - x), Math.max(y, innerHeight - y))}px`)
    themeTransitioning.current = true
    doc.startViewTransition(switchTheme).finished.finally(() => {
      themeTransitioning.current = false
    })
  }
  const labels = lang === 'es'
    ? { langTitle: 'Idioma', themeTitle: 'Tema', dark: 'Oscuro', light: 'Claro', toggleTheme: 'Cambiar tema', toggleLang: 'Cambiar idioma', menu: 'Menú' }
    : { langTitle: 'Language', themeTitle: 'Theme', dark: 'Dark', light: 'Light', toggleTheme: 'Toggle theme', toggleLang: 'Toggle language', menu: 'Menu' }

  // Nav links styled as CLI flags. The top nav only navigates ACROSS pages —
  // in-page sections belong to the tmux bar at the bottom on both pages.
  const links = [
    { flag: lang === 'es' ? '--inicio' : '--home', href: pathFor('home', lang), current: page === 'home' },
    { flag: lang === 'es' ? '--guia' : '--guide', href: pathFor('guia', lang), current: page === 'guia' },
  ]

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 16)
    onScroll()
    window.addEventListener('scroll', onScroll, { passive: true })
    return () => window.removeEventListener('scroll', onScroll)
  }, [])

  useEffect(() => {
    if (page !== 'home') return
    const target = document.getElementById('hero-cta')
    if (!target || typeof IntersectionObserver === 'undefined') { setHeroCtaInView(false); return }
    // the sticky header covers the top --nav-h px: a button under it is not "in view"
    const navH = navRef.current?.offsetHeight ?? 0
    const io = new IntersectionObserver(
      ([entry]) => setHeroCtaInView(entry.isIntersecting),
      { rootMargin: `-${navH}px 0px 0px 0px` },
    )
    io.observe(target)
    return () => io.disconnect()
  }, [page])

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

  const handleLangToggle = () => {
    const next: Lang = lang === 'es' ? 'en' : 'es'
    chooseLang(next)
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
    <nav ref={navRef} className={`nav${scrolled ? ' is-sticky' : ''}`} aria-label={lang === 'es' ? 'Principal' : 'Main'}>
      <div className="nav-inner">
        <a href={page === 'home' ? '#motd' : pathFor('home', lang)} className="nav-logo">
          <img src={logoSrc} alt="Kongtrol" />
          <span className="nav-logo-name">
            <strong aria-hidden="true">K O N G T R O L</strong>
            <small className="beta">BETA</small>
          </span>
        </a>

        <div className="nav-right">
          <ul className="nav-links">
            {links.map(l => (
              <li key={l.flag}>
                <a href={l.href} className={l.current ? 'current' : ''} aria-current={l.current ? 'page' : undefined}>
                  <span className="nav-flag-dashes">--</span>{l.flag.slice(2)}
                </a>
              </li>
            ))}
          </ul>

          <span className={`nav-dl-slot${page === 'home' && heroCtaInView ? ' is-hidden' : ''}`}>
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
            <div className="nav-theme-switcher" title={labels.themeTitle}>
              <button
                className="nav-theme-toggle"
                onClick={toggleTheme}
                aria-label={`${labels.toggleTheme}: ${theme === 'dark' ? labels.dark : labels.light}`}
                title={`${labels.themeTitle}: ${theme === 'dark' ? labels.dark : labels.light}`}
              >
                <span className="theme-icon-stack">
                  <svg className="theme-icon-sun" viewBox="0 0 24 24" aria-hidden="true">
                    <path fill="currentColor" d="M12 18a6 6 0 1 1 0-12 6 6 0 0 1 0 12Zm0-16a1 1 0 0 1 1 1v2h-2V3a1 1 0 0 1 1-1Zm0 17a1 1 0 0 1 1 1v2h-2v-2a1 1 0 0 1 1-1Zm10-8v2h-2v-2h2ZM4 11v2H2v-2h2Zm14.95-6.54 1.41 1.41-1.42 1.42-1.41-1.42 1.42-1.41ZM6.47 16.95l1.41 1.41-1.41 1.42-1.42-1.42 1.42-1.41Zm12.48 2.83-1.41-1.42 1.41-1.41 1.42 1.41-1.42 1.42ZM6.47 7.05 5.05 5.63l1.42-1.41 1.41 1.41-1.41 1.42Z"/>
                  </svg>
                  <svg className="theme-icon-moon" viewBox="0 0 24 24" aria-hidden="true">
                    <path fill="currentColor" d="M21 14.2A9 9 0 1 1 9.8 3a7 7 0 1 0 11.2 11.2Z"/>
                  </svg>
                </span>
              </button>
            </div>

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
