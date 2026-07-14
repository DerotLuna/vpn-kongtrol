import { OS, OS_LABELS } from '../hooks/useOS'
import { Lang } from '../i18n'
import { Theme } from '../theme'

const OS_LIST: OS[] = ['windows', 'macos', 'linux']

interface Props {
  os: OS
  setOS: (os: OS) => void
  lang: Lang
  setLang: (lang: Lang) => void
  theme: Theme
  setTheme: (theme: Theme) => void
}

export default function Nav({ os, setOS, lang, setLang, theme, setTheme }: Props) {
  const labels = lang === 'es'
    ? { download: 'Descargar', guide: 'Guía', osTitle: 'Sistema operativo', langTitle: 'Idioma', themeTitle: 'Tema' }
    : { download: 'Download', guide: 'Guide', osTitle: 'Operating system', langTitle: 'Language', themeTitle: 'Theme' }

  return (
    <nav className="nav">
      <div className="nav-inner">
        <a href="#top" className="nav-logo">
          <img src="/logo-kong.png" alt="Kongtrol" />
          <span className="nav-logo-name">KONGTROL CLI</span>
        </a>

        <ul className="nav-links">
          <li><a href="#descargar">{labels.download}</a></li>
          <li><a href="#guia">{labels.guide}</a></li>
        </ul>

        <div className="nav-controls">
          <div className="nav-theme-switcher" title={labels.themeTitle}>
            {(['dark', 'light'] as Theme[]).map(t => (
              <button
                key={t}
                className={`nav-theme-btn${theme === t ? ' active' : ''}`}
                onClick={() => setTheme(t)}
              >
                {t === 'dark' ? (lang === 'es' ? 'Oscuro' : 'Dark') : (lang === 'es' ? 'Claro' : 'Light')}
              </button>
            ))}
          </div>

          <div className="nav-lang-switcher" title={labels.langTitle}>
            {(['es', 'en'] as Lang[]).map(l => (
              <button
                key={l}
                className={`nav-lang-btn${lang === l ? ' active' : ''}`}
                onClick={() => setLang(l)}
              >
                {l.toUpperCase()}
              </button>
            ))}
          </div>

          <div className="nav-os-switcher" title={labels.osTitle}>
            {OS_LIST.map(o => (
              <button
                key={o}
                className={`nav-os-btn${os === o ? ' active' : ''}`}
                onClick={() => setOS(o)}
              >
                {OS_LABELS[o]}
              </button>
            ))}
          </div>
        </div>
      </div>
    </nav>
  )
}
