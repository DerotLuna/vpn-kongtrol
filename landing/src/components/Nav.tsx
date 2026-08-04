import { useEffect, useRef, useState } from 'react'
import { Lang } from '../i18n'
import { Theme } from '../theme'
import { GITHUB_RELEASES } from '../links'

interface Props {
  page: 'home' | 'guide' | 'terminos'
  lang: Lang
  setLang: (lang: Lang) => void
  theme: Theme
  setTheme: (theme: Theme) => void
}

type DocumentWithViewTransition = Document & {
  startViewTransition?: (callback: () => void) => { finished: Promise<void> }
}

export default function Nav({ page, lang, setLang, theme, setTheme }: Props) {
  const logoSrc = `${import.meta.env.BASE_URL}logo-kong.svg`
  const [langPulse, setLangPulse] = useState<string | null>(null)
  const [scrolled, setScrolled] = useState(false)
  const [menuOpen, setMenuOpen] = useState(false)
  const langPulseTimer = useRef<number | null>(null)
  const themeTransitioning = useRef(false)

  const toggleTheme = () => {
    if (themeTransitioning.current) return
    const next = theme === 'dark' ? 'light' : 'dark'
    const switchTheme = () => setTheme(next)
    const doc = document as DocumentWithViewTransition
    if (!doc.startViewTransition) { switchTheme(); return }
    themeTransitioning.current = true
    doc.startViewTransition(switchTheme).finished.finally(() => {
      themeTransitioning.current = false
    })
  }
  const labels = lang === 'es'
    ? { langTitle: 'Idioma', themeTitle: 'Tema', dark: 'Oscuro', light: 'Claro', toggleTheme: 'Cambiar tema', toggleLang: 'Cambiar idioma', download: 'Descargar', menu: 'Menú' }
    : { langTitle: 'Language', themeTitle: 'Theme', dark: 'Dark', light: 'Light', toggleTheme: 'Toggle theme', toggleLang: 'Toggle language', download: 'Download', menu: 'Menu' }

  // Nav links styled as CLI flags. The top nav only navigates ACROSS pages —
  // in-page sections belong to the tmux bar at the bottom on both pages.
  const links = [
    { flag: lang === 'es' ? '--inicio' : '--home', href: import.meta.env.BASE_URL, current: page === 'home' },
    { flag: lang === 'es' ? '--guia' : '--guide', href: `${import.meta.env.BASE_URL}guia`, current: page === 'guide' },
  ]

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 16)
    onScroll()
    window.addEventListener('scroll', onScroll, { passive: true })
    return () => window.removeEventListener('scroll', onScroll)
  }, [])

  useEffect(() => {
    return () => {
      if (langPulseTimer.current != null) {
        window.clearTimeout(langPulseTimer.current)
      }
    }
  }, [])

  useEffect(() => {
    if (!menuOpen) return
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape') setMenuOpen(false) }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [menuOpen])

  const handleLangToggle = () => {
    const next: Lang = lang === 'es' ? 'en' : 'es'
    setLang(next)
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
    <nav className={`nav${scrolled ? ' is-sticky' : ''}`}>
      <div className="nav-inner">
        <a href={page === 'home' ? '#motd' : import.meta.env.BASE_URL} className="nav-logo">
          <img src={logoSrc} alt="Kongtrol" className={theme === 'dark' ? 'logo-dark-invert' : ''} />
          <span className="nav-logo-name">
            <strong>K O N G T R O L</strong>
            <small>CLI</small>
            <small className="beta">BETA</small>
          </span>
        </a>

        <div className="nav-right">
          <ul className="nav-links">
            {links.map(l => (
              <li key={l.flag}>
                <a href={l.href} className={l.current ? 'current' : ''}>
                  <span className="nav-flag-dashes">--</span>{l.flag.slice(2)}
                </a>
              </li>
            ))}
          </ul>

          <a href={GITHUB_RELEASES} className="nav-cta" target="_blank" rel="noreferrer" aria-label={labels.download}>
            <svg viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
              <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27s1.36.09 2 .27c1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8Z"/>
            </svg>
            <span className="nav-cta-label">{labels.download}</span>
          </a>

          <div className="nav-controls nav-utils">
            <div className="nav-theme-switcher" title={labels.themeTitle}>
              <button
                className="nav-theme-toggle"
                onClick={toggleTheme}
                aria-label={`${labels.toggleTheme}: ${theme === 'dark' ? labels.dark : labels.light}`}
                title={`${labels.themeTitle}: ${theme === 'dark' ? labels.dark : labels.light}`}
              >
                <span className={`theme-icon-stack ${theme === 'dark' ? 'dark' : 'light'}`}>
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
              <button
                className="nav-lang-toggle"
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
              </button>
            </div>
          </div>

          <button
            type="button"
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
              <a href={l.href} className={l.current ? 'current' : ''} onClick={() => setMenuOpen(false)}>
                <span className="nav-flag-dashes">--</span>{l.flag.slice(2)}
              </a>
            </li>
          ))}
        </ul>
      </div>
    </nav>
  )
}
