// Kongtrol UI — app shell · v0.6.0 · 2026-11-19
//
// Layout route for every page (ADR-0038):
// - Header: logo top-left, aggregate VPN status, "＋ Agregar" (the add
//   sheet, ADR-0055), help (same place on every page, WCAG 3.2.6),
//   version. Language and theme live only in Ajustes › Idioma y apariencia
//   (owner feedback 2026-11-19: the header copies were duplicates).
// - Navigation by width (Material 3 window classes, Apple HIG, Fluent
//   NavigationView): ≥ 600px a side menu that collapses to an icon rail
//   with tooltips ("Contraer menú", remembered on this device, collapsed
//   by default under 1100px: app/navCollapse.ts); under 600px a bottom
//   bar with four sections + "Más" (a sheet with the rest).
// - App-wide notices (daemon stopped, helper missing/outdated) above the page;
//   daemon notices as toasts; "Tienes cambios sin guardar" before leaving a
//   form with edits (UnsavedChangesGuard).
// - After navigation, focus moves to the new page's h1 so screen readers
//   announce it; the page fades in (opacity/transform, 150ms, none with
//   reduced motion).
import { useEffect, useId, useRef, useState } from 'react'
import { Link, NavLink, Outlet, useLocation } from 'react-router'
import { usePreferences } from '../../app/preferences-context'
import { SECTIONS } from '../../app/sections'
import { useNoticeToasts } from '../../data/home'
import { useHelperReadyToast } from '../../data/resources'
import { GUIDE_URL } from '../../links'
import { useAddSheet } from '../../app/addLink'
import { AddSheet } from '../add/AddSheet'
import { useNavCollapse } from '../../app/navCollapse'
import { Button } from '../ui/Button'
import { Dialog } from '../ui/Dialog'
import { BookIcon, MoreIcon, PlusIcon, SidebarIcon } from '../ui/icons'
import { Tooltip } from '../ui/Tooltip'
import { Logo } from '../ui/Logo'
import { Toaster } from '../ui/Toaster'
import { AggregateStatus, VersionTag } from './AggregateStatus'
import { AppNotices } from './AppNotices'
import { UnsavedChangesGuard } from './UnsavedChangesGuard'

function SideNav({ collapsed, onToggle }: { collapsed: boolean; onToggle: () => void }) {
  const { t } = usePreferences()
  const listId = useId()
  const toggleLabel = t(collapsed ? 'nav.expand' : 'nav.collapse')
  return (
    <nav className="k-sidenav" aria-label={t('nav.label')}>
      <ul id={listId}>
        {SECTIONS.map((s) => (
          <li key={s.id}>
            {/* Collapsed: icon only, the name stays in the link (visually
                hidden) and shows as a tooltip on hover or keyboard focus. */}
            <Tooltip content={t(s.label)} describe={false} placement="right" disabled={!collapsed}>
              <NavLink to={s.path} end={s.path === '/'} className="k-nav-item">
                <span className="k-nav-item__icon">
                  <s.icon />
                </span>
                <span className="k-nav-item__label">{t(s.label)}</span>
              </NavLink>
            </Tooltip>
          </li>
        ))}
      </ul>
      <Tooltip content={toggleLabel} placement="right" disabled={!collapsed}>
        <button type="button" className="k-nav-item k-nav-toggle" aria-expanded={!collapsed} aria-controls={listId} aria-label={toggleLabel} onClick={onToggle}>
          <span className="k-nav-item__icon">
            <SidebarIcon />
          </span>
          <span className="k-nav-item__label" aria-hidden="true">
            {toggleLabel}
          </span>
        </button>
      </Tooltip>
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
  const { t } = usePreferences()
  const nav = useNavCollapse()
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

  return (
    <div className="k-app" data-nav={nav.collapsed ? 'collapsed' : 'expanded'}>
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
          <VersionTag className="k-topbar__version" />
        </div>
      </header>
      <div className="k-navcol">
        <SideNav collapsed={nav.collapsed} onToggle={nav.toggle} />
      </div>
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
