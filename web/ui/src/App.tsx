// Kongtrol UI — providers + routes · v0.3.0 · 2026-11-18
//
// React Router (declarative mode) under /app (ADR-0038); the daemon serves
// index.html for any /app/... path (internal/api/ui_assets.go), so deep
// links and reloads work. Every section has its route now; phase 2 replaces
// the placeholders page by page (src/app/placeholders.ts). /app/tray is the
// tray flyout (ADR-0036): its own compact page, outside the panel shell,
// sharing the same data layer.
import { lazy, Suspense } from 'react'
import { BrowserRouter, Route, Routes } from 'react-router'
import type { QueryClient } from '@tanstack/react-query'
import { DataProvider } from './app/DataProvider'
import { BASENAME, PLACEHOLDERS } from './app/placeholders'
import { PreferencesProvider } from './app/PreferencesProvider'
import { AppShell } from './components/layout/AppShell'
import { ActivityPage } from './pages/ActivityPage'
import { NotFoundPage } from './pages/NotFoundPage'
import { PlaceholderPage } from './pages/PlaceholderPage'
import { RulesPage } from './pages/RulesPage'
import { SchedulePage } from './pages/SchedulePage'
import { SecurityPage } from './pages/SecurityPage'
import { SettingsPage } from './pages/SettingsPage'
import { TrafficPage } from './pages/TrafficPage'
import { TrayPage } from './pages/TrayPage'
import { HomePage } from './pages/HomePage'

// Component review page: development builds only, never shipped.
const KitPage = import.meta.env.DEV ? lazy(() => import('./pages/KitPage')) : null

function AppRoutes() {
  return (
    <Routes>
      <Route path="tray" element={<TrayPage />} />
      <Route element={<AppShell />}>
        <Route index element={<HomePage />} />
        <Route path="traffic" element={<TrafficPage />} />
        <Route path="activity" element={<ActivityPage />} />
        <Route path="settings" element={<SettingsPage />} />
        <Route path="security" element={<SecurityPage />} />
        <Route path="rules" element={<RulesPage />} />
        <Route path="rules/schedule" element={<SchedulePage />} />
        {Object.entries(PLACEHOLDERS).map(([path, spec]) => (
          <Route key={path} path={path} element={<PlaceholderPage spec={spec} />} />
        ))}
        {KitPage && (
          <Route
            path="_kit"
            element={
              <Suspense fallback={null}>
                <KitPage />
              </Suspense>
            }
          />
        )}
        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  )
}

export function App({ client }: { client?: QueryClient }) {
  return (
    <PreferencesProvider>
      <DataProvider {...(client ? { client } : {})}>
        <BrowserRouter basename={BASENAME}>
          <AppRoutes />
        </BrowserRouter>
      </DataProvider>
    </PreferencesProvider>
  )
}
