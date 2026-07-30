import { OS, OS_LABELS } from '../hooks/useOS'
import { Lang } from '../i18n'
import { GITHUB_RELEASES, GITHUB_REPO, releaseAsset } from '../links'
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
      sub: 'Descarga el binario de tu plataforma directamente y a operar.',
      download: 'Descargar',
      allReleases: 'Ver todos los releases en GitHub',
      beta: 'Beta:',
      betaText: 'Kongtrol está en pruebas activas — no todos los adaptadores ni escenarios están validados. Si algo falla, abre un issue en GitHub.',
      source: 'FUENTE',
      sourceScript: `# Requiere Go 1.22+ — desde Git Bash en Windows
$ git clone ${GITHUB_REPO}
$ cd kongtrol
$ make build-all-cli
# → binarios en build/dist/ para todas las plataformas`,
      yourSystem: 'TU SISTEMA',
      verifyTitle: 'Sobre los binarios no firmados',
      verifyText: 'Kongtrol es de código abierto y no paga por un certificado de firma de código (Windows/macOS lo exigen para evitar advertencias de SmartScreen/Gatekeeper). Es normal que el binario descargado dispare una advertencia — no significa que contenga malware. Cada release en GitHub incluye un checksums.txt (SHA256) para que verifiques que lo que descargaste coincide con lo que se compiló. Si prefieres no confiar en un binario prebuild, revisa el código y compílalo tú mismo (ver FUENTE abajo).',
      verifyScript: `# Verificar checksum (Windows PowerShell)
Get-FileHash kongtrol-windows-amd64.exe -Algorithm SHA256
# compara el resultado con la línea correspondiente en checksums.txt del release

# Verificar checksum (Linux/macOS)
sha256sum -c checksums.txt`,
    }
    : {
      section: '$ make install',
      title: 'One binary. Zero dependencies.',
      sub: 'Grab the binary for your platform directly and you are operating.',
      download: 'Download',
      allReleases: 'View all releases on GitHub',
      beta: 'Beta:',
      betaText: 'Kongtrol is under active testing — not every adapter and scenario is validated yet. If something breaks, open an issue on GitHub.',
      source: 'SOURCE',
      sourceScript: `# Requires Go 1.22+ — run from Git Bash on Windows
$ git clone ${GITHUB_REPO}
$ cd kongtrol
$ make build-all-cli
# -> binaries in build/dist/ for all platforms`,
      yourSystem: 'YOUR SYSTEM',
      verifyTitle: 'About unsigned binaries',
      verifyText: 'Kongtrol is open source and does not pay for a code-signing certificate (Windows/macOS require one to skip SmartScreen/Gatekeeper warnings). It is normal for the downloaded binary to trigger a warning — that does not mean it contains malware. Every GitHub release ships a checksums.txt (SHA256) so you can verify the download matches what was actually built. If you would rather not trust a prebuilt binary, review the source and build it yourself (see SOURCE below).',
      verifyScript: `# Verify checksum (Windows PowerShell)
Get-FileHash kongtrol-windows-amd64.exe -Algorithm SHA256
# compare the output against the matching line in the release's checksums.txt

# Verify checksum (Linux/macOS)
sha256sum -c checksums.txt`,
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
                  <div key={b.filename} style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12 }}>
                    <div>
                      <div className="download-filename">{b.filename}</div>
                      <div className="download-arch-note">{b.archNote}</div>
                    </div>
                    <a href={releaseAsset(b.filename)} className="btn-download" download>
                      <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
                        <path d="M8 1a.5.5 0 0 1 .5.5v7.793l2.646-2.647a.5.5 0 0 1 .708.708l-3.5 3.5a.5.5 0 0 1-.708 0l-3.5-3.5a.5.5 0 1 1 .708-.708L7.5 9.293V1.5A.5.5 0 0 1 8 1ZM2.5 13a.5.5 0 0 0 0 1h11a.5.5 0 0 0 0-1h-11Z"/>
                      </svg>
                      {copy.download}
                    </a>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>

        <div style={{ marginTop: 12 }}>
          <a href={GITHUB_RELEASES} target="_blank" rel="noreferrer" className="section-sub" style={{ textDecoration: 'underline' }}>
            {copy.allReleases}
          </a>
        </div>

        <div className="callout warn" style={{ marginTop: 32 }}>
          <strong>{copy.beta}</strong> {copy.betaText}
        </div>

        <div className="source-install reveal" style={{ marginTop: 20 }}>
          <div className="source-label">{copy.verifyTitle}</div>
          <p className="section-sub" style={{ marginTop: 8, marginBottom: 12 }}>{copy.verifyText}</p>
          <CodeBlock lang="bash">{copy.verifyScript}</CodeBlock>
        </div>

        <div className="source-install reveal">
          <div className="source-label">{copy.source}</div>
          <CodeBlock lang="bash">{copy.sourceScript}</CodeBlock>
        </div>
      </div>
    </section>
  )
}
