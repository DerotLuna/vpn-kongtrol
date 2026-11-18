import React from 'react'
import { hydrateRoot } from 'react-dom/client'
import App from './App'
import { pageLang } from '../../site'
import '../../index.css'

// The HTML already holds this page, prerendered by scripts/prerender.mjs:
// hydrate it instead of replacing it (react.dev/reference/react-dom/client/hydrateRoot).
hydrateRoot(
  document.getElementById('root')!,
  <React.StrictMode>
    <App lang={pageLang()} />
  </React.StrictMode>,
)
