import { Lang } from '../i18n'
import { PageId } from '../site'
import { useLangRouting, useTheme } from '../hooks/usePreferences'
import Nav from '../components/ui/Nav'
import SkipLink from '../components/ui/SkipLink'
import Footer from '../components/ui/Footer'
import TmuxBar from '../components/ui/TmuxBar'
import LangSuggest from '../components/ui/LangSuggest'
import LegalDoc, { LegalDocContent } from '../components/sections/LegalDoc'

interface Props { page: PageId; lang: Lang; doc: Record<Lang, LegalDocContent> }

// Page shell shared by the legal documents: logo top-left (Nav), title,
// version + date, the document, footer, tmux bar.
export default function LegalApp({ page, lang, doc: docs }: Props) {
  const { theme, setTheme } = useTheme()
  const { suggest, dismiss } = useLangRouting(page, lang)
  const doc = docs[lang]

  return (
    <>
      <SkipLink lang={lang} />
      <Nav page={page} lang={lang} theme={theme} setTheme={setTheme} />
      <main id="main" tabIndex={-1}>
        <header className="guide-hero">
          <div className="container">
            <div className="section-label cmd-label">{doc.label}</div>
            <h1 className="section-title">{doc.title}</h1>
            <p className="section-sub">{doc.sub}</p>
          </div>
        </header>
        <LegalDoc doc={doc} />
      </main>
      <Footer lang={lang} page={page} />
      <TmuxBar
        lang={lang}
        session={doc.session}
        windows={doc.sections.map((s, n) => ({ n, name: s.tmux, id: s.id }))}
        showStatus={false}
      />
      <LangSuggest page={page} suggest={suggest} onDismiss={dismiss} />
    </>
  )
}
