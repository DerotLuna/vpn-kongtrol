import { usePreferences } from './hooks/usePreferences'
import Nav from './components/Nav'
import Terminos from './components/Terminos'
import Footer from './components/Footer'
import TmuxBar, { TmuxWindow } from './components/TmuxBar'

// terms sections as tmux windows — same ids the sidebar/IntersectionObserver
// in Terminos.tsx already use
const TERMS_WINDOWS_ES: TmuxWindow[] = [
  { n: 0, name: 'que-es', id: 'que-es' },
  { n: 1, name: 'garantia', id: 'garantia' },
  { n: 2, name: 'beta', id: 'beta' },
  { n: 3, name: 'responsabilidad', id: 'responsabilidad' },
  { n: 4, name: 'binarios', id: 'binarios' },
  { n: 5, name: 'privacidad', id: 'privacidad' },
  { n: 6, name: 'cambios', id: 'cambios' },
]
const TERMS_WINDOWS_EN: TmuxWindow[] = [
  { n: 0, name: 'what-is', id: 'que-es' },
  { n: 1, name: 'warranty', id: 'garantia' },
  { n: 2, name: 'beta', id: 'beta' },
  { n: 3, name: 'liability', id: 'responsabilidad' },
  { n: 4, name: 'binaries', id: 'binarios' },
  { n: 5, name: 'privacy', id: 'privacidad' },
  { n: 6, name: 'changes', id: 'cambios' },
]

export default function TerminosApp() {
  const { lang, setLang, theme, setTheme } = usePreferences()

  const head = lang === 'es'
    ? { label: '$ cat TERMS.md', title: 'Términos y condiciones', sub: 'Software de código abierto, sin garantía, uso bajo tu propia responsabilidad.' }
    : { label: '$ cat TERMS.md', title: 'Terms & conditions', sub: 'Open-source software, no warranty, use at your own responsibility.' }

  return (
    <>
      <Nav
        page="terminos"
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
        <Terminos lang={lang} />
      </main>
      <Footer lang={lang} page="terminos" />
      <TmuxBar lang={lang} session="kongtrol:terminos" windows={lang === 'es' ? TERMS_WINDOWS_ES : TERMS_WINDOWS_EN} showStatus={false} />
    </>
  )
}
