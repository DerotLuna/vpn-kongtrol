// Closing call to action, right before the footer — version 1.0.0 · 2026-11-18.
// One reassurance line (each claim per PRODUCT.md: local only, free and open
// source, no account) and the same download entry point as the hero and
// "Para todos": on a computer it scrolls to #install (ADR-0027); on a phone it
// becomes "send to my computer". Reused, not reimplemented.
import { Lang } from '../../i18n'
import { useDetectedOS, useIsMobile } from '../../hooks/useVisitor'
import { downloadHref, downloadLabel, DownloadIcon } from '../ui/downloadCta'
import SendToComputer from '../ui/SendToComputer'
import './finalcta.css'

interface Props {
  lang: Lang
}

export default function FinalCta({ lang }: Props) {
  const detectedOS = useDetectedOS()
  const mobile = useIsMobile()
  const t = lang === 'es'
    ? {
      line: 'Local, gratis y sin cuenta.',
      sub: 'Kongtrol funciona en tu computador; tus datos se quedan ahí.',
    }
    : {
      line: 'Local, free, no account.',
      sub: 'Kongtrol runs on your computer; your data stays there.',
    }

  return (
    <section className="final-cta" aria-labelledby="final-cta-line">
      <div className="container">
        <div className="final-cta-inner reveal">
          <div className="final-cta-text">
            <h2 id="final-cta-line" className="final-cta-line">{t.line}</h2>
            <p className="final-cta-sub">{t.sub}</p>
          </div>
          {mobile ? (
            <SendToComputer lang={lang} className="final-cta-stc" />
          ) : (
            <a href={downloadHref('home', lang)} className="btn-download">
              <DownloadIcon size={14} />
              {downloadLabel(lang, detectedOS)}
            </a>
          )}
        </div>
      </div>
    </section>
  )
}
