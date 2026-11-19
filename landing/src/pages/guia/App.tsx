import { Suspense, useMemo } from 'react'
import { useOS } from '../../hooks/useOS'
import { useLangRouting } from '../../hooks/usePreferences'
import { Lang } from '../../i18n'
import LangSuggest from '../../components/ui/LangSuggest'
import { useKonami } from '../../hooks/useKonami'
import Nav from '../../components/ui/Nav'
import SkipLink from '../../components/ui/SkipLink'
import Guide from '../../components/sections/Guide'
import Footer from '../../components/ui/Footer'
import TmuxBar, { TmuxWindow } from '../../components/ui/TmuxBar'

// guide sections as tmux windows — short command-flavored names, same IDs the
// sidebar and IntersectionObserver use (content/guideSections.ts). `init`,
// `up` and `doctor` are literal kongtrol subcommands, so they stay in both
// languages.
const GUIDE_WINDOWS_ES: TmuxWindow[] = [
  { n: 0, name: 'inicio', id: 'quickstart' },
  { n: 1, name: 'antes', id: 'prereqs' },
  { n: 2, name: 'instalar', id: 'install' },
  { n: 3, name: 'init', id: 'wizard' },
  { n: 4, name: 'up', id: 'connect' },
  { n: 5, name: 'reglas', id: 'policies' },
  { n: 6, name: 'doctor', id: 'doctor' },
  { n: 7, name: 'panel', id: 'dashboard' },
  { n: 8, name: 'diario', id: 'daily' },
  { n: 9, name: 'debug', id: 'trouble' },
  { n: 10, name: 'update', id: 'update' },
]
const GUIDE_WINDOWS_EN: TmuxWindow[] = [
  { n: 0, name: 'start', id: 'quickstart' },
  { n: 1, name: 'before', id: 'prereqs' },
  { n: 2, name: 'install', id: 'install' },
  { n: 3, name: 'init', id: 'wizard' },
  { n: 4, name: 'up', id: 'connect' },
  { n: 5, name: 'rules', id: 'policies' },
  { n: 6, name: 'doctor', id: 'doctor' },
  { n: 7, name: 'panel', id: 'dashboard' },
  { n: 8, name: 'daily', id: 'daily' },
  { n: 9, name: 'debug', id: 'trouble' },
  { n: 10, name: 'update', id: 'update' },
]

export default function GuideApp({ lang }: { lang: Lang }) {
  const [os, setOS] = useOS()
  const { suggest, dismiss } = useLangRouting('guia', lang)
  useKonami()
  // footer hydrates in its own later task; memoized so a re-render before
  // that keeps its server HTML (ADR-0031, same as home)
  const footer = useMemo(() => <Suspense><Footer lang={lang} page="guia" /></Suspense>, [lang])

  const head = lang === 'es'
    ? { label: '$ man kongtrol', title: 'Guía: instala y conecta', sub: 'Instala Kongtrol y conecta tu primera VPN en unos minutos, sin terminal. Cada paso dice qué hacer y qué vas a ver; si prefieres la terminal, también trae el comando.' }
    : { label: '$ man kongtrol', title: 'Guide: install and connect', sub: 'Install Kongtrol and connect your first VPN in a few minutes, no terminal needed. Each step says what to do and what you will see; if you prefer the terminal, it has the command too.' }

  return (
    <>
      <SkipLink lang={lang} />
      <Nav
        page="guia"
        lang={lang}
      />
      <main id="main" tabIndex={-1}>
        <header className="guide-hero">
          <div className="container">
            <div className="section-label cmd-label">{head.label}</div>
            <h1 className="section-title">{head.title}</h1>
            <p className="section-sub">{head.sub}</p>
          </div>
        </header>
        <Guide os={os} setOS={setOS} lang={lang} />
      </main>
      {footer}
      <TmuxBar lang={lang} session="kongtrol:guia" windows={lang === 'es' ? GUIDE_WINDOWS_ES : GUIDE_WINDOWS_EN} showStatus={false} />
      <LangSuggest page="guia" suggest={suggest} onDismiss={dismiss} />
    </>
  )
}
