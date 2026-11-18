// Kongtrol UI — providers + routes · v0.5.0 · 2026-11-18
//
// React Router in data mode (createBrowserRouter) at / (ADR-0038, ADR-0041,
// ADR-0043): the same routes as before, plus what only the data router
// offers — useBlocker for "Tienes cambios sin guardar" (UnsavedChangesGuard)
// and route-level code splitting (`lazy`): each page loads with its first
// visit and the old page stays on screen until it is ready, so there is no
// loading flash. The daemon serves index.html for any page path
// (internal/api/ui_assets.go), so deep links and reloads work. /tray is
// the tray flyout (ADR-0036): its own compact page, outside the panel
// shell, sharing the same data layer.
import { useState } from 'react'
import { createBrowserRouter, createRoutesFromElements, Route, RouterProvider } from 'react-router'
import type { QueryClient } from '@tanstack/react-query'
import { DataProvider } from './app/DataProvider'
import { PreferencesProvider } from './app/PreferencesProvider'
import { AppShell } from './components/layout/AppShell'
import { HomePage } from './pages/HomePage'

/** Router basename: Vite's base without the trailing slash, "/" at the site root. */
const BASENAME = import.meta.env.BASE_URL.replace(/\/$/, '') || '/'

const routes = createRoutesFromElements(
  <>
    <Route path="tray" lazy={() => import('./pages/TrayPage').then((m) => ({ Component: m.TrayPage }))} />
    <Route element={<AppShell />}>
      {/* Inicio is the usual first page: it ships in the main bundle. */}
      <Route index element={<HomePage />} />
      <Route path="profiles" lazy={() => import('./pages/ProfilesPage').then((m) => ({ Component: m.ProfilesPage }))} />
      <Route path="profiles/groups" lazy={() => import('./pages/GroupsPage').then((m) => ({ Component: m.GroupsPage }))} />
      <Route path="rules" lazy={() => import('./pages/RulesPage').then((m) => ({ Component: m.RulesPage }))} />
      <Route path="rules/schedule" lazy={() => import('./pages/SchedulePage').then((m) => ({ Component: m.SchedulePage }))} />
      <Route path="traffic" lazy={() => import('./pages/TrafficPage').then((m) => ({ Component: m.TrafficPage }))} />
      <Route path="security" lazy={() => import('./pages/SecurityPage').then((m) => ({ Component: m.SecurityPage }))} />
      <Route path="activity" lazy={() => import('./pages/ActivityPage').then((m) => ({ Component: m.ActivityPage }))} />
      <Route path="settings" lazy={() => import('./pages/SettingsPage').then((m) => ({ Component: m.SettingsPage }))} />
      {/* Component review page: development builds only, never shipped. */}
      {import.meta.env.DEV && <Route path="_kit" lazy={() => import('./pages/KitPage').then((m) => ({ Component: m.default }))} />}
      <Route path="*" lazy={() => import('./pages/NotFoundPage').then((m) => ({ Component: m.NotFoundPage }))} />
    </Route>
  </>,
)

export function App({ client }: { client?: QueryClient }) {
  // One router per App (tests mount several): it reads the URL once, here.
  const [router] = useState(() => createBrowserRouter(routes, { basename: BASENAME }))
  return (
    <PreferencesProvider>
      <DataProvider {...(client ? { client } : {})}>
        <RouterProvider router={router} />
      </DataProvider>
    </PreferencesProvider>
  )
}
