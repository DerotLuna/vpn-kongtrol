// Kongtrol UI — app shell · v0.5.0 · 2026-11-19
//
// Layout route for every page (ADR-0038):
// - Header: logo top-left, aggregate VPN status, "＋ Agregar" (the add
//   sheet, ADR-0052), help (same place on every
//   page, WCAG 3.2.6), language, theme, version.
// - Navigation by width (Material 3 window classes, Apple HIG, Fluent
//   NavigationView): sidebar with labels ≥ 840px, rail 600–839px, bottom
//   bar under 600px with four sections + "Más" (a sheet with the rest).
// - App-wide notices (daemon stopped, helper missing/outdated) above the page;
//   daemon notices as toasts; "Tienes cambios sin guardar" before leaving a
//   form with edits (UnsavedChangesGuard).
// - After navigation, focus moves to the new page's h1 so screen readers
//   announce it; the page fades in (opacity/transform, 150ms, none with
//   reduced motion).
import { useEffect, useRef, useState } from 'react'
import { Link, NavLink, Outlet, useLocation } from 'react-router'
import { usePreferences } from '../../app/preferences-context'
import { SECTIONS } from '../../app/sections'
import { useNoticeToasts } from '../../data/home'
import { useHelperReadyToast } from '../../data/resources'
import { GUIDE_URL } from '../../links'
import { useAddSheet } from '../../app/addLink'
import { AddSheet } from '../add/AddSheet'
import { Button, IconButton } from '../ui/Button'
import { Dialog } from '../ui/Dialog'
import { BookIcon, MoonIcon, MoreIcon, PlusIcon, SunIcon } from '../ui/icons'
import { Logo } from '../ui/Logo'
import { Toaster } from '../ui/Toaster'
import { AggregateStatus, VersionTag } from './AggregateStatus'
import { AppNotices } from './AppNotices'
import { UnsavedChangesGuard } from './UnsavedChangesGuard'

function SideNav() {
  const { t } = usePreferences()
  return (
    <nav className="k-sidenav" aria-label={t('nav.label')}>
      <ul>
        {SECTIONS.map((s) => (
          <li key={s.id}>
            <NavLink to={s.path} end={s.path === '/'} className="k-nav-item">
              <span className="k-nav-item__icon">
                <s.icon />
              </span>
              <span className="k-nav-item__label">{t(s.label)}</span>
            </NavLink>
          </li>
        ))}
      </ul>
    </nav>
  )
}

function BottomBar() {
  const { t } = usePreferences()
  const [moreOpen, setMoreOpen] = useState(false)
  const { pathname } = useLocation()
  const secondary = SECTIONS.filter((s) => !s.mobile)
  const inMore = secondary.some((s) => pathname.startsWith(s.path))
  const close = () => setMoreOpen(false)
  return (
    <>
      <nav className="k-bottombar" aria-label={t('nav.label')}>
        <ul>
          {SECTIONS.filter((s) => s.mobile).map((s) => (
            <li key={s.id}>
              <NavLink to={s.path} end={s.path === '/'} className="k-nav-item">
                <span className="k-nav-item__icon">
                  <s.icon />
                </span>
                <span className="k-nav-item__label">{t(s.label)}</span>
              </NavLink>
            </li>
          ))}
          <li>
            <button
              type="button"
              className={`k-nav-item ${inMore ? 'active' : ''}`.trim()}
              aria-haspopup="dialog"
              aria-expanded={moreOpen}
              onClick={() => setMoreOpen(true)}
            >
              <span className="k-nav-item__icon">
                <MoreIcon />
              </span>
              <span className="k-nav-item__label">{t('nav.more')}</span>
            </button>
          </li>
        </ul>
      </nav>
      <Dialog open={moreOpen} onClose={close} title={t('nav.moreTitle')} variant="sheet">
        <ul className="k-more-list">
          {secondary.map((s) => (
            <li key={s.id}>
              <NavLink to={s.path} className="k-more-item" onClick={close}>
                <s.icon />
                {t(s.label)}
              </NavLink>
            </li>
          ))}
        </ul>
        <VersionTag />
      </Dialog>
    </>
  )
}

export function AppShell() {
  const { lang, setLang, t, theme, toggleTheme } = usePreferences()
  const { pathname } = useLocation()
  const first = useRef(true)
  const { open: openAdd } = useAddSheet()
  useHelperReadyToast()
  // Daemon notices (policy, watchdog, update) toast on every page, like the
  // tray; one poll for the whole app, shared with Inicio's list.
  useNoticeToasts()

  // Move focus to the new page's heading after a navigation (not on load).
  useEffect(() => {
    if (first.current) {
      first.current = false
      return
    }
    const heading = document.querySelector<HTMLElement>('#main h1')
    heading?.focus({ preventScroll: true })
    window.scrollTo({ top: 0 })
  }, [pathname])

  const themeLabel = theme === 'dark' ? t('header.theme.toLight') : t('header.theme.toDark')
  return (
    <div className="k-app">
      <a className="skip-link" href="#main">
        {t('app.skip')}
      </a>
      <header className="k-topbar">
        <Link className="k-brand" to="/">
          <Logo size={32} />
          <span>{t('app.name')}</span>
        </Link>
        <AggregateStatus />
        <div className="k-topbar__actions" role="group" aria-label={t('header.toolbar')}>
          <Button
            variant="tonal"
            className="k-topbar__add"
            icon={<PlusIcon />}
            aria-haspopup="dialog"
            onClick={() => openAdd({ mode: 'menu' })}
          >
            {t('add.button')}
          </Button>
          <a className="k-topbar__link" href={GUIDE_URL} target="_blank" rel="noopener noreferrer">
            <BookIcon />
            <span className="k-topbar__label">{t('help.guide')}</span>
            <span className="visually-hidden"> {t('help.newTab')}</span>
          </a>
          <button
            type="button"
            className="k-icon-btn k-icon-btn--standard k-lang"
            onClick={() => setLang(lang === 'es' ? 'en' : 'es')}
          >
            {/* Visible "EN"/"ES" stays in the accessible name (WCAG 2.5.3). */}
            {lang === 'es' ? 'EN' : 'ES'}
            <span className="visually-hidden"> · {t('header.lang.switchLabel')}</span>
          </button>
          <IconButton label={themeLabel} icon={theme === 'dark' ? <SunIcon /> : <MoonIcon />} onClick={toggleTheme} />
          <VersionTag className="k-topbar__version" />
        </div>
      </header>
      <SideNav />
      <main id="main" tabIndex={-1} className="k-main">
        <AppNotices />
        <div key={pathname} className="k-page">
          <Outlet />
        </div>
      </main>
      <BottomBar />
      <Toaster />
      <AddSheet />
      <UnsavedChangesGuard />
    </div>
  )
}
