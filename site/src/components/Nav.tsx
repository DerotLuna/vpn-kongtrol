import { OS, OS_LABELS } from '../hooks/useOS'

const OS_LIST: OS[] = ['windows', 'macos', 'linux']

interface Props {
  os: OS
  setOS: (os: OS) => void
}

export default function Nav({ os, setOS }: Props) {
  return (
    <nav className="nav">
      <div className="nav-inner">
        <a href="#top" className="nav-logo">
          <img src="/logo-kong.png" alt="Kongtrol" />
          <span className="nav-logo-name">VPN KONGTROL</span>
        </a>

        <ul className="nav-links">
          <li><a href="#descargar">Descargar</a></li>
          <li><a href="#guia">Guía</a></li>
          <li><a href="https://github.com/vpn-kongtrol/kongtrol" target="_blank" rel="noreferrer">GitHub</a></li>
        </ul>

        <div className="nav-os-switcher" title="Sistema operativo">
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
    </nav>
  )
}
