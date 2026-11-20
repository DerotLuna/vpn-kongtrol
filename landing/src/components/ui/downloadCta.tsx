import { OS } from '../../hooks/useOS'
import { Lang } from '../../i18n'
import { MACOS_APP_URL, WINDOWS_INSTALLER_URL } from '../../links'
import { PageId, pathFor } from '../../site'

// Shared by every "download" entry point (hero, header, Para todos, the
// download section, footer). The header and "Para todos" buttons scroll to
// the download section (ADR-0027); the primary buttons (hero, download
// section, footer: PrimaryDownload below) download the Windows installer
// directly and send macOS / Linux visitors to the section to pick their file.
export const DOWNLOAD_SECTION_ID = 'install'

export function downloadHref(page: PageId, lang: Lang): string {
  return page === 'home' ? `#${DOWNLOAD_SECTION_ID}` : `${pathFor('home', lang)}#${DOWNLOAD_SECTION_ID}`
}

export const CTA_COPY: Record<Lang, {
  generic: string
  forOS: Record<OS, string>
  github: string
  githubLabel: string
  navDownload: string
  mobileNotice: string
  send: string
  copy: string
  copied: string
  copyFailed: string
  shareTitle: string
}> = {
  es: {
    generic: 'Ver descargas',
    forOS: { windows: 'Ver descargas para Windows', macos: 'Ver descargas para Mac', linux: 'Ver descargas para Linux' },
    github: 'Ver en GitHub',
    githubLabel: 'Kongtrol en GitHub',
    navDownload: 'Descargas',
    mobileNotice: 'Estás en un celular: Kongtrol se instala en tu computador.',
    send: 'Enviar a mi computador',
    copy: 'Copiar enlace',
    copied: 'Enlace copiado. Ábrelo en tu computador.',
    copyFailed: 'No se pudo copiar. Copia la dirección de esta página.',
    shareTitle: 'Todas tus VPN bajo Kongtrol',
  },
  en: {
    generic: 'See downloads',
    forOS: { windows: 'See downloads for Windows', macos: 'See downloads for Mac', linux: 'See downloads for Linux' },
    github: 'View on GitHub',
    githubLabel: 'Kongtrol on GitHub',
    navDownload: 'Downloads',
    mobileNotice: "You're on a phone: Kongtrol installs on your computer.",
    send: 'Send to my computer',
    copy: 'Copy link',
    copied: 'Link copied. Open it on your computer.',
    copyFailed: "Couldn't copy. Copy this page's address.",
    shareTitle: 'All your VPNs under Kongtrol',
  },
}

/** "Ver descargas para Windows", or the generic label while the OS is unknown (prerender). */
export const downloadLabel = (lang: Lang, os: OS | null) =>
  os ? CTA_COPY[lang].forOS[os] : CTA_COPY[lang].generic

/** The footer's id: the header hides while it is on screen (Nav.tsx). */
export const FOOTER_ID = 'site-footer'
/** "Otras formas de instalar" in the download section: standalone command + script. */
export const OTHER_WAYS_ID = 'install-other'

// feature:distribution
// The primary download (hero, download section, footer — one source). On
// Windows it downloads the installer (links.ts WINDOWS_INSTALLER_URL). macOS
// (Apple Silicon or Intel: the browser can't tell) and Linux keep what works
// today: the download section, where the visitor picks the file. Before
// hydration the OS is unknown, so it points at the download section too.
export const PRIMARY_COPY: Record<Lang, {
  download: string
  forOS: Record<OS, string>
  note: Record<OS, string>
  noteGeneric: string
  otherWays: string
  guide: string
}> = {
  es: {
    download: 'Descargar',
    forOS: { windows: 'Descargar para Windows', macos: 'Descargar para Mac', linux: 'Descargar para Linux' },
    note: {
      windows: 'Instalador · se actualiza sin perder nada',
      macos: 'La app (ícono + panel + comando) · Apple Silicon e Intel',
      linux: 'La app y el comando · se instala con una línea',
    },
    noteGeneric: 'Windows, macOS y Linux · gratis, sin cuenta',
    otherWays: '¿Usas la terminal? Otras formas de instalar',
    guide: 'Guía de instalación',
  },
  en: {
    download: 'Download',
    forOS: { windows: 'Download for Windows', macos: 'Download for Mac', linux: 'Download for Linux' },
    note: {
      windows: 'Installer · updates without losing anything',
      macos: 'The app (icon + panel + command) · Apple Silicon and Intel',
      linux: 'The app and the command · installs with one line',
    },
    noteGeneric: 'Windows, macOS and Linux · free, no account',
    otherWays: 'Use the terminal? Other ways to install',
    guide: 'Installation guide',
  },
}

/** Where the primary button goes: the app's file on Windows and macOS, else the download section (ADR-0077). */
export function primaryDownloadHref(os: OS | null, page: PageId, lang: Lang): string {
  if (os === 'windows') return WINDOWS_INSTALLER_URL
  if (os === 'macos') return MACOS_APP_URL
  return downloadHref(page, lang)
}

export function otherWaysHref(page: PageId, lang: Lang): string {
  return page === 'home' ? `#${OTHER_WAYS_ID}` : `${pathFor('home', lang)}#${OTHER_WAYS_ID}`
}

/** The small line under a primary download button: what it is + other ways. */
export function DownloadNote({ lang, os, page, className = '' }: { lang: Lang; os: OS | null; page: PageId; className?: string }) {
  const t = PRIMARY_COPY[lang]
  return (
    <p className={`pd-note ${className}`}>
      <span>{os ? t.note[os] : t.noteGeneric}</span>
      <a href={otherWaysHref(page, lang)}>{t.otherWays}</a>
    </p>
  )
}

/** Button + note: the footer's call to action (the hero styles its own button). */
export function PrimaryDownload({ lang, os, page }: { lang: Lang; os: OS | null; page: PageId }) {
  const t = PRIMARY_COPY[lang]
  const direct = os === 'windows' || os === 'macos'
  return (
    <div className="pd">
      <a href={primaryDownloadHref(os, page, lang)} className="btn-download pd-btn" download={direct ? '' : undefined}>
        <DownloadIcon size={16} />
        {os ? t.forOS[os] : t.download}
      </a>
      <DownloadNote lang={lang} os={os} page={page} />
    </div>
  )
}

export function DownloadIcon({ size = 16 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
      <path d="M8 1a.5.5 0 0 1 .5.5v7.793l2.646-2.647a.5.5 0 0 1 .708.708l-3.5 3.5a.5.5 0 0 1-.708 0l-3.5-3.5a.5.5 0 1 1 .708-.708L7.5 9.293V1.5A.5.5 0 0 1 8 1ZM2.5 13a.5.5 0 0 0 0 1h11a.5.5 0 0 0 0-1h-11Z"/>
    </svg>
  )
}

export function GitHubIcon({ size = 16 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
      <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27s1.36.09 2 .27c1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8Z"/>
    </svg>
  )
}

export function SendIcon({ size = 16 }: { size?: number }) {
  // phone → screen: a laptop outline with an arrow coming in
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M4 16V6a1 1 0 0 1 1-1h14a1 1 0 0 1 1 1v10" />
      <path d="M2 19h20" />
      <path d="M12 8v5m-2.5-2.5L12 13l2.5-2.5" />
    </svg>
  )
}
