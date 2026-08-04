import { useState } from 'react'
import { detectOS, OS } from './hooks/useOS'
import { usePreferences } from './hooks/usePreferences'
import { useKonami } from './hooks/useKonami'
import { useActiveSection } from './hooks/useActiveSection'
import Nav from './components/Nav'
import Hero from './components/Hero'
import Features from './components/Features'
import Security from './components/Security'
import HowItWorks from './components/HowItWorks'
import Download from './components/Download'
import Footer from './components/Footer'
import TmuxBar, { TmuxWindow } from './components/TmuxBar'
import SectionJumpMenu from './components/SectionJumpMenu'

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

  const windows = lang === 'es' ? HOME_WINDOWS_ES : HOME_WINDOWS_EN
  // same tracking TmuxBar uses below — the tmux bar hides under 720px, so
  // this feeds the mobile-only jump menu that replaces it there
  const [activeSection, setActiveSection] = useActiveSection(windows.map(w => w.id), '-40% 0px -55% 0px')

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
        {/* wraps every section the jump menu can point to, so its sticky
            position has room to stay pinned under the header for the whole
            scroll range — a wrapper limited to just the menu itself has no
            scrollable height of its own for sticky to hold within */}
        <div className="home-jump-scope">
          <SectionJumpMenu
            className="home-jump"
            label={lang === 'es' ? 'Ir a' : 'Jump to'}
            value={activeSection}
            options={windows.map(w => ({ id: w.id, num: String(w.n), label: w.name }))}
            onChange={id => {
              setActiveSection(id)
              document.getElementById(id)?.scrollIntoView({ behavior: 'smooth' })
            }}
          />
          <Features lang={lang} />
          <Security lang={lang} />
          <HowItWorks lang={lang} />
          <Download os={os} setOS={setOS} lang={lang} />
        </div>
      </main>
      <Footer lang={lang} page="home" />
      <TmuxBar lang={lang} windows={windows} />
    </>
  )
}
