// Kongtrol UI entry · v0.3.0 · 2026-11-19
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
// Self-hosted fonts (assets/brand/fonts, shared with the dashboard and the
// landing): nothing is fetched from Google.
import '@brand/fonts/fonts.css'
import { App } from './App'
import { startAuth } from './api/auth'
import { applyTheme, initialTheme } from './theme/theme'
import './theme/global.css'
import './theme/kit.css'
import './theme/kit-controls.css'
import './theme/shell.css'

// Set the theme before the first render to avoid a flash.
applyTheme(initialTheme())
// Take a one-time `#k=` bootstrap code out of the URL before anything else
// runs (api/auth.ts); requests wait for the session exchange.
void startAuth()

const root = document.getElementById('root')
if (!root) throw new Error('missing #root')
createRoot(root).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
