import type { ReactNode } from 'react'
import logoUrl from '@brand/logo-kong-64.png'
import { usePreferences } from '../../app/preferences-context'
import { GUIDE_URL, LEGACY } from '../../links'
import { ExternalLink } from '../ui/ExternalLink'
import { BookIcon, GridIcon, MoonIcon, SunIcon } from '../ui/icons'

/** Header (logo top-left, help, classic dashboard, language, theme) + footer. */
export function AppShell({ children, version }: { children: ReactNode; version?: string }) {
  const { lang, setLang, t, theme, toggleTheme } = usePreferences()
  const themeLabel = theme === 'dark' ? t('header.theme.toLight') : t('header.theme.toDark')
  return (
    <div className="shell">
      <a className="skip-link" href="#main">
        {t('app.skip')}
      </a>
      <header className="app-header">
        <div className="container">
          <a className="brand" href="/app/">
            <img src={logoUrl} alt="" width="32" height="32" />
            {t('app.name')}
          </a>
          <nav className="header-actions" aria-label={t('app.name')}>
            <a className="header-link" href={GUIDE_URL} target="_blank" rel="noopener noreferrer">
              <BookIcon />
              <span className="label">{t('help.guide')}</span>
              <span className="visually-hidden"> {t('help.newTab')}</span>
            </a>
            <a className="header-link" href={LEGACY.home}>
              <GridIcon />
              <span className="label">{t('header.legacy')}</span>
            </a>
            <button
              type="button"
              className="icon-btn"
              aria-label={t('header.lang.switchLabel')}
              title={t('header.lang.switch')}
              onClick={() => setLang(lang === 'es' ? 'en' : 'es')}
            >
              {lang === 'es' ? 'EN' : 'ES'}
            </button>
            <button
              type="button"
              className="icon-btn"
              aria-label={themeLabel}
              title={themeLabel}
              onClick={toggleTheme}
            >
              {theme === 'dark' ? <SunIcon /> : <MoonIcon />}
            </button>
          </nav>
        </div>
      </header>
      <main id="main" tabIndex={-1}>
        <div className="container">{children}</div>
      </main>
      <footer className="app-footer">
        <div className="container">
          <span>{version ? t('app.footer.version', { version }) : t('app.name')}</span>
          <ExternalLink href={GUIDE_URL}>{t('help.guide')}</ExternalLink>
        </div>
      </footer>
    </div>
  )
}
