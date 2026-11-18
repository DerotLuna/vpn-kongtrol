import { PreferencesProvider } from './app/PreferencesProvider'
import { TrayPage } from './pages/TrayPage'
import { WelcomePage } from './pages/WelcomePage'
import { isTrayRoute } from './tray/bridge'

// Two pages so far, picked by path: /app/tray is the tray flyout
// (ADR-0036), everything else the welcome page. Phase 2 adds a router
// (pages under /app/...) as more legacy pages migrate.
export function App() {
  return (
    <PreferencesProvider>
      {isTrayRoute(window.location.pathname) ? <TrayPage /> : <WelcomePage />}
    </PreferencesProvider>
  )
}
