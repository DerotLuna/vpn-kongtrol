import { useDetectedOS, useIsMobile } from '../../hooks/useVisitor'
import { Lang } from '../../i18n'
import { GITHUB_RELEASES, GITHUB_REPO } from '../../links'
import { PageId, pathFor, SITE_UPDATED, SITE_VERSION } from '../../site'
import { downloadHref, FOOTER_ID, PRIMARY_COPY, PrimaryDownload } from './downloadCta'
import SendToComputer from './SendToComputer'
import BackToTop from './BackToTop'

interface Props {
  lang: Lang
  page?: PageId
}

// Full-screen footer (2026-11-19, ADR-0060): exactly one screen above the
// tmux bar on laptops — the large line-art mark with the tagline, the primary
// download (same component as the hero and the download section), the site
// and legal links (GitHub only once: the `git clone` line), the version line,
// then the facts row, and the giant wordmark alone at the bottom (ADR-0066). The header hides while it is on screen (Nav.tsx, FOOTER_ID); the round
// back-to-top button (BackToTop) is fixed bottom-right on every page.
export default function Footer({ lang, page = 'home' }: Props) {
  const os = useDetectedOS()
  const mobile = useIsMobile()
  const copy = lang === 'es'
    ? {
      claim: 'Todas tus VPN bajo Kongtrol',
      tagline: 'Panel web y CLI completa. Local, gratis y de código abierto.',
      facts: ['beta', '15 VPN', 'Win / macOS / Linux', 'ES / EN', 'Apache-2.0'],
      wink: 'no place like 127.0.0.1',
      madeBy: ['Hecho con', 'por'],
      madeByHeart: 'cariño',
      latest: 'último release',
      siteTitle: 'Kongtrol',
      site: [['Inicio', pathFor('home', 'es')], ['Guía', pathFor('guia', 'es')], ['Descargas', downloadHref(page, 'es')]] as [string, string][],
      legalTitle: 'Legal',
      legal: [['privacidad', 'Privacidad'], ['seguridad', 'Seguridad'], ['terminos', 'Términos']] as [PageId, string][],
      license: 'Licencia (Apache-2.0)',
    }
    : {
      claim: 'All your VPNs under Kongtrol',
      tagline: 'Web panel and a full CLI. Local, free and open source.',
      facts: ['beta', '15 VPNs', 'Win / macOS / Linux', 'ES / EN', 'Apache-2.0'],
      wink: 'no place like 127.0.0.1',
      madeBy: ['Made with', 'by'],
      madeByHeart: 'love',
      latest: 'latest release',
      siteTitle: 'Kongtrol',
      site: [['Home', pathFor('home', 'en')], ['Guide', pathFor('guia', 'en')], ['Downloads', downloadHref(page, 'en')]] as [string, string][],
      legalTitle: 'Legal',
      legal: [['privacidad', 'Privacy'], ['seguridad', 'Security'], ['terminos', 'Terms']] as [PageId, string][],
      license: 'License (Apache-2.0)',
    }

  return (
    <footer id={FOOTER_ID} className="footer">
      <div className="container footer-inner">
        <div className="footer-hero">
          <img
            className="footer-mark logo-mark"
            src={`${import.meta.env.BASE_URL}logo-kong.svg`}
            alt="Kongtrol"
            width={176}
            height={176}
            loading="lazy"
            decoding="async"
          />
          <div className="footer-hero-text">
            <p className="footer-claim">{copy.claim}</p>
            <p className="footer-tagline">{copy.tagline}</p>
            <div className="footer-cta">
              {mobile ? <SendToComputer lang={lang} /> : <PrimaryDownload lang={lang} os={os} page={page} />}
              <a className="footer-guide-link" href={pathFor('guia', lang)}>{PRIMARY_COPY[lang].guide}</a>
            </div>
          </div>
        </div>

        <div className="footer-grid">
          <nav className="footer-col" aria-label={copy.siteTitle}>
            <p className="footer-col-title">{copy.siteTitle}</p>
            <ul>
              {copy.site.map(([label, href]) => <li key={label}><a href={href} className="footer-legal-link">{label}</a></li>)}
            </ul>
          </nav>
          <nav className="footer-col" aria-label={copy.legalTitle}>
            <p className="footer-col-title">{copy.legalTitle}</p>
            <ul>
              {copy.legal.map(([id, label]) => (
                <li key={id}><a href={pathFor(id, lang)} className="footer-legal-link">{label}</a></li>
              ))}
              <li><a href={`${GITHUB_REPO}/blob/main/LICENSE`} className="footer-legal-link" target="_blank" rel="noreferrer">{copy.license}</a></li>
            </ul>
          </nav>

          <div className="footer-term mono">
            <div>
              <span className="t-prompt">$ </span>
              <span className="t-cmd">kongtrol version</span>
            </div>
            {/* the latest stable release, baked at build time (vite.config.ts) */}
            {__LATEST_RELEASE__ && (
              <div className="footer-term-out">
                <a href={GITHUB_RELEASES} target="_blank" rel="noreferrer" className="footer-gh-link">
                  kongtrol <span className="footer-version">{__LATEST_RELEASE__}</span> · {copy.latest}
                </a>
              </div>
            )}
            <div className="footer-term-out">
              kongtrol-site <span className="footer-version">{SITE_VERSION}</span> · {SITE_UPDATED} · <span className="t-dim">exit 0</span>
            </div>
            <div className="footer-term-cmd">
              <span className="t-prompt">$ </span>
              <span className="t-cmd">git clone</span>
            </div>
            <div className="footer-term-out">
              <a href={GITHUB_REPO} target="_blank" rel="noreferrer" className="footer-gh-link">
                <svg width="13" height="13" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
                  <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27s1.36.09 2 .27c1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8Z"/>
                </svg>
                github.com/DerotLuna/vpn-kongtrol
              </a>
            </div>
          </div>
        </div>

        <div className="footer-meta">
          <ul className="footer-facts">
            {copy.facts.map(f => <li key={f}>{f}</li>)}
          </ul>
          {/* the author (ADR-0081): one person, not a company */}
          <span className="footer-made mono">
            {copy.madeBy[0]}{' '}
            <svg className="footer-heart" width="12" height="12" viewBox="0 0 16 16" role="img" aria-label={copy.madeByHeart}>
              <path fill="currentColor" d="M8 14.2 6.9 13.2C2.9 9.6.3 7.3.3 4.4.3 2.1 2.1.3 4.4.3c1.3 0 2.6.6 3.6 1.6C9 .9 10.3.3 11.6.3c2.3 0 4.1 1.8 4.1 4.1 0 2.9-2.6 5.2-6.6 8.8L8 14.2Z" />
            </svg>{' '}
            {copy.madeBy[1]}{' '}
            <a href="https://github.com/DerotLuna" target="_blank" rel="noreferrer" className="footer-gh-link">Rubén Luna</a>
          </span>
          <span className="footer-wink mono">{copy.wink}</span>
        </div>
      </div>

      <div className="footer-giant" aria-hidden="true">KONGTROL</div>
      <BackToTop lang={lang} />
    </footer>
  )
}
