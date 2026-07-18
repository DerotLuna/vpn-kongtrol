import { useState } from 'react'
import { detectOS, OS } from './hooks/useOS'
import { usePreferences } from './hooks/usePreferences'
import { useKonami } from './hooks/useKonami'
import Nav from './components/Nav'
import Guide from './components/Guide'
import Footer from './components/Footer'
import TmuxBar, { TmuxWindow } from './components/TmuxBar'

// guide sections as tmux windows — short command-flavored names, same IDs the
// sidebar and IntersectionObserver already use
const GUIDE_WINDOWS: TmuxWindow[] = [
  { n: 0, name: 'start', id: 'quickstart' },
  { n: 1, name: 'prereqs', id: 'prereqs' },
  { n: 2, name: 'install', id: 'install' },
  { n: 3, name: 'init', id: 'wizard' },
  { n: 4, name: 'groups', id: 'groups' },
  { n: 5, name: 'policies', id: 'policies' },
  { n: 6, name: 'doctor', id: 'doctor' },
  { n: 7, name: 'up', id: 'connect' },
  { n: 8, name: 'dash', id: 'dashboard' },
  { n: 9, name: 'daily', id: 'daily' },
  { n: 10, name: 'debug', id: 'trouble' },
]

export default function GuideApp() {
  const [os, setOS] = useState<OS>(detectOS)
  const { lang, setLang, theme, setTheme } = usePreferences()
  useKonami()

  const head = lang === 'es'
    ? { label: '$ man kongtrol', title: 'Guía de configuración', sub: 'De cero a operar: instalación, wizard, políticas, doctor y primera conexión. 15–20 minutos, paso a paso por sistema operativo.' }
    : { label: '$ man kongtrol', title: 'Setup guide', sub: 'From zero to operating: install, wizard, policies, doctor, and first connect. 15–20 minutes, step by step per operating system.' }

  return (
    <>
      <Nav
        page="guide"
        lang={lang}
        setLang={setLang}
        theme={theme}
        setTheme={setTheme}
      />
      <main>
        <header className="guide-hero">
          <div className="container">
            <div className="section-label cmd-label">{head.label}</div>
            <h1 className="section-title">{head.title}</h1>
            <p className="section-sub">{head.sub}</p>
          </div>
        </header>
        <Guide os={os} setOS={setOS} lang={lang} />
      </main>
      <Footer lang={lang} page="guide" />
      <TmuxBar lang={lang} session="kongtrol:guia" windows={GUIDE_WINDOWS} showStatus={false} />
    </>
  )
}
