import { OS, OS_LABELS } from '../../hooks/useOS'
import { useDetectedOS, useIsMobile } from '../../hooks/useVisitor'
import { Lang } from '../../i18n'
import { GITHUB_RELEASES, GITHUB_REPO, INSTALL_CMD, RELEASE_ASSETS, ReleaseAssetKey, releaseAsset } from '../../links'
import CodeBlock from '../ui/CodeBlock'
import { DOWNLOAD_SECTION_ID, DownloadIcon } from '../ui/downloadCta'
import SendToComputer from '../ui/SendToComputer'
import { pathFor } from '../../site'
import './download.css'

const ORDER: OS[] = ['windows', 'macos', 'linux']

interface Variant {
  asset: ReleaseAssetKey
  /** friendly name: what the visitor recognises about their computer */
  name: Record<Lang, string>
  /** short tag for the button, when one OS has several files */
  tag: Record<Lang, string>
}

// One OS → its files, recommended first. Filenames come from links.ts.
const VARIANTS: Record<OS, Variant[]> = {
  windows: [
    { asset: 'windows-amd64', name: { es: 'Windows 10/11 · 64-bit', en: 'Windows 10/11 · 64-bit' }, tag: { es: '64-bit', en: '64-bit' } },
    { asset: 'windows-arm64', name: { es: 'Windows en ARM · 64-bit', en: 'Windows on ARM · 64-bit' }, tag: { es: 'Windows en ARM', en: 'Windows on ARM' } },
  ],
  macos: [
    { asset: 'macos-arm64', name: { es: 'Mac con Apple Silicon (M1 o posterior)', en: 'Mac with Apple Silicon (M1 or later)' }, tag: { es: 'Apple Silicon', en: 'Apple Silicon' } },
    { asset: 'macos-amd64', name: { es: 'Mac con procesador Intel', en: 'Mac with an Intel processor' }, tag: { es: 'Intel', en: 'Intel' } },
  ],
  linux: [
    { asset: 'linux-amd64', name: { es: 'Linux · 64-bit (x86_64)', en: 'Linux · 64-bit (x86_64)' }, tag: { es: '64-bit', en: '64-bit' } },
    { asset: 'linux-arm64', name: { es: 'Linux en ARM · 64-bit (p. ej. Raspberry Pi 4)', en: 'Linux on ARM · 64-bit (e.g. Raspberry Pi 4)' }, tag: { es: 'Linux en ARM', en: 'Linux on ARM' } },
  ],
}

// Featured-card title per OS (macOS covers both chips: the browser can't tell them apart).
const FEATURED_TITLE: Record<OS, Record<Lang, string>> = {
  windows: { es: 'Windows 10/11 · 64-bit', en: 'Windows 10/11 · 64-bit' },
  macos: { es: 'macOS · Apple Silicon o Intel', en: 'macOS · Apple Silicon or Intel' },
  linux: { es: 'Linux · 64-bit', en: 'Linux · 64-bit' },
}

interface Props {
  lang: Lang
}

// The filename never wraps mid-word: one line, ellipsis if the card is too
// narrow, full name in the tooltip.
function FileName({ asset }: { asset: ReleaseAssetKey }) {
  return <code className="dl-file" title={RELEASE_ASSETS[asset]}>{RELEASE_ASSETS[asset]}</code>
}

