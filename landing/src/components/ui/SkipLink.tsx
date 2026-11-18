import { Lang } from '../../i18n'

// "Skip to content" — the first focusable element on every page, visible only
// while focused (WCAG 2.2 SC 2.4.1 Bypass Blocks). Targets <main id="main">.
export default function SkipLink({ lang }: { lang: Lang }) {
  return (
    <a className="skip-link" href="#main">
      {lang === 'es' ? 'Saltar al contenido' : 'Skip to content'}
    </a>
  )
}
