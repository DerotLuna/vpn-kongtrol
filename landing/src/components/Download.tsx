import { OS, OS_LABELS } from '../hooks/useOS'
import { Lang } from '../i18n'
import { GITHUB_RELEASES, GITHUB_REPO } from '../links'
import CodeBlock from './CodeBlock'

interface Binary {
  filename: string
  archNote: string
  primary?: boolean
}

const BINARIES: Record<OS, Binary[]> = {
  windows: [
    { filename: 'kongtrol-windows-amd64.exe', archNote: 'x64 (64-bit)', primary: true },
  ],
  macos: [
    { filename: 'kongtrol-darwin-arm64', archNote: 'Apple Silicon (M1/M2/M3)', primary: true },
    { filename: 'kongtrol-darwin-amd64', archNote: 'Intel (x86_64)' },
  ],
  linux: [
    { filename: 'kongtrol-linux-amd64', archNote: 'x64 (64-bit)', primary: true },
    { filename: 'kongtrol-linux-arm64', archNote: 'ARM64 (Raspberry Pi 4+)' },
  ],
}

interface Props {
  os: OS
  setOS: (os: OS) => void
  lang: Lang
}

export default function Download({ os, setOS, lang }: Props) {
  const copy = lang === 'es'
    ? {
      section: '$ make install',
      title: 'Un binario. Cero dependencias.',
      sub: 'Los releases viven en GitHub: descarga el binario de tu plataforma y a operar.',
      release: 'Ir al release',
      beta: 'Beta:',
      betaText: 'Kongtrol está en pruebas activas — no todos los adaptadores ni escenarios están validados. Si algo falla, abre un issue en GitHub.',
      source: 'FUENTE',
      sourceScript: `# Requiere Go 1.22+ — desde Git Bash en Windows
$ git clone ${GITHUB_REPO}
$ cd kongtrol
$ make build-all-cli
# → binarios en build/dist/ para todas las plataformas`,
      yourSystem: 'TU SISTEMA',
    }
    : {
      section: '$ make install',
      title: 'One binary. Zero dependencies.',
      sub: 'Releases live on GitHub: grab the binary for your platform and you are operating.',
      release: 'Go to release',
      beta: 'Beta:',
      betaText: 'Kongtrol is under active testing — not every adapter and scenario is validated yet. If something breaks, open an issue on GitHub.',
      source: 'SOURCE',
      sourceScript: `# Requires Go 1.22+ — run from Git Bash on Windows
$ git clone ${GITHUB_REPO}
$ cd kongtrol
$ make build-all-cli
# -> binaries in build/dist/ for all platforms`,
      yourSystem: 'YOUR SYSTEM',
    }

  return (
    <section id="install" className="section">
      <div className="container">
        <div className="section-label cmd-label">{copy.section}</div>
        <h2 className="section-title">{copy.title}</h2>
        <p className="section-sub">{copy.sub}</p>

        <div className="os-tabs" style={{ marginTop: 32 }}>
          {(['windows', 'macos', 'linux'] as OS[]).map(o => (
            <button key={o} className={`os-tab${os === o ? ' active' : ''}`} onClick={() => setOS(o)}>
              {OS_LABELS[o]}
            </button>
          ))}
        </div>

        <div className="download-grid" style={{ marginTop: 20 }}>
          {(Object.entries(BINARIES) as [OS, Binary[]][]).map(([osKey, bins]) => (
            <div key={osKey} className={`download-card reveal${os === osKey ? ' current' : ''}`} data-current-label={copy.yourSystem}>
              <div className="download-os-name">{OS_LABELS[osKey]}</div>

              <div style={{ display: 'flex', flexDirection: 'column', gap: 10, flex: 1 }}>
                {bins.map(b => (
                  <div key={b.filename}>
                    <div className="download-filename">{b.filename}</div>
                    <div className="download-arch-note">{b.archNote}</div>
                  </div>
                ))}
              </div>

              <a href={GITHUB_RELEASES} className="btn-download" target="_blank" rel="noreferrer">
                <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
                  <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27s1.36.09 2 .27c1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8Z"/>
                </svg>
                {copy.release}
              </a>
            </div>
          ))}
        </div>

        <div className="callout warn" style={{ marginTop: 32 }}>
          <strong>{copy.beta}</strong> {copy.betaText}
        </div>

        <div className="source-install reveal">
          <div className="source-label">{copy.source}</div>
          <CodeBlock lang="bash">{copy.sourceScript}</CodeBlock>
        </div>
      </div>
    </section>
  )
}
