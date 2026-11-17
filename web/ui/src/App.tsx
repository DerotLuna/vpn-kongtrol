import { PreferencesProvider } from './app/PreferencesProvider'
import { WelcomePage } from './pages/WelcomePage'

// Phase 1 has one page. Phase 2 adds a router (pages under /app/...) as more
// legacy pages migrate.
export function App() {
  return (
    <PreferencesProvider>
      <WelcomePage />
    </PreferencesProvider>
  )
}
