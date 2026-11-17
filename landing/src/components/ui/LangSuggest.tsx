import { Lang } from '../../i18n'
import { PageId, pathFor } from '../../site'
import { chooseLang } from '../../hooks/usePreferences'

interface Props { page: PageId; suggest: Lang | null; onDismiss: () => void }

// Small, dismissible "view in your language" bar. Shown client-side only when
// the browser language differs from the page and the visitor never picked one.
export default function LangSuggest({ page, suggest, onDismiss }: Props) {
  if (!suggest) return null
  const copy = suggest === 'es'
    ? { text: 'Esta página está disponible en español.', go: 'Ver en español', close: 'Seguir en inglés' }
    : { text: 'This page is available in English.', go: 'View in English', close: 'Stay in Spanish' }
  return (
    <div className="lang-suggest" role="region" lang={suggest} aria-label={copy.text}>
      <span>{copy.text}</span>
      <a href={pathFor(page, suggest) + (typeof window !== 'undefined' ? window.location.hash : '')} onClick={() => chooseLang(suggest)}>{copy.go}</a>
      <button type="button" onClick={onDismiss} aria-label={copy.close} title={copy.close}>×</button>
    </div>
  )
}
