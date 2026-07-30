import { Lang } from '../i18n'
import { GITHUB_REPO } from '../links'

export const SITE_VERSION = 'v2.1.0'
export const SITE_UPDATED = '2026-07-15'

interface Props {
  lang: Lang
  page?: 'home' | 'guide'
}

export default function Footer({ lang }: Props) {
  const copy = lang === 'es'
    ? {
      tagline: 'Orquestación multi-VPN terminal-first. Un binario, ocho adaptadores, cero fugas.',
      facts: ['beta', '8 adaptadores', 'Win / macOS / Linux', 'ES / EN', 'MIT'],
      wink: 'no place like 127.0.0.1',
    }
    : {
      tagline: 'Terminal-first multi-VPN orchestration. One binary, eight adapters, zero leaks.',
      facts: ['beta', '8 adapters', 'Win / macOS / Linux', 'ES / EN', 'MIT'],
      wink: 'no place like 127.0.0.1',
    }

  return (
    <footer className="footer">
      <div className="container">
        <div className="footer-grid">
          <div className="footer-brand-col">
            <div className="footer-brand">
              <img src={`${import.meta.env.BASE_URL}logo.svg`} alt="Kongtrol" />
            </div>
            <p className="footer-tagline">{copy.tagline}</p>
            <ul className="footer-facts">
              {copy.facts.map(f => <li key={f}>{f}</li>)}
            </ul>
          </div>

          <div className="footer-term mono">
            <div>
              <span className="t-prompt">$ </span>
              <span className="t-cmd">kongtrol version</span>
            </div>
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
                github.com/vpn-kongtrol/kongtrol
              </a>
            </div>
          </div>
        </div>

        <div className="footer-meta">
          <span className="footer-wink mono">{copy.wink}</span>
        </div>
      </div>

      <div className="footer-giant" aria-hidden="true">KONGTROL</div>
    </footer>
  )
}
