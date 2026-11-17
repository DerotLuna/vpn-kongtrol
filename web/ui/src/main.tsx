// Kongtrol UI entry · v0.1.0 · 2026-11-17
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import faviconUrl from '@brand/logo-kong-32.png'
// Self-hosted fonts (assets/brand/fonts, shared with the dashboard and the
// landing): nothing is fetched from Google.
import '@brand/fonts/fonts.css'
import { App } from './App'
import { applyTheme, initialTheme } from './theme/theme'
import './theme/global.css'

// Set the theme before the first render to avoid a flash.
applyTheme(initialTheme())

const icon = document.createElement('link')
icon.rel = 'icon'
icon.type = 'image/png'
icon.href = faviconUrl
document.head.append(icon)

const root = document.getElementById('root')
if (!root) throw new Error('missing #root')
createRoot(root).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
