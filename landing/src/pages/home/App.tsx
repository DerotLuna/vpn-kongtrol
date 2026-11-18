import { useOS } from '../../hooks/useOS'
import { useLangRouting, useTheme } from '../../hooks/usePreferences'
import { Lang } from '../../i18n'
import LangSuggest from '../../components/ui/LangSuggest'
import { useKonami } from '../../hooks/useKonami'
import { useActiveSection } from '../../hooks/useActiveSection'
import Nav from '../../components/ui/Nav'
import SkipLink from '../../components/ui/SkipLink'
import { useReveal } from '../../hooks/useReveal'
import Hero from '../../components/sections/Hero'
import Start from '../../components/sections/Start'
import Providers from '../../components/sections/Providers'
import Features from '../../components/sections/Features'
import Security from '../../components/sections/Security'
import HowItWorks from '../../components/sections/HowItWorks'
import UnderTheHood from '../../components/sections/UnderTheHood'
import Download from '../../components/sections/Download'
import Footer from '../../components/ui/Footer'
import TmuxBar, { TmuxWindow } from '../../components/ui/TmuxBar'
import SectionJumpMenu from '../../components/ui/SectionJumpMenu'

// home sections as tmux windows — same IDs the page's own anchors use
const HOME_WINDOWS_ES: TmuxWindow[] = [
  { n: 0, name: 'motd', id: 'motd' },
  { n: 1, name: 'empezar', id: 'start' },
  { n: 2, name: 'pasos', id: 'steps' },
  { n: 3, name: 'vpns', id: 'vpns' },
  { n: 4, name: 'mapa', id: 'map' },
  { n: 5, name: 'check', id: 'check' },
  { n: 6, name: 'capó', id: 'hood' },
  { n: 7, name: 'instalar', id: 'install' },
]
const HOME_WINDOWS_EN: TmuxWindow[] = [
  { n: 0, name: 'motd', id: 'motd' },
  { n: 1, name: 'start', id: 'start' },
  { n: 2, name: 'steps', id: 'steps' },
  { n: 3, name: 'vpns', id: 'vpns' },
  { n: 4, name: 'map', id: 'map' },
  { n: 5, name: 'check', id: 'check' },
  { n: 6, name: 'hood', id: 'hood' },
  { n: 7, name: 'install', id: 'install' },
]

export default function App({ lang }: { lang: Lang }) {
  const [os] = useOS()
  const { theme, setTheme } = useTheme()
  const { suggest, dismiss } = useLangRouting('home', lang)
  useKonami()
  useReveal()

  const windows = lang === 'es' ? HOME_WINDOWS_ES : HOME_WINDOWS_EN
  // same tracking TmuxBar uses below — the tmux bar hides under 720px, so
  // this feeds the mobile-only jump menu that replaces it there
  const [activeSection, setActiveSection] = useActiveSection(windows.map(w => w.id), '-40% 0px -55% 0px')

  return (
    <>
      <SkipLink lang={lang} />
      <Nav
        page="home"
        lang={lang}
        theme={theme}
        setTheme={setTheme}
      />
      <main id="main" tabIndex={-1}>
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
          <Start lang={lang} />
          <HowItWorks lang={lang} />
          <Providers lang={lang} />
          <Features lang={lang} />
          <Security lang={lang} />
          <UnderTheHood lang={lang} />
          <Download lang={lang} />
        </div>
      </main>
      <Footer lang={lang} page="home" />
      <TmuxBar lang={lang} windows={windows} />
      <LangSuggest page="home" suggest={suggest} onDismiss={dismiss} />
    </>
  )
}
