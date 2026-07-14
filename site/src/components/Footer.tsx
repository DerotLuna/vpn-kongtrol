import { Lang } from '../i18n'

interface Props {
  lang: Lang
}

export default function Footer({ lang }: Props) {
  const copy = lang === 'es'
    ? { download: 'Descargar', guide: 'Guía', tagline: 'Orquestación VPN terminal-first' }
    : { download: 'Download', guide: 'Guide', tagline: 'Terminal-first VPN orchestration' }

  return (
    <footer className="footer">
      <div className="container">
        <div className="footer-inner">
          <div className="footer-brand">
            <img src="/logo-kong.png" alt="Kongtrol" />
            <span className="footer-brand-name">KONGTROL CLI</span>
          </div>

          <ul className="footer-links">
            <li><a href="#descargar">{copy.download}</a></li>
            <li><a href="#guia">{copy.guide}</a></li>
          </ul>

          <span className="footer-copy">{copy.tagline} · MIT</span>
        </div>
      </div>
    </footer>
  )
}
