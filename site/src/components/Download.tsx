import { OS, OS_LABELS } from '../hooks/useOS'
import CodeBlock from './CodeBlock'

// Binaries are served via /api/download — requires a valid key.
// Files live in _binaries/ (private, not in public/).
const BASE = '/api/download'

interface Binary {
  filename: string
  archNote: string
  primary?: boolean  // shown as the main download for this OS
}

const BINARIES: Record<OS, Binary[]> = {
  windows: [
    { filename: 'kongtrol-windows-amd64.exe', archNote: 'x64 (64-bit)', primary: true },
  ],
  macos: [
    { filename: 'kongtrol-darwin-arm64',  archNote: 'Apple Silicon (M1/M2/M3)', primary: true },
    { filename: 'kongtrol-darwin-amd64',  archNote: 'Intel (x86_64)' },
  ],
  linux: [
    { filename: 'kongtrol-linux-amd64',  archNote: 'x64 (64-bit)',  primary: true },
    { filename: 'kongtrol-linux-arm64',  archNote: 'ARM64 (Raspberry Pi 4+, servidores ARM)' },
  ],
}

interface Props {
  os: OS
  downloadKey: string | null
  onRequestKey: (filename: string) => void
}

export default function Download({ os, downloadKey, onRequestKey }: Props) {
  const handleClick = (filename: string, e: React.MouseEvent) => {
    if (!downloadKey) {
      e.preventDefault()
      onRequestKey(filename)
    }
    // Si hay clave, el href ya tiene ?key=... y el browser descarga directo
  }

  const url = (filename: string) =>
    downloadKey
      ? `${BASE}?file=${filename}&token=${encodeURIComponent(downloadKey)}`
      : '#'

  return (
    <section id="descargar" className="section">
      <div className="container">
        <div className="section-label">Descargar</div>
        <h2 className="section-title">Elige tu plataforma</h2>
        <p className="section-sub">
          Binario único, sin dependencias de runtime.
          Requiere los clientes VPN instalados por separado.
        </p>

        <div className="download-grid">
          {(Object.entries(BINARIES) as [OS, Binary[]][]).map(([osKey, bins]) => (
            <div key={osKey} className={`download-card${os === osKey ? ' current' : ''}`}>
              <div className="download-os-name">{OS_LABELS[osKey]}</div>

              <div style={{ display: 'flex', flexDirection: 'column', gap: 10, flex: 1 }}>
                {bins.map(b => (
                  <div key={b.filename}>
                    <div className="download-filename">{b.filename}</div>
                    <div className="download-arch-note">{b.archNote}</div>
                    <a
                      href={url(b.filename)}
                      className="btn-download"
                      onClick={e => handleClick(b.filename, e)}
                      style={b.primary ? {} : {
                        marginTop: 8,
                        background: 'transparent',
                        border: '1px solid var(--border-2)',
                        color: 'var(--text-muted)',
                        fontSize: '0.78rem',
                      }}
                    >
                      <svg width="14" height="14" viewBox="0 0 15 15" fill="none">
                        <path d="M7.5 1v9M4 7l3.5 3.5L11 7M2 13h11" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
                      </svg>
                      {b.primary ? 'Descargar' : `Descargar (${b.archNote})`}
                    </a>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>

        <div className="callout" style={{ marginTop: 32 }}>
          <strong>Verificar integridad:</strong> después de descargar, compara el SHA256
          con el archivo <code className="mono">checksums.txt</code>.{' '}
          <a
            href={url('checksums.txt')}
            onClick={e => handleClick('checksums.txt', e)}
          >
            Ver checksums
          </a>
        </div>

        <div className="source-install">
          <div className="source-label">FUENTE</div>
          <CodeBlock lang="bash">{`# Requiere Go 1.22+ — desde Git Bash en Windows
$ git clone https://github.com/vpn-kongtrol/kongtrol
$ cd vpn-kongtrol
$ make build-all-cli
# → binarios en build/dist/ para todas las plataformas`}</CodeBlock>
        </div>
      </div>
    </section>
  )
}
