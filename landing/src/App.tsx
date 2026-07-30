import { useState } from 'react'
import { detectOS, OS } from './hooks/useOS'
import { usePreferences } from './hooks/usePreferences'
import { useKonami } from './hooks/useKonami'
import Nav from './components/Nav'
import Hero from './components/Hero'
import Features from './components/Features'
import Security from './components/Security'
import HowItWorks from './components/HowItWorks'
import Download from './components/Download'
import Footer from './components/Footer'
import TmuxBar, { TmuxWindow } from './components/TmuxBar'

// home sections as tmux windows — same IDs the page's own anchors use
const HOME_WINDOWS_ES: TmuxWindow[] = [
  { n: 0, name: 'motd', id: 'motd' },
  { n: 1, name: 'mapa', id: 'map' },
  { n: 2, name: 'check', id: 'check' },
  { n: 3, name: 'init', id: 'init' },
  { n: 4, name: 'instalar', id: 'install' },
]
const HOME_WINDOWS_EN: TmuxWindow[] = [
  { n: 0, name: 'motd', id: 'motd' },
  { n: 1, name: 'map', id: 'map' },
  { n: 2, name: 'check', id: 'check' },
  { n: 3, name: 'init', id: 'init' },
  { n: 4, name: 'install', id: 'install' },
]

export default function App() {
  const [os, setOS] = useState<OS>(detectOS)
  const { lang, setLang, theme, setTheme } = usePreferences()
  useKonami()

  return (
    <>
      <Nav
        page="home"
        lang={lang}
        setLang={setLang}
        theme={theme}
        setTheme={setTheme}
      />
      <main>
        <Hero os={os} lang={lang} />
        <Features lang={lang} />
        <Security lang={lang} />
        <HowItWorks lang={lang} />
        <Download os={os} setOS={setOS} lang={lang} />
      </main>
      <Footer lang={lang} page="home" />
      <TmuxBar lang={lang} windows={lang === 'es' ? HOME_WINDOWS_ES : HOME_WINDOWS_EN} />
    </>
  )
}
