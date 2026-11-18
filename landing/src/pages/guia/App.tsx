import { Suspense, useMemo } from 'react'
import { useOS } from '../../hooks/useOS'
import { useLangRouting, useTheme } from '../../hooks/usePreferences'
import { Lang } from '../../i18n'
import LangSuggest from '../../components/ui/LangSuggest'
import { useKonami } from '../../hooks/useKonami'
import Nav from '../../components/ui/Nav'
import SkipLink from '../../components/ui/SkipLink'
import Guide from '../../components/sections/Guide'
import Footer from '../../components/ui/Footer'
import TmuxBar, { TmuxWindow } from '../../components/ui/TmuxBar'

// guide sections as tmux windows — short command-flavored names, same IDs the
// sidebar and IntersectionObserver already use. `init`, `up`, `doctor` are
// literal kongtrol CLI subcommands, so they stay unchanged in both languages.
const GUIDE_WINDOWS_ES: TmuxWindow[] = [
  { n: 0, name: 'inicio', id: 'quickstart' },
  { n: 1, name: 'prereqs', id: 'prereqs' },
  { n: 2, name: 'instalar', id: 'install' },
  { n: 3, name: 'init', id: 'wizard' },
  { n: 4, name: 'grupos', id: 'groups' },
  { n: 5, name: 'politicas', id: 'policies' },
  { n: 6, name: 'doctor', id: 'doctor' },
  { n: 7, name: 'up', id: 'connect' },
  { n: 8, name: 'panel', id: 'dashboard' },
  { n: 9, name: 'diario', id: 'daily' },
  { n: 10, name: 'debug', id: 'trouble' },
]
const GUIDE_WINDOWS_EN: TmuxWindow[] = [
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

export default function GuideApp({ lang }: { lang: Lang }) {
  const [os, setOS] = useOS()
  const { theme, setTheme } = useTheme()
  const { suggest, dismiss } = useLangRouting('guia', lang)
  useKonami()
  // footer hydrates in its own later task; memoized so a re-render before
  // that keeps its server HTML (ADR-0031, same as home)
  const footer = useMemo(() => <Suspense><Footer lang={lang} page="guia" /></Suspense>, [lang])

  const head = lang === 'es'
    ? { label: '$ man kongtrol', title: 'Guía de configuración', sub: 'De cero a operar: instalación, wizard, políticas, doctor y primera conexión. 15–20 minutos, paso a paso por sistema operativo.' }
    : { label: '$ man kongtrol', title: 'Setup guide', sub: 'From zero to operating: install, wizard, policies, doctor, and first connect. 15–20 minutes, step by step per operating system.' }

  return (
    <>
      <SkipLink lang={lang} />
      <Nav
        page="guia"
        lang={lang}
        theme={theme}
        setTheme={setTheme}
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
