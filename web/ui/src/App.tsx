// Kongtrol UI — providers + routes · v0.2.0 · 2026-11-18
//
// React Router (declarative mode) under /app (ADR-0037); the daemon serves
// index.html for any /app/... path (internal/api/ui_assets.go), so deep
// links and reloads work. Every section has its route now; phase 2 replaces
// the placeholders page by page (src/app/placeholders.ts).
import { lazy, Suspense } from 'react'
import { BrowserRouter, Route, Routes } from 'react-router'
import type { QueryClient } from '@tanstack/react-query'
import { DataProvider } from './app/DataProvider'
import { BASENAME, PLACEHOLDERS } from './app/placeholders'
import { PreferencesProvider } from './app/PreferencesProvider'
import { AppShell } from './components/layout/AppShell'
import { NotFoundPage } from './pages/NotFoundPage'
import { PlaceholderPage } from './pages/PlaceholderPage'
import { WelcomePage } from './pages/WelcomePage'

// Component review page: development builds only, never shipped.
const KitPage = import.meta.env.DEV ? lazy(() => import('./pages/KitPage')) : null

function AppRoutes() {
  return (
    <Routes>
      <Route element={<AppShell />}>
        <Route index element={<WelcomePage />} />
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
