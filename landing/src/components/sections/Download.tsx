import { Fragment } from 'react'
import { OS, OS_LABELS } from '../../hooks/useOS'
import { useDetectedOS, useIsMobile } from '../../hooks/useVisitor'
import { Lang } from '../../i18n'
import { GITHUB_RELEASES, GITHUB_REPO, MACOS_APP_FILE, MACOS_APP_URL, RELEASE_ASSETS, ReleaseAssetKey, releaseAsset, WINDOWS_INSTALLER_FILE, WINDOWS_INSTALLER_URL } from '../../links'
import CodeBlock from '../ui/CodeBlock'
import DevInstall from './DevInstall'
import { DOWNLOAD_SECTION_ID, DownloadIcon, DownloadNote, OTHER_WAYS_ID, PRIMARY_COPY } from '../ui/downloadCta'
import SendToComputer from '../ui/SendToComputer'
import { pathFor } from '../../site'
import './download.css'

const ORDER: OS[] = ['windows', 'macos', 'linux']

interface FileItem {
  file: string
  href: string
  /** friendly name: what the visitor recognises about their computer */
  name: Record<Lang, string>
  /** short tag for a featured button, when one OS features several files */
  tag: Record<Lang, string>
}

const asset = (key: ReleaseAssetKey) => ({ file: RELEASE_ASSETS[key], href: releaseAsset(key) })

// Every downloadable file, once, per OS (filenames come from links.ts). The
// featured card shows the best option for the detected OS (FEATURED); every
// other file appears exactly once in "Otras formas de instalar" (2026-11-19:
// no file is listed twice). The app (tray + panel + command) comes first on
// every system; the bare binaries are "only the command" (ADR-0077). Windows
// on ARM has no installer yet: its option is the command, listed once.
const FILES: Record<OS, FileItem[]> = {
  windows: [
    { file: WINDOWS_INSTALLER_FILE, href: WINDOWS_INSTALLER_URL, name: { es: 'Instalador · 64-bit', en: 'Installer · 64-bit' }, tag: { es: '64-bit', en: '64-bit' } },
    { ...asset('windows-amd64'), name: { es: 'Solo el comando · 64-bit', en: 'Command only · 64-bit' }, tag: { es: '64-bit', en: '64-bit' } },
    { ...asset('windows-arm64'), name: { es: 'Solo el comando · ARM 64-bit', en: 'Command only · ARM 64-bit' }, tag: { es: 'ARM', en: 'ARM' } },
  ],
  macos: [
    { file: MACOS_APP_FILE, href: MACOS_APP_URL, name: { es: 'App · Apple Silicon e Intel', en: 'App · Apple Silicon and Intel' }, tag: { es: 'App', en: 'App' } },
    { ...asset('macos-arm64'), name: { es: 'Solo el comando · Apple Silicon (M1 o posterior)', en: 'Command only · Apple Silicon (M1 or later)' }, tag: { es: 'Apple Silicon', en: 'Apple Silicon' } },
    { ...asset('macos-amd64'), name: { es: 'Solo el comando · procesador Intel', en: 'Command only · Intel processor' }, tag: { es: 'Intel', en: 'Intel' } },
  ],
  linux: [
    { ...asset('linux-amd64'), name: { es: 'Solo el comando · 64-bit (x86_64)', en: 'Command only · 64-bit (x86_64)' }, tag: { es: '64-bit', en: '64-bit' } },
    { ...asset('linux-arm64'), name: { es: 'Solo el comando · ARM 64-bit (p. ej. Raspberry Pi 4)', en: 'Command only · ARM 64-bit (e.g. Raspberry Pi 4)' }, tag: { es: 'ARM', en: 'ARM' } },
  ],
}

// The featured file per detected OS: the app. Windows: the installer; macOS:
// the universal app (no chip to choose); Linux: no single file fits every
// distribution, so the card shows the one-line install (app + command) and
// points to the .deb/.rpm packages, whose names carry the version.
const FEATURED: Record<OS, string[]> = {
  windows: [WINDOWS_INSTALLER_FILE],
  macos: [MACOS_APP_FILE],
  linux: [],
}

// Featured-card title per OS (macOS covers both chips: the browser can't tell them apart).
const FEATURED_TITLE: Record<OS, Record<Lang, string>> = {
  windows: { es: 'Windows 10/11 · 64-bit', en: 'Windows 10/11 · 64-bit' },
  macos: { es: 'macOS · Apple Silicon o Intel', en: 'macOS · Apple Silicon or Intel' },
  linux: { es: 'Linux', en: 'Linux' },
}

interface Props {
  lang: Lang
}

// The filename is always shown whole; when the card is narrow it may wrap
// only after a "-" (a <wbr> after each one), never inside a word.
function FileName({ file }: { file: string }) {
  const parts = file.split('-')
  return (
    <code className="dl-file">
      {parts.map((p, i) => <Fragment key={i}>{p}{i < parts.length - 1 && <>-<wbr /></>}</Fragment>)}
    </code>
  )
}