export default function Download({ lang }: Props) {
  const detected = useDetectedOS()
  const mobile = useIsMobile()
  // prerender (OS unknown) and phones: no recommendation, every system equal
  const featured = mobile ? null : detected
  const copy = lang === 'es'
    ? {
      section: '$ make install',
      title: 'Descárgalo gratis.',
      sub: 'Windows, macOS y Linux. Local, gratis y de código abierto. Sin cuenta.',
      download: 'Descargar',
      downloadFor: { windows: 'Descargar para Windows', macos: 'Descargar para Mac', linux: 'Descargar para Linux' } as Record<OS, string>,
      recommended: 'Recomendado para tu equipo',
      others: 'Otros sistemas',
      choose: 'Elige tu sistema',
      macWhich: '¿Apple Silicon o Intel? Tu navegador no puede saberlo. En tu Mac abre el menú Apple → Acerca de este Mac: si dice «Chip», es Apple Silicon; si dice «Procesador» Intel, es Intel.',
      allReleases: 'Ver todos los releases en GitHub',
      keepNote: <>Si desinstalas, tus datos se quedan (salvo que pidas borrarlos). El aviso de versión nueva es
        opcional y viene apagado. Ver cómo <a href={`${pathFor('guia', 'es')}#update`}>actualizar</a> y{' '}
        <a href={`${pathFor('guia', 'es')}#uninstall`}>desinstalar</a>.</>,
      terminalTitle: 'INSTALAR POR TERMINAL',
      terminalText: 'También puedes instalar kongtrol directo desde la terminal: el script detecta tu sistema operativo y arquitectura, descarga el binario correcto de la última release, verifica su checksum SHA256, y lo deja listo en el PATH.',
      terminalScript: `# Linux / macOS
$ ${INSTALL_CMD.unix}

# Windows (PowerShell)
$ ${INSTALL_CMD.windows}`,
      beta: 'Beta pública.',
      betaText: 'Todavía no está probado con cada VPN en cada sistema. Si algo no va bien,',
      betaLink: 'cuéntanoslo en GitHub',
      afterTitle: 'Después de descargar',
      winTitle: 'Si Windows lo frena con un aviso azul (SmartScreen)',
      winSteps: [
        <>Pulsa <strong>Más información</strong>.</>,
        <>Pulsa <strong>Ejecutar de todos modos</strong>.</>,
      ],
      winWhy: 'Sale porque el archivo es nuevo y no está firmado con un certificado de pago, no porque tenga un virus.',
      macTitle: 'Si tu Mac dice que no puede abrir kongtrol',
      macSteps: [
        <>Cierra el aviso. No lo muevas a la papelera.</>,
        <>Abre el menú Apple → <strong>Ajustes del Sistema</strong> → <strong>Privacidad y seguridad</strong>.</>,
        <>Baja hasta <strong>Seguridad</strong> y pulsa <strong>Abrir igualmente</strong>. El botón aparece durante una hora después de intentar abrirlo.</>,
        <>Escribe la contraseña de tu Mac.</>,
      ],
      macCurl: 'Si instalas con el comando de terminal de más abajo (curl), este aviso no sale.',
      adminTitle: 'Un permiso de administrador, una sola vez',
      adminWhat: 'Al configurar Kongtrol se instala un ayudante para que pueda cambiar rutas y DNS. Nada sale de tu equipo. Después no vuelve a pedir permiso.',
      adminSee: [
        <><strong>Windows:</strong> una ventana de Control de cuentas de usuario te pregunta si permites los cambios. Pulsa <strong>Sí</strong>.</>,
        <><strong>Mac:</strong> te pide la contraseña de tu Mac.</>,
        <><strong>Linux:</strong> te pide tu contraseña (sudo).</>,
      ],
      adminTech: 'Detalle técnico:',
      source: 'FUENTE',
      sourceScript: `# Requiere Go 1.27+ — desde Git Bash en Windows
$ git clone ${GITHUB_REPO}
$ cd vpn-kongtrol
$ make build-all-cli
# → binarios en build/dist/ para todas las plataformas`,
      verifyTitle: 'Sobre los binarios no firmados',
      verifyText: 'Kongtrol es de código abierto y no paga por un certificado de firma de código (Windows/macOS lo exigen para evitar advertencias de SmartScreen/Gatekeeper). Es normal que el binario descargado dispare una advertencia — no significa que contenga malware. Cada release en GitHub incluye un checksums.txt (SHA256) para que verifiques que lo que descargaste coincide con lo que se compiló. Si prefieres no confiar en un binario prebuild, revisa el código y compílalo tú mismo (ver FUENTE abajo).',
      verifyScript: `# Verificar checksum (Windows PowerShell)
Get-FileHash ${RELEASE_ASSETS['windows-amd64']} -Algorithm SHA256
# compara el resultado con la línea correspondiente en checksums.txt del release

# Verificar checksum (Linux)
sha256sum -c --ignore-missing checksums.txt

# Verificar checksum (macOS)
shasum -a 256 -c --ignore-missing checksums.txt`,
    }
    : {
      section: '$ make install',
      title: 'Download it free.',
      sub: 'Windows, macOS and Linux. Local, free and open source. No account.',
      download: 'Download',
      downloadFor: { windows: 'Download for Windows', macos: 'Download for Mac', linux: 'Download for Linux' } as Record<OS, string>,
      recommended: 'Recommended for your computer',
      others: 'Other systems',
      choose: 'Choose your system',
      macWhich: "Apple Silicon or Intel? Your browser can't tell. On your Mac open the Apple menu → About This Mac: if it says \"Chip\", it's Apple Silicon; if it says \"Processor\" with Intel, it's Intel.",
      allReleases: 'View all releases on GitHub',
      keepNote: <>If you uninstall, your data stays (unless you ask to delete it). The new version notice is
        optional and off by default. See how to <a href={`${pathFor('guia', 'en')}#update`}>update</a> and{' '}
        <a href={`${pathFor('guia', 'en')}#uninstall`}>uninstall</a>.</>,
      terminalTitle: 'INSTALL FROM TERMINAL',
      terminalText: 'You can also install kongtrol straight from the terminal: the script detects your OS and architecture, downloads the correct binary from the latest release, verifies its SHA256 checksum, and drops it on your PATH.',
      terminalScript: `# Linux / macOS
$ ${INSTALL_CMD.unix}

# Windows (PowerShell)
$ ${INSTALL_CMD.windows}`,
      beta: 'Public beta.',
      betaText: "It isn't tested yet with every VPN on every system. If something doesn't work,",
      betaLink: 'tell us on GitHub',
      afterTitle: 'After you download',
      winTitle: 'If Windows stops it with a blue warning (SmartScreen)',
      winSteps: [
        <>Click <strong>More info</strong>.</>,
        <>Click <strong>Run anyway</strong>.</>,
      ],
      winWhy: "It shows up because the file is new and isn't signed with a paid certificate, not because it has a virus.",
      macTitle: "If your Mac says it can't open kongtrol",
      macSteps: [
        <>Close the warning. Don't move it to the Trash.</>,
        <>Open the Apple menu → <strong>System Settings</strong> → <strong>Privacy &amp; Security</strong>.</>,
        <>Scroll to <strong>Security</strong> and click <strong>Open Anyway</strong>. The button shows for about an hour after you tried to open it.</>,
        <>Enter your Mac's password.</>,
      ],
      macCurl: "If you install with the terminal command below (curl), this warning doesn't appear.",
      adminTitle: 'One administrator permission, only once',
      adminWhat: "Setting up Kongtrol installs a helper so it can change routes and DNS. Nothing leaves your computer. It won't ask again.",
      adminSee: [
        <><strong>Windows:</strong> a User Account Control window asks whether to allow the changes. Click <strong>Yes</strong>.</>,
        <><strong>Mac:</strong> it asks for your Mac's password.</>,
        <><strong>Linux:</strong> it asks for your password (sudo).</>,
      ],
      adminTech: 'Technical detail:',
      source: 'SOURCE',
      sourceScript: `# Requires Go 1.27+ — run from Git Bash on Windows
$ git clone ${GITHUB_REPO}
$ cd vpn-kongtrol
$ make build-all-cli
# -> binaries in build/dist/ for all platforms`,
      verifyTitle: 'About unsigned binaries',
      verifyText: 'Kongtrol is open source and does not pay for a code-signing certificate (Windows/macOS require one to skip SmartScreen/Gatekeeper warnings). It is normal for the downloaded binary to trigger a warning — that does not mean it contains malware. Every GitHub release ships a checksums.txt (SHA256) so you can verify the download matches what was actually built. If you would rather not trust a prebuilt binary, review the source and build it yourself (see SOURCE below).',
      verifyScript: `# Verify checksum (Windows PowerShell)
Get-FileHash ${RELEASE_ASSETS['windows-amd64']} -Algorithm SHA256
# compare the output against the matching line in the release's checksums.txt

# Verify checksum (Linux)
sha256sum -c --ignore-missing checksums.txt

# Verify checksum (macOS)
shasum -a 256 -c --ignore-missing checksums.txt`,
    }

  return (
    <section id={DOWNLOAD_SECTION_ID} className="section dl-section">
      <div className="container">
        <div className="section-label cmd-label">{copy.section}</div>
        <h2 className="section-title">{copy.title}</h2>
        <p className="section-sub">{copy.sub}</p>

        {mobile && <SendToComputer lang={lang} className="dl-mobile" />}

        {featured && (
          <div className="dl-featured">
            <p className="dl-featured-tag">{copy.recommended}</p>
            <h3 className="dl-featured-title">{FEATURED_TITLE[featured][lang]}</h3>
            <div className="dl-featured-actions">
              {VARIANTS[featured].map((v, i) => (
                <div key={v.asset} className="dl-featured-item">
                  <a
                    href={releaseAsset(v.asset)}
                    className={i === 0 ? 'btn-download dl-btn-lg' : 'btn-ghost dl-btn-outline'}
                    download
                  >
                    <DownloadIcon size={i === 0 ? 18 : 14} />
                    {featured === 'macos' || i > 0 ? `${copy.download} · ${v.tag[lang]}` : copy.downloadFor[featured]}
                  </a>
                  <FileName asset={v.asset} />
                </div>
              ))}
            </div>
            {featured === 'macos' && <p className="dl-mac-which">{copy.macWhich}</p>}
          </div>
        )}

        <h3 className="dl-others-title">{featured ? copy.others : copy.choose}</h3>
        <div className="dl-grid">
          {ORDER.filter(o => o !== featured).map(o => (
            <div key={o} className="dl-card">
              <h4 className="dl-card-os">{OS_LABELS[o]}</h4>
              <ul className="dl-list">
                {VARIANTS[o].map(v => (
                  <li key={v.asset} className="dl-row">
                    <div className="dl-row-info">
                      <span className="dl-row-name">{v.name[lang]}</span>
                      <FileName asset={v.asset} />
                    </div>
                    {!mobile && (
                      <a href={releaseAsset(v.asset)} className="btn-ghost dl-btn-sm" download>
                        <DownloadIcon size={14} />
                        {copy.download}
                        <span className="sr-only"> · {v.name[lang]}</span>
                      </a>
                    )}
                  </li>
                ))}
              </ul>
              {o === 'macos' && <p className="dl-mac-which">{copy.macWhich}</p>}
            </div>
          ))}
        </div>

        <p className="dl-all">
          <a href={GITHUB_RELEASES} target="_blank" rel="noreferrer">{copy.allReleases}</a>
        </p>
        {/* feature:uninstall feature:update_check */}
        <p className="dl-note">{copy.keepNote}</p>

        <div className="dl-after">
          <h3 className="dl-others-title">{copy.afterTitle}</h3>
          <div className="dl-help">
            <details className="dl-more">
              <summary><span className="disclosure-mark" aria-hidden="true" />{copy.winTitle}</summary>
              <div className="dl-more-body">
                <ol className="dl-steps">{copy.winSteps.map((s, i) => <li key={i}>{s}</li>)}</ol>
                <p className="dl-note">{copy.winWhy}</p>
              </div>
            </details>
            <details className="dl-more">
              <summary><span className="disclosure-mark" aria-hidden="true" />{copy.macTitle}</summary>
              <div className="dl-more-body">
                <ol className="dl-steps">{copy.macSteps.map((s, i) => <li key={i}>{s}</li>)}</ol>
                <p className="dl-note">{copy.macCurl}</p>
              </div>
            </details>
          </div>

          <div className="dl-admin">
            <h4 className="dl-admin-title">{copy.adminTitle}</h4>
            <p className="dl-admin-what">{copy.adminWhat}</p>
            <ul className="dl-admin-see">{copy.adminSee.map((s, i) => <li key={i}>{s}</li>)}</ul>
            <p className="dl-admin-tech">{copy.adminTech} <code>kongtrol service install</code></p>
          </div>
        </div>

        <div className="callout dl-beta">
          <strong>{copy.beta}</strong> {copy.betaText}{' '}
          <a href={`${GITHUB_REPO}/issues`} target="_blank" rel="noreferrer">{copy.betaLink}</a>.
        </div>

        <div className="source-install reveal" style={{ marginTop: 20 }}>
          <div className="source-label">{copy.terminalTitle}</div>
          <div>
            <p className="source-desc" style={{ marginTop: 8, marginBottom: 12 }}>{copy.terminalText}</p>
            <CodeBlock ui={lang} lang="bash">{copy.terminalScript}</CodeBlock>
          </div>
        </div>

        <div className="source-install reveal">
          <div className="source-label">{copy.verifyTitle}</div>
          <div>
            <p className="source-desc" style={{ marginTop: 8, marginBottom: 12 }}>{copy.verifyText}</p>
            <CodeBlock ui={lang} lang="bash">{copy.verifyScript}</CodeBlock>
          </div>
        </div>

        <div className="source-install reveal">
          <div className="source-label">{copy.source}</div>
          <CodeBlock ui={lang} lang="bash">{copy.sourceScript}</CodeBlock>
        </div>
      </div>
    </section>
  )
}
