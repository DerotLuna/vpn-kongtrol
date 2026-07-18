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
import TmuxBar from './components/TmuxBar'

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
      <TmuxBar lang={lang} />
    </>
  )
}
