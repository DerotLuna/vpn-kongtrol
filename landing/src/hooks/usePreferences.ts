import { useCallback, useEffect, useState } from 'react'
import { detectLang, Lang } from '../i18n'
import { PageId, pathFor } from '../site'

// Preference rule (same as the dashboard): follow the browser/OS until the
// visitor explicitly picks; an explicit pick is stored and wins from then on.
// Nothing is written to storage unless the visitor clicks something, so a
// stored value always means "the visitor chose this".
const LANG_KEY = 'kglang'

function read(key: string): string | null {
  try { return localStorage.getItem(key) } catch { return null }
}
function write(key: string, value: string) {
  try { localStorage.setItem(key, value) } catch { /* private mode: choice lasts this page only */ }
}

const storedLang = (): Lang | null => {
  const v = read(LANG_KEY)
  return v === 'es' || v === 'en' ? v : null
}

// Records an explicit language pick (language toggle, suggestion banner).
export const chooseLang = (lang: Lang) => write(LANG_KEY, lang)

// Language is decided by the URL. Client-side only, never for crawlers (they
// keep no storage): a visitor who explicitly picked the other language is sent
// to that URL; a visitor who never picked gets a dismissible suggestion when
// the browser language differs. No automatic redirect on a guess — Google:
// "Avoid automatically redirecting users from one language version of a site
// to a different language version" (managing-multi-regional-sites).
export function useLangRouting(page: PageId, lang: Lang) {
  const [suggest, setSuggest] = useState<Lang | null>(null)

  useEffect(() => {
    const chosen = storedLang()
    if (chosen && chosen !== lang) {
      window.location.replace(pathFor(page, chosen) + window.location.hash)
      return
    }
    if (!chosen && detectLang() !== lang) setSuggest(detectLang())
  }, [page, lang])

  const dismiss = useCallback(() => {
    chooseLang(lang) // "stay here" is an explicit pick too
    setSuggest(null)
  }, [lang])

  return { suggest, dismiss }
}
