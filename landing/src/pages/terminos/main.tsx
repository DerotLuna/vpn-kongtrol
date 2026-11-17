import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import { pageLang } from '../../site'
import '../../index.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App lang={pageLang()} />
  </React.StrictMode>,
)