export default function Download({ lang }: Props) {
  const detected = useDetectedOS()
  const mobile = useIsMobile()
  // prerender (OS unknown) and phones: no recommendation, every system equal
  const featured = mobile ? null : detected
  const featuredFiles = featured ? FILES[featured].filter(f => FEATURED[featured].includes(f.file)) : []
  // every file not in the featured card, grouped by OS (empty groups dropped)
  const groups = ORDER
    .map(o => [o, FILES[o].filter(f => !featuredFiles.includes(f))] as [OS, FileItem[]])
    .filter(([, files]) => files.length > 0)
  const copy = lang === 'es'
    ? {
      section: '$ make install',
      title: 'Descárgalo gratis.',
      sub: 'Windows, macOS y Linux. Local, gratis y de código abierto. Sin cuenta.',
      download: 'Descargar',
      recommended: 'Tu equipo',
      macWhichShort: '¿Cuál? Menú Apple → Acerca de este Mac.',
      allReleases: 'Ver todos los releases en GitHub',
      keepNote: <>Si desinstalas, tus datos y contraseñas se quedan en tu equipo (en ningún otro lugar), salvo que pidas borrarlos. El aviso de versión nueva es
        opcional y viene apagado. Ver cómo <a href={`${pathFor('guia', 'es')}#update`}>actualizar</a> y{' '}
        <a href={`${pathFor('guia', 'es')}#uninstall`}>desinstalar</a>.</>,
      otherTitle: 'Otras formas de instalar',
      whatTitle: '¿Qué descargo?',
      what: [
        ['La app', 'El ícono junto al reloj, el panel y el comando kongtrol. Es lo que necesita casi todo el mundo: el botón de arriba.'],
        ['Solo el comando', 'Para servidores y scripts, sin ícono ni panel: los archivos «Solo el comando» o el script: en Windows antes escribe $env:KONGTROL_CLI_ONLY=1; en macOS y Linux termina la línea en | sh -s -- --cli-only.'],
        ['El ícono necesita el comando', 'La app siempre lo trae; el comando funciona solo. Tus datos se quedan en tu equipo en los dos casos.'],
      ] as [string, string][],
      linuxPackages: 'También hay paquetes .deb (Ubuntu, Debian) y .rpm (Fedora) con la app y el comando, en',
      linuxPackagesLink: 'la página del release',
      allTitle: 'Todas las descargas',
      scriptTitle: 'Script de instalación',
      terminalText: 'Una línea en la terminal: detecta tu sistema, verifica el SHA256 y lo deja en el PATH.',
      beta: 'Beta pública.',
      betaText: 'Todavía no está probado con cada VPN en cada sistema. Si algo no va bien,',
      betaLink: 'cuéntanoslo en GitHub',
      afterTitle: 'Después de descargar',
      winTitle: 'Si Windows lo frena con un aviso azul (SmartScreen)',
      winSteps: [
        <>Pulsa <strong>Más información</strong>.</>,
        <>Pulsa <strong>Ejecutar de todos modos</strong>.</>,
      ],
      winWhy: 'Sale porque el archivo es nuevo y no está firmado con un certificado de pago, no porque tenga un virus. Si Microsoft Defender lo marca como amenaza (por ejemplo «Trojan:Win32/Bearfoos.A!ml»), es un falso positivo de un archivo sin firma: compara su SHA256 con checksums.txt, restáuralo en Seguridad de Windows › Historial de protección y repórtalo a Microsoft (la guía tiene los pasos).',
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
      recommended: 'Your computer',
      macWhichShort: 'Which one? Apple menu → About This Mac.',
      allReleases: 'View all releases on GitHub',
      keepNote: <>If you uninstall, your data and passwords stay on your computer (nowhere else), unless you ask to delete them. The new version notice is
        optional and off by default. See how to <a href={`${pathFor('guia', 'en')}#update`}>update</a> and{' '}
        <a href={`${pathFor('guia', 'en')}#uninstall`}>uninstall</a>.</>,
      otherTitle: 'Other ways to install',
      whatTitle: 'Which file do I download?',
      what: [
        ['The app', 'The icon next to the clock, the panel and the kongtrol command. It is what almost everyone needs: the button above.'],
        ['Command only', 'For servers and scripts, no icon or panel: the "Command only" files or the script: on Windows type $env:KONGTROL_CLI_ONLY=1 first; on macOS and Linux end the line with | sh -s -- --cli-only.'],
        ['The icon needs the command', 'The app always includes it; the command works alone. Your data stays on your computer either way.'],
      ] as [string, string][],
      linuxPackages: 'There are also .deb (Ubuntu, Debian) and .rpm (Fedora) packages with the app and the command, on',
      linuxPackagesLink: 'the release page',
      allTitle: 'All downloads',
      scriptTitle: 'Install script',
      terminalText: 'One line in the terminal: it detects your system, verifies the SHA256 and puts it on your PATH.',
      beta: 'Public beta.',
      betaText: "It isn't tested yet with every VPN on every system. If something doesn't work,",
      betaLink: 'tell us on GitHub',
      afterTitle: 'After you download',
      winTitle: 'If Windows stops it with a blue warning (SmartScreen)',
      winSteps: [
        <>Click <strong>More info</strong>.</>,
        <>Click <strong>Run anyway</strong>.</>,
      ],
      winWhy: "It shows up because the file is new and isn't signed with a paid certificate, not because it has a virus. If Microsoft Defender flags it as a threat (for example \"Trojan:Win32/Bearfoos.A!ml\"), it is a false positive on an unsigned file: compare its SHA256 with checksums.txt, restore it in Windows Security › Protection history and report it to Microsoft (the guide has the steps).",
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
      {/* ADR-0070: gold only for the call to action (heading + the visitor's
          file); it ends on a contour edge and everything after it is
          reference on the Bone page. Two contour lines echo the edge. */}
      <div className="dl-band tone-signal edge edge-b-ring">
        <div className="container">
          {/* laptops: heading left, the recommended file right (one row) */}
          <div className="dl-lead">
          <div className="dl-lead-text">
            <div className="section-label cmd-label">{copy.section}</div>
            <h2 className="section-title">{copy.title}</h2>
            <p className="section-sub">{copy.sub}</p>
          </div>

          {mobile && <SendToComputer lang={lang} className="dl-mobile" />}

          {featured && (
            // Not a boxed card: a terminal comment naming the detected system,
            // the band's one filled button and the file line, set off from
            // the heading by a hairline (owner feedback 2026-11-19: the boxed
            // "eyebrow + title" card read as generic).
            <div className="dl-featured">
              <div className="dl-featured-head">
                <h3 className="dl-featured-title">
                  <span className="dl-featured-hash" aria-hidden="true"># </span>
                  {copy.recommended}: <span className="dl-featured-os">{FEATURED_TITLE[featured][lang]}</span>
                </h3>
              </div>
              <div className="dl-featured-body">
                <div className="dl-featured-actions">
                  {featuredFiles.map(v => (
                    <div key={v.file} className="dl-featured-item">
                      <a href={v.href} className="btn-download dl-btn-lg" download>
                        <DownloadIcon size={18} />
                        {PRIMARY_COPY[lang].forOS[featured]}
                      </a>
                      <FileName file={v.file} />
                    </div>
                  ))}
                </div>
                {featured === 'linux' && (
                  <>
                    <DevInstall lang={lang} />
                    <p className="dl-mac-which">
                      {copy.linuxPackages}{' '}
                      <a href={GITHUB_RELEASES} target="_blank" rel="noreferrer">{copy.linuxPackagesLink}</a>.
                    </p>
                  </>
                )}
                <DownloadNote lang={lang} os={featured} page="home" className="dl-featured-note" />
              </div>
            </div>
          )}
          </div>
        </div>
      </div>
      <svg className="dl-echo" viewBox="0 0 1440 100" preserveAspectRatio="none" aria-hidden="true" focusable="false">
        <path pathLength={1} d="M0 12 C380 98 1060 98 1440 12" />
        <path pathLength={1} d="M0 12 C380 98 1060 98 1440 12" />
      </svg>
      <div className="container dl-rest">

        {/* feature:distribution — every other file and the one-line script,
            each exactly once */}
        {/* ADR-0077: which file is for whom, in one glance */}
        <div className="dl-what">
          <h3 className="dl-others-title">{copy.whatTitle}</h3>
          <dl className="dl-what-list">
            {copy.what.map(([term, desc]) => (
              <div key={term}>
                <dt>{term}</dt>
                <dd>{desc}</dd>
              </div>
            ))}
          </dl>
        </div>

        <div id={OTHER_WAYS_ID} className="dl-other">
          <h3 className="dl-others-title">{featured ? copy.otherTitle : copy.allTitle}</h3>
          <div className="dl-ways">
            {groups.map(([o, files]) => (
              <div key={o} className="dl-card">
                <h4 className="dl-card-os">{OS_LABELS[o]}</h4>
                <ul className="dl-list">
                  {files.map(v => (
                    <li key={v.file} className="dl-row">
                      <div className="dl-row-info">
                        <span className="dl-row-name">{v.name[lang]}</span>
                        <FileName file={v.file} />
                      </div>
                      {!mobile && (
                        <a href={v.href} className="btn-ghost dl-btn-icon" download title={`${copy.download} · ${v.file}`}>
                          <DownloadIcon size={16} />
                          <span className="sr-only">{copy.download} · {OS_LABELS[o]} · {v.name[lang]}</span>
                        </a>
                      )}
                    </li>
                  ))}
                </ul>
                {o === 'macos' && <p className="dl-card-note">{copy.macWhichShort}</p>}
              </div>
            ))}
            <div className="dl-card dl-script">
              <h4 className="dl-card-os">{copy.scriptTitle}</h4>
              <p className="dl-card-note">{copy.terminalText}</p>
              <DevInstall lang={lang} />
            </div>
          </div>
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
